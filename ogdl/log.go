// (C) Copyright 2012-2013, Rolf Veen.
// See the LICENCE file.

package ogdl

import "os"

// Log is a log store for binary OGDL objects.
//
// All objects are appended to a file, and
// a position is returned. 
//
type Log struct {
    f *os.File
    autoSync bool
}

func OpenLog(file string) *Log {

    f, err := os.OpenFile(file, os.O_RDWR | os.O_CREATE, 0666) 
    if err != nil {
         return nil
    }
    
    log := Log{f,true}
    
    return &log
}

func (log *Log) ReadByteAt (pos int64) byte {
    _,_ = log.f.Seek(pos,0);
    
    b := make([]byte,1)
    _, _ = log.f.Read(b)
    return b[0]
}

func (log *Log) Close() {
    log.f.Close()
}

func (log *Log) Sync() {
    log.f.Sync();
}

func (log *Log) Add(g *Graph) int64 {
    
    b, _ := g.Binary()
    log.f.Write(b)
    
    i,_ := log.f.Seek(0,2);
    if (log.autoSync) {
        log.f.Sync()
    }
    
    return i
}

func (log *Log) AddBinary(b []byte) int64 {
    
    log.f.Write(b);
    i,_ := log.f.Seek(0,2)
    
    if (log.autoSync) {
        log.f.Sync()
    }

    return i
}

func (log *Log) Get(i int64) (*Graph, error, int64) {

    /* Position in file */
    _,err := log.f.Seek(i,0);
    if err != nil {
         return nil,err,-1
    }
    
    p := NewBinParser(log.f)
    g := p.Graph()

    return g,err,i+int64(p.N)  
}

func (log *Log) GetBinary(i int64) ([]byte, error, int64) {
    
    /* Position in file */
    _,err := log.f.Seek(i,0)
    if err != nil {
         return nil,err,0
    }
    
    /* Read until EOS of binary OGDL.
    
       There should be a Header first.
     */
    p := NewBinParser(log.f)
    
    if !p.Header() {
        return nil,err,0
    }
    for {
        lev, _, _ /* typ, b*/ := p.Line(false)
        if lev==0 {
            break
        }
    }
    
    n := p.N;
    
    // Read bytes
    b := make([]byte,n)
    _, err = log.f.ReadAt(b,i);
    
    return b,err,int64(n)
}


