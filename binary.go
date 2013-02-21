// (C) Copyright 2012-2013, Rolf Veen.
// See the LICENCE file.

package ogdl

import ( 
   "bufio"
   "io"
   "bytes"
   "io/ioutil"  
//   "fmt"
)

type BinParser struct {
    r *bufio.Reader
    ix int
    last int
    // Used to count bytes readed.
    N  int
}

/* ---------------------------------------------
   Constructors
   --------------------------------------------- */

func NewByteBinParser(s []byte) *BinParser {
    return &BinParser{bufio.NewReader(bytes.NewReader(s)),0,0,0}
}

func NewFileBinParser(file string) *BinParser {

    // Read the entire file into memory
    b, err := ioutil.ReadFile(file)
    if err != nil || len(b)==0 {
        return nil
    }
    
    return NewByteBinParser(b)
}

func NewBinParser(r io.Reader) *BinParser {
    return &BinParser{bufio.NewReader(r),0,0,0}
}

/* ---------------------------------------------
   Graph to Binary
   Binary to Graph
   --------------------------------------------- */
   
func (g * Graph) Binary() ([]byte, int) {
    
    if g == nil {
        return nil, -1
    }
    
    // Header
    buf := make([]byte,3)
    buf[0] = 1;
    buf[1] = 'G';
    buf[2] = 0;
    
    buf = g.bin(1,buf)
    
    // Ending null
    buf = append(buf,0)
    
    return buf, 0
}

func (g *Graph) bin(level int, buf []byte) []byte {

    // Skip empty nodes
    if len(g.Name)!=0 {       
        buf = append(buf,NewVarInt(level)...)
        buf = append(buf,[]byte(g.Name)...)
        buf = append(buf,0)
        level++
    }
//fmt.Printf("bin g.Name: %v\n",g.Name)        
    
    for _,node := range g.Nodes {
        buf = node.bin(level,buf)
    }
    
    return buf
}

func (p * BinParser) Graph() * Graph {
    
    if !p.Header() {
        return nil
    }
    
    ev := NewEventHandlerG()

    for {
        // BUG(): blobs not handled
        lev, _, b := p.Line(true)
        if lev == 0 {
            break
        }
        ev.EventL(string(b),lev)
    }
    return ev.Graph()
}
   
/* ---------------------------------------------
   Productions:
   
   - Header
   - VarInt
   - Line
   
   A binary OGDL file is a sequence of Lines:
   
     Binary_OGDL := Line+
   
   where the first Line is a Header.
   --------------------------------------------- */

func (p *BinParser) Header() bool {
  
    if p.Read() != 1 {
        return false;
    }
    if p.Read() != 'G' {
        return false;
    }
    if p.Read() != 0 {
        return false;
    }
    return true;
}

func NewVarInt(i int) []byte {

    if i<0x80 {
        b := make([]byte,1)
        b[0] = byte(i)
        return b
    }
// ...
    return nil
}

func (p *BinParser) VarInt() int {
    
    b0 := p.Read();
    
    if b0 < 0x80 {
        return b0
    }
    
    if b0 < 0xc0 {
		b1 := p.Read();
		return (b0 & 0x3f) << 8 | b1;
	}
    
    if b0 < 0xe0 {
		b1 := p.Read();
		b2 := p.Read();
		return ((b0 & 0x1f) << 16) | (b1 << 8) | b2;
	}
	
	if b0 < 0xf0 {
		b1 := p.Read();
		b2 := p.Read();
		b3 := p.Read();
		return ((b0 & 0x0f) << 24) | (b1 << 16) | (b2 << 8) | b3;
	}
		
    return -1; 
}

func (p *BinParser) Line(write bool) (/* level */ int, /* blob*/ bool, []byte) {
	
	// Read int 
	level := p.VarInt()
	if level<1 {
	    return 0, false, nil
	}
	
	// Binary? 
	n := p.Read();
	if n==1 {
	    // Read bytes...
	    return level, true, nil;   
	} else {
	    p.Unread()
	}
	
	// Read bytes until 0
	
	buf := bytes.Buffer{}
	
	for {
	    c := p.Read();
	    if c == 0 {
	        return level, false, buf.Bytes()
	    } 
	    if (write) {
	        buf.WriteByte(byte(c))
	    }
	}
	
	// BUG(): Error
	return 0, false, nil;
}

/* ---------------------------------------------
   Elementary byte handling
   --------------------------------------------- */

func (p *BinParser) Read() int {

    i, err := p.r.ReadByte()
  
    c := int(i)    
    if err == io.EOF {
        c = -1
    } else {
        p.N++
    }
    
    p.last = c
    
//fmt.Printf("[%d]\n",int(i))
    return c
}

// Unread puts the last character readed back into the stream.
func (p *BinParser) Unread() {
    if p.last > 0 {
        p.N--
        p.r.UnreadByte()
    }
}
