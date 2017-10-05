// Copyright 2017, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdlrf

import (
	"net"
	"ogdl"
)

// Function is the prototype of functions to be server by ogdlrf.Serve
type Function func(net.Conn, *ogdl.Graph) *ogdl.Graph
