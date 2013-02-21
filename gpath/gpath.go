// (C) Copyright 2012-2013, Rolf Veen.
// See the LICENCE file.

package main

import (
  "os"
  "ogdl"
)

// gpath <path> [file]
//
// Return the specified 'path' from an OGDL file,
// or from stdin
//
func main() {
    
    if len(os.Args)<2 {
        println("usage\n  gpath <path> [file]")
        return
    }
    
    // If there is only one argument, than that is a path
    // If the path is just '.', return the whole graph
    // (in canonical form).
    //
    path := os.Args[1]
    
    // A second argument is a file name
    source := os.Stdin
    var err error = nil
    if len(os.Args)>2 {
        source, err = os.Open(os.Args[2])
        if err!=nil {
            println(err.Error())
            return
        }
    }
    
    p := ogdl.NewParser(source)
    if p==nil {
        println("Parser == nil")
        return
    }
    
    err = p.Parse()
    
    if err!=nil {
        println("Error",err.Error())
        return
    }
    
    g := p.Graph()
    
    r := g
    
    if path != "." {
        r = g.Get(path)
    }   
    
    print(r.String())
}
