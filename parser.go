// (C) Copyright 2012-2013, Rolf Veen.
// See the LICENCE file.

package ogdl

import ( 
  "io"
  "strings"
  "bytes"
  "io/ioutil"
  "bufio"
  "errors"
)

/* 
Parser is used to parse textual OGDL streams into Graph objects.

There are several types of functions here:
   
   - Read and Unread: elementary character handling.
   - Character classifiers.
   - Elementary productions that return a bool.
   - Productions that return a string (or nil).
   - Productions that produce an event.
   
The OGDL parser doesn't need to know about Unicode. The character 
classification relies on values < 127, thus in the ASCII range, 
which is also part of Unicode.

Note: On the other hand it would be stupid not to recognize for example
Unicode quotation marks if we know that we have UTF-8. But when do we
know for sure?
   
BUG(): Level 2 (graphs) not implemented
*/
type Parser struct {
    // The input (byte) stream
    in  io.ByteReader

    // The output (event) stream
    ev  EventHandler
    
    // ind holds indentation at different levels, that is,
    // the number of spaces at each level.
    ind []int
    
    // last holds the 2 last characters read. 
    // We need 2 characters of look-ahead (for Block()).
    last [2]int
    
    // unread index
    lastn int
    
    // the number of characters after a NL was found
    // (Used in Quoted)
    lastnl int
    
    // line keeps track of the line number
    line int
    
    // save spaces at end of block
    spaces int
}

func NewStringParser(s string) *Parser {
    return &Parser{ strings.NewReader(s), NewEventHandlerG(), make([]int,32), [2]int{0,0}, 0, 0, 1, 0 }
}

func NewParser(r io.Reader) *Parser {
    return &Parser{ bufio.NewReader(r), NewEventHandlerG(), make([]int,32), [2]int{0,0}, 0, 0, 1, 0 }
}

func NewFileParser(s string) *Parser {
    b, err := ioutil.ReadFile(s)
    if err != nil || len(b)==0 {
        return nil
    }

    buf := bytes.NewBuffer(b)
    return &Parser{ buf, NewEventHandlerG(), make([]int,32), [2]int{0,0}, 0, 0, 1, 0 }
}

func (p *Parser) Graph() *Graph {
    return p.ev.Graph()
}

/* Parse is the main function for parsing OGDL text.

   An OGDL stream is a sequence of lines (a block
   of text or a quoted string can span multiple lines
   but is still a single node)
   
     Graph ::= Line* End
*/
func (p *Parser) Parse() (errx error) {
    
    defer func() {
        if r := recover(); r != nil {
            errx = errors.New(r.(string))
        }
    }()
    
    for {
        more, err := p.Line()
        if err!=nil {
            return err
        }
        if (!more) {
            break
        }
    }
    p.End()
    
    return nil
}

/* Line processes an OGDL line or a multiline scalar.
   
  - A Line is composed of scalars and groups.
  - A Scalar is a Quoted or a String.
  - A Group is a sequence of Scalars enclosed in parenthesis
  - Scalars can be separated by commas or by space
  - The last element of a line can be a Comment, or a Block
   
 The indentation of the line and the Scalar sequences and Groups on it define
 the tree structure characteristic of OGDL level 1.
   
     Line ::= Space(n) Sequence? ((Comment? Break)|Block)?
              
 Anything other than one Scalar before a Block should be an syntax error.
 Anything after a closing ')' that is not a comment is a syntax error, thus
 only one Group per line is allowed. That is because it would be difficult to
 define the origin of the edges pointing to what comes after a Group.
    
 Indentation rules:
    
    a           -> level 0
      b         -> level 1
      c         -> level 1
        d       -> level 2
       e        -> level 2
     f          -> level 1
    
*/
func (p *Parser) Line() (bool, error) {
    
    sp, n := p.Space()

    // if a line begins with non-uniform space, throw a syntax error. 
    if sp && n==0 {
        errors.New("OGDL syntax error: non-uniform space")
    }
    
    if (p.End()) {
        return false, nil
    }
    
    // We should not have a Comma here, but lets ignore it.
    if p.NextByteIs(',') {
        p.Space() // Eat eventual space characters
    }
    
    /* indentation TO level
    
       The number of spaces (indentation) for each level is stored in
       p.ind[level]
    */
    
    l := 0
    
    if n!=0 {
        l = 1
        for {
            if p.ind[l] == 0 {
                break;
            }
            if p.ind[l]>=n {
                break;
            }
            l++
        }
    }
    
    p.ind[l] = n
    p.ev.SetLevel(l)   
    
//println("line: ",n,", ",l)    

    // Now we can expect a sequence of scalars, groups, and finally
    // a block or comment.
     
    //wasGroup := false
     
    for {
		if p.Group() {
		
        } else if p.Comment() {
            p.Space()
            p.Break()
            break
        } else {
            s := p.Block()
              
            if len(s)>0 {
                p.ev.Event(s)
                p.Break()
                break
            } else {
                s := p.Scalar()
                if len(s)>0 {
                    p.ev.Event(s)      
                } else {
                    p.Break()
                    break
                }
            }
        }   
    
        p.Space()
 
        co := p.NextByteIs(',')
        
        if (co) {
            p.Space()
            p.ev.SetLevel(l)
        } else {
            p.ev.Inc()
        }
        
    }
    
    // Restore the level to that at the beginning of the line.
    p.ev.SetLevel(l)
    
    return true, nil
}

