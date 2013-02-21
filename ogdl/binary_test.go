package ogdl

import ( 
  "testing"
)

func TestBin2G(t *testing.T) {
    
    g := NewGraph("hola")
    g.Add("world")
    
    b, _ := g.Binary()
    
    println("Len ",len(b))
    
    if len(b)!=17 {
        t.Fatal("Binary lenght should be 17 and is ",len(b))
    }
    
    p := NewByteBinParser(b)
    
    g = p.Graph()
    
    println(g.String())
}