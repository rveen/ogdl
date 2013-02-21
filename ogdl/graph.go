// (C) Copyright 2012-2013, Rolf Veen.
// See the LICENCE file.

package ogdl

import ( 
  "bytes"
  "strings"
  "strconv"
)

type Graph struct {
    Name string
    Nodes []*Graph
}

// NullGraph returns a pointer to a 'null' Graph, a so called transparent node.
// A transparent node is a node whose Name has zero length.
//
// Not simply called Null() because the ogdl package has more objects.
//
func NullGraph() *Graph {
    return &Graph { "", nil }
}

func NewGraph(s string) *Graph {
    return &Graph { s, nil }
}

// Add adds a subnode to the current node.
//
func (g *Graph) Add (s string) *Graph {

    if g == nil {
        return nil
    }
    n := Graph { s, nil }
    g.Nodes = append(g.Nodes,&n)
    return &n
}

// AddGraph adds a Graph to this node.
//
// An eventual transparent root will not
// be added (it will be bypassed).
//
func (g *Graph) AddGraph (node *Graph) *Graph {
    if g == nil || node == nil {
        return nil	
    }
    if len(node.Name) == 0 {
        for _, node = range g.Nodes {
            g.Nodes = append(g.Nodes, node)
        }
    } else { 
        g.Nodes = append(g.Nodes,node)
    }
    return node
}

// GetNode returns the node with a direct edge to this graph, 
// that has the specified name.
//
func (g *Graph) GetNode (s string) *Graph {
    for _, node := range g.Nodes {
        if (s == node.Name) {
	        return node
	    }
    }
    return nil
}

func (g *Graph) GetNodeI (s string) (*Graph, int) {
    for i, node := range g.Nodes {
        if (s == node.Name) {
	        return node, i
	    }
    }
    return nil, -1
}

// Node gets the specified node by name,
// or it creates it.
//
func (g *Graph) Node (s string) *Graph {
    for _, node := range g.Nodes {
        if (s == node.Name) {
	        return node
	    }
    }
    return g.Add(s)
}

func (g *Graph) Remove (s string) {
    _, i := g.GetNodeI(s)
    if i == -1 {
        return
    }
    g.Nodes = append(g.Nodes[:i],g.Nodes[i+1:]...)
}

func (g *Graph) GetIndex (i int) *Graph {
    
    if i>=len(g.Nodes) || i<0 {
        return nil
    }
    
    return g.Nodes[i]
}

// Query(path) produces a new graph object with
// the result, in contrast to Get(path), which returns
// a node within the given graph.
//
// OGDL 'Search Path' adds:
//
// [*], [**]
// {} All elements with that name
// [/regex/]
//
func (g *Graph) Query (s string) *Graph {
    return nil
}

// Get(path): this is it.
// 
// OGDL Path:
// elements are separated by '.' or [] or {}
// index := [N]
// selector := {N}
// tokens can be quoted
//
func (g *Graph) Get (s string) *Graph {

    p := NewPath(s)
    p.Parse()
    path := p.Graph()
    strip := true
    
    if path == nil {
        return nil
    }
     
    node := g
    
    for _, elem := range path.Nodes {
    
        if len(elem.Name)==0 {
            continue; // bypass transparent nodes
        }
    
        c := elem.Name[0]
        
        if c == '!' {
            strip = false;
            c = elem.Name[1]
            switch c {
            
            case 'i':
               if (elem.Size()==0) {
                   return nil
               }
               i, err := strconv.Atoi(elem.Nodes[0].Name)
               if err != nil {
                   return nil
               }
               node = node.GetIndex(i)
            case 's':
               if (elem.Size()==0) {
                   // This case is {}, meaning that we must return
                   // all ocurrences of the token just before.
                   // And that means creating a new Graph object.
                   // BUG(): TO-DO
                   return nil
               }
               i, err := strconv.Atoi(elem.Nodes[0].Name)
               if err != nil {
                   return nil
               }
               i++
               for _, subnode := range node.Nodes {
                   if (elem.Name == subnode.Name) {
	                   i--
	               }
	               if i==0 {
	                   node = subnode
	               }
               }
               
            case 'g':
               // BUG(): TO-DO
               return nil
            default: return nil
            
            }
        } else {
            strip = true;
            node = node.GetNode(elem.Name)
        }
        
        if node == nil {
            break
        }
    }
    
    if (strip && (node!=nil) ) {
        node2 := NullGraph()
        node2.Nodes = node.Nodes
        return node2
    }
    
    return node
}

// String is the OGDL text emitter. 
//
// Strings need to be quoted if they contain spaces, newlines or special characters.
// Null elements are not printed, and act as transparent nodes.
// BUG():Handle comments correctly
//
func (g *Graph) String() string {
    if g==nil {
       return ""
    }
    
    buffer := &bytes.Buffer{}
    
    g.print(0, buffer)
    
    return buffer.String()
}

// print is the private, lower level, implementation of String.
// It takes two parameters, the level and a string to which the
// result is printed.
//
func (g *Graph) print(n int, buffer *bytes.Buffer) {

    sp :="";
    for i:=0; i<n; i++ {
        sp += "  ";
    }
    
    /* 
    When printing strings with newlines, there are two possibilities:
    block or quoted. Block is cleaner, but limited to leaf nodes. If the node is
    not leaf (it has subnodes), then we are forced to print a multiline quoted
    string.
    
    If the string has no newlines but spaces or special characters, then the
    same rule applies: quote those nodes that are non-leaf, print a block
    otherways.
    
    [!] Cannot print blocks at level 0? Or can we?
    */
    
    if strings.IndexAny(g.Name, "\n\r \t'\",()")!=-1 {
    
            /* print quoted */
            buffer.WriteString(sp)
            buffer.WriteByte('"')

            var c byte
            
            for i:=0;i<len(g.Name);i++ {
                c = g.Name[i] // byte, not rune
                if c==13 {
                    continue; // just ignore CR
                } else if c==10 {
                    buffer.WriteByte('\n')
                    buffer.WriteString(sp)
                    buffer.WriteByte(' ')
                } else if c=='"' { 
                    buffer.WriteString("\\\"")  // BUG(): check if \ was already there
                } else {
                    buffer.WriteByte(c)
                }
            }
            
            buffer.WriteString("\"\n")
    } else {   
        if len(g.Name) == 0 {
            n--
        } else {
            buffer.WriteString(sp)
            buffer.WriteString(g.Name)
            buffer.WriteByte('\n')
        }
    }
    
    for i:=0; i<len(g.Nodes); i++ {   
        node := g.Nodes[i]
        node.print(n+1,buffer)
    }
}

// Size returns the number of subnodes (outgoing edges) of this node
//
func (g *Graph) Size() int {
    return len(g.Nodes)
}




