// (C) Copyright 2012-2013, Rolf Veen.
// See the LICENCE file.

package ogdl

import ( 
  "strings"
)

// Path is used to parse OGDL path into a Graph object.
//
//   a.b(c, d).e[1].b{5}
//
//   a
//   b
//   !g
//     c
//     d
//   e
//   !i
//     1
//   v
//   !s
//     5
//
type Path struct {
    // Annonymous reference to Parser,
    // as Path is an extension of it.
    Parser
}

func NewPath(s string) *Path {
    // BUG(): ind[] should be var length
    return &Path{ Parser{ strings.NewReader(s), NewEventHandlerG(), make([]int,32), [2]int{0,0}, 0, 0, 1, 0 } }
}

/*   
     Path ::= Elem (Separator Elem)*
     
     Elem ::= Scalar | Group | Index | Selector
     Separator ::= Dot
     
     Dot optional before Group, Index, Selector
*/


func (p *Path) Parse() {

    expectSep := false
    
    s := ""
    
    p.ev.Event("")
    
    for {
    
        // If expectSep is true, we expext a dot, index, group or selector
        if expectSep {
            if p.NextByteIs('.') {
                expectSep = false
                continue;
            }
            expectSep = false
            
        } else {
        
            p.NextByteIs('.')
            
            s = p.Scalar()

            if len(s) != 0 {               
                p.ev.Event(s)
                expectSep = true
                continue
           }
        }
   
        s = p.Index()
        if len(s) != 0 {
            p.ev.Event("!i")
            p.ev.Inc()
            p.ev.Event(s)
            p.ev.Dec()
            continue
        }
        
        s = p.Selector()
        if len(s) != 0 {
            p.ev.Event("!s")
            if len(s)>1 || s[0] != ' ' {
                p.ev.Inc()
                p.ev.Event(s)
                p.ev.Dec()
            }
            continue
        }
        
        b := p.Group()
        if b {
            continue
        }
        
        break
    }
    
    return
}


func (p *Path) Index() string {
    
    if !p.NextByteIs('[') {
        return ""
    }
    
    p.Space()
    s := p.String()
    p.Space()
    
    if !p.NextByteIs(']') {
        return ""
    }
    return s
}

func (p *Path) Selector() string {
    
    if !p.NextByteIs('{') {
        return ""
    }
    
    p.Space()
    s := p.String()
    p.Space()
    
    if !p.NextByteIs('}') {
        return ""
    }
    
    // Return one space to indicate an empty selector
    if len(s)==0 {
        return " "
    }
    return s
}

/* Group

   Group ::= '(' Space? Sequence?  Space? ')'
*/

func (p *Path) Group() bool {

    if !p.NextByteIs('(') {
        return false
    }
    
    i := p.ev.Level()
    
    p.ev.Event("!g")
    p.ev.Inc()

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

/* ---------------------------------------------
   (4) Character types
   --------------------------------------------- */
   
func isTextByte (c int) bool {
    if c>32 && c!='(' && c!=')' && c!=',' && c!='.' && c!='[' && c!=']' && c!='{' && c!='}' {
        return true
    }
    return false
}





