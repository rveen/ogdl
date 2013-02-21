package ogdl

import ( 
  "testing"
)

// This array should hold all known cases
// that the parser may encounter.

var text = [...]string {
    // Short documents
    "a",
    "\na",
    "a\n",
    " a",
    "a     ",
    "a     \n",
    "a\nb",
    "x\r\ny",
    "x \r \n y",
    // Blocks
    "a \\\n  b",
    "a \\ \n  b",			// Space between \ and end of line. Nasty.
    "a \\\n  b\n",
    "a \\\n  b\n  c",
    // Groups
    "a (b c) d",			// Should trigger an error.
    "a (b c), d",
    // Quoted
    
}

func TestParser(t *testing.T) {
    
    for i:=0; i<len(text); i++ {
        p := NewStringParser(text[i])
        p.Parse()
        g := p.Graph()
        print(g.String())
    }
}

func TestUnread(t *testing.T) {
    
    p := NewStringParser("ab")
    
    c := p.Read()
    if (c != 'a') {
        t.Fatal("Read failed: a")
    }
    p.Unread()
    c = p.Read()
    if (c != 'a') {
        t.Fatal("Unread failed: a")
    }
    c = p.Read()
    if (c != 'b') {
        t.Fatal("Read failed: b")
    }
    c = p.Read()
    if (c != 0) {
        t.Fatal("Read failed: EOS")
    }
    
    p.Unread()
    p.Unread()
    c = p.Read()
    if (c != 'b') {
        t.Fatal("Unread failed: b")
    }
    c = p.Read()
    if (c != 0) {
        t.Fatal("Read failed: EOS")
    }
    c = p.Read()
    if (c != 0) {
        t.Fatal("Read failed: EOS")
    }
}