package ogdl

import ( 
  "testing"
)

func TestPath1(t *testing.T) {
    
    p := NewPath("a[3][6].{2}(b c).[1]")
    
    p.Parse()

    g := p.Graph()
    
    print(g.String())
}

func TestPath2(t *testing.T) {
    
    p := NewPath("for(i,a)")
    
    p.Parse()

    g := p.Graph()
    
    print(g.String())
}

func TestPath3(t *testing.T) {
    
    p := NewPath("$a.$x($b + 2, $c)")
    
    p.Parse()

    g := p.Graph()
    
    print(g.String())
}