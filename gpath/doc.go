// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// gpath <path> [file]
//
// Return the specified path from an OGDL file,
// or from stdin. If the path is a dot, then the complete input graph
// is returned. The specifications for the OGDL language and the OGDL path language
// are available at http://ogdl.org.
//
// For example, if we have a configuration file conf.g like this:
//
//     eth0
//        ip 128.0.0.10
//        gw 128.0.0.1
//
// then the command
//
//     # gpath eth0.ip conf.g
//
// will print
//
//     128.0.0.1
//
package main