/* Sequence


     Sequence ::= (Scalar|Group) (Space? (Comma? Space?) (Scalar|Group))*
     [!] with the requirement that after a group a comma is required if there are more elements.

   Examples:
     a b c
     a b,c
     a(b,c)
     (a b,c)
     (b,c),(d,e) <-- This can be handled
     a (b c) d   <-- This is an error
     
     
   This method returns two booleans: if there has been a sequence, and if the last element was a Group
*/

func (p *Parser) Sequence() (bool, bool, error) {

    i := p.ev.Level();
    
    wasGroup := false
    n := 0
    
    for {
        if p.Group() {
            wasGroup = true
        } else {
            s := p.Scalar()
              
            if len(s) == 0 {
                return n>0, wasGroup, nil
            }          
            wasGroup = false
            p.ev.Event(s)
        }

        n++
           
        // We first eat spaces
        
        p.Space()
 
        co := p.NextByteIs(',')
        
        if (co) {
            p.Space()
            p.ev.SetLevel(i)
        } else {
            p.ev.Inc()
        }
    }
    panic("Should not reach this point")
}

/* Group

   Group ::= '(' Space? Sequence?  Space? ')'
*/

func (p *Parser) Group() bool {

    if !p.NextByteIs('(') {
        return false
    }
    
    i := p.ev.Level()
    
    p.Space()

    p.Sequence()
    
    p.Space()
    
    if !p.NextByteIs(')') {
        return false
    }
    
    /* Level before and after a group is the same */
    p.ev.SetLevel(i)
    return true
}

/* Scalar is either a String or a Quoted */
func (p *Parser) Scalar() string {
    s := p.Quoted()   
    if len(s)!=0 {
        return s
    }
    return p.String()
}

/* Comment

   Anything from # up to the end of the line.
   
   BUG(): Special cases: #?, #{
*/
func (p *Parser) Comment() bool {
    c := p.Read()
    if c=='#' {
        for {
            c = p.Read()
            if c==13 {
                c = p.Read()
                if (c!=10) {
                    p.Unread()
                }
                break
            } 
            if c==10 {
                break;
            }
        }
        return true
    }
    p.Unread()
    return false
}

// String is a concatenation of characters that are > 0x20
// and are not '(', ')', ',', and do not begin with '#'.
//   
// NOTE: '#' is allowed inside a string. For '#' to start
// a comment it must be preceeded by break or space, or come
// after a closing ')'.
func (p *Parser) String() string {
    
    c := p.Read()
       
    if !isTextChar(c) || c == '#' {
        p.Unread()
        return ""
    }
    
    buffer := &bytes.Buffer{}
    buffer.WriteByte(byte(c))
    
    for {   
        c := p.Read()
        if !isTextChar(c) {
     
            p.Unread()
            break
        }
        buffer.WriteByte(byte(c))
    }
   
    // We have at least one character
    return buffer.String()
}

// Quoted string.
//
// a "quoted string"
//   "text with
//   some
// newlines"
//
func (p *Parser) Quoted() string {
    
    cs := p.Read()
    if cs != '"' && cs != '\'' {
        p.Unread()
        return ""
    }
    
    buffer := &bytes.Buffer{}
    
    // p.lastnl is the indentation of this quoted string
    lnl := p.lastnl

    /* Handle \", \', and spaces after NL */
    for {   
        c := p.Read()
        if c==cs {
            break
        }
        
        buffer.WriteByte(byte(c))
        
        if c==10 {
            _, n := p.Space()
            // There are n spaces. Skip lnl spaces and add rest.
            for ;n-lnl>0;n-- {
                buffer.WriteByte(32)
            }
        } else if c == '\\' {
            c = p.Read()
            if c != '"' && c != '\'' {
                buffer.WriteByte('\\')
            }
            buffer.WriteByte(byte(c))
        }
    }
    
    // We have at least one character
    return buffer.String()
}

// Block ::= '\\' NL LINES_OF_TEXT
//
func (p *Parser) Block() string {
    
    var c int
    
    c = p.Read()
    if c != '\\' {
        p.Unread()
        return ""
    }
    
    c = p.Read()
    if c != 10 && c!=13 {
        p.Unread()
        p.Unread()
        return ""
    }
    
    // Read lines until indentation is >= to upper level.
    i := p.ind[p.ev.Level()-1]

    u, ns := p.Space();

    if u && ns== 0 {
        println("Non uniform space at beginning of block at line",p.line)
        panic("")
    }
    
    buffer := &bytes.Buffer{}
    
    j := ns;
    
    for {
        if j<=i {
            p.spaces = j /// XXX: unread spaces!
            break
        }
        
        // Adjust indentation if less that initial
        if (j<ns) {
            ns = j;  
        }
        
        // Read bytes until end of line
        for {
            c = p.Read();
            
            buffer.WriteByte(byte(c))
            if c==13 {
                continue;
            }
            
            if c==10 || p.End() {
                break;
            }
        }
        
        _, j = p.Space()
    }
    
    // Remove trailing NL
    if c == 10 {
        if buffer.Len()>0 {
            buffer.Truncate( buffer.Len() -1)
        }
    }
    
    return buffer.String()
}


// Break is NL, CR or CR+NL
//
func (p *Parser) Break() bool {
    c := p.Read()
    if c==13 {
        c = p.Read()
        if (c!=10) {
            p.Unread()
        }
        return true
    }
    if c==10 {
        return true
    }
    p.Unread()
    return false
}

// Space is (0x20|0x09)+. It returns a boolean indicating
// if space has been found, and an integer indicating
// how many spaces, iff uniform (either all 0x20 or 0x09)
//
func (p *Parser) Space() (bool, int) {
    
    // The Block() production eats to many spaces trying to
    // detect the end of it. They are saved in p.spaces.
    if p.spaces>0 {
        i := p.spaces
        p.spaces = 0
        return true, i  
    }
    
    c := p.Read()
    if c!=32 && c!=9 {
        p.Unread()
        return false, 0
    }
    
    n := 1
    /* We keep 'c' to tell us what spaces will count as uniform. */
    
    for {   
        cs := p.Read()
        if cs!=32 && cs!=9 {
            p.Unread()
            break
        }
        if (n!=0 && cs == c) {
            n++
        } else {
            n=0
        }
    }
    
    return true, n
}

func (p *Parser) End() bool {
    c := p.Read()
    if c<32 && c!=9 && c!=10 && c!=13 {
        return true
    }
    p.Unread()
    return false
}

// NextByteIs tests if the next character in the 
// stream is the one given as parameter, in which
// case it is consumed.
//
func (p *Parser) NextByteIs(c int) bool {
    ch := p.Read()
    if ch==c {
        return true
    }
    p.Unread()
    return false
}

func (p *Parser) Newline() bool {
    c := p.Read()
    if c=='\r' {
        c=p.Read()
    }
    
    if (c=='\n') {
        return true
    }
    
    p.Unread()
    return false
}

/* ---------------------------------------------
   (4) Character types
   --------------------------------------------- */
   
func isTextChar(c int) bool {
    if c>32 && c!='(' && c!=')' && c!=',' {
        return true
    }
    return false
}

/* ---------------------------------------------
   (5) Elementary byte handling
   
   OGDL doesn't need to look ahead further than 2 chars.
   --------------------------------------------- */

// Read reads the next byte out of the stream.
//
func (p *Parser) Read() int {

    var c int

    if p.lastn>0 {
        p.lastn--
//println("reread ",p.last[p.lastn], "lastn=", p.lastn)        
        c = p.last[p.lastn]
    } else {
        i, _ := p.in.ReadByte()
        c = int(i)
        p.last[1] = p.last[0]
        p.last[0] = c;
    }
    
    if (c==10) {
        p.lastnl = 0
        p.line++
    } else {
        p.lastnl++
    } 
//println("read: ",c)    
    return c
}

// Unread puts the last readed character back into the stream.
// Up to two consecutive Unread()'s can be issued.
//
func (p *Parser) Unread() {
    p.lastn++;
    p.lastnl--;
//println("unread")       
}





