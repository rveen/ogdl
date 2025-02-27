# OGDL for Go

[OGDL](http://ogdl.org) is a textual format to write trees or graphs of text, where indentation and spaces define the structure. Here is an example:

    network
      ip 192.168.1.100
      gw 192.168.1.9

The language is simple, either in its textual representation or its number of productions (the specification rules), allowing for compact implementations.

OGDL character streams are normally formed by Unicode characters, and encoded as UTF-8 strings, but any encoding that is ASCII transparent is compatible with the specification and the implementations.

This implementation does not support cyles.

## Documentation

See [here]([http://godoc.org/github.com/rveen/ogdl](https://pkg.go.dev/github.com/rveen/ogdl)).

## Installation

    go get github.com/rveen/ogdl

    go get gopkg.in/rveen/ogdl.v1  (for the previous -stable- version)

## Discussion

There is a list: [ogdl-go](https://groups.google.com/forum/?fromgroups&hl=en#!forum/ogdl-go).

## Example: a configuration file

If we have a text file 'conf.ogdl' like this:

    eth0
      ip
        192.168.1.1
      gateway
        192.168.1.10
      mask
        255.255.255.0
      timeout
        20
then,

    g := ogdl.FromFile("conf.ogdl")
    ip := g.Get("eth0.ip").String()
    to := g.Get("eth0.timeout").Int64(60)
    println("ip:",ip,", timeout:",to)

will print

    ip: 192.168.1.1, timeout: 20

If the timeout parameter were not present, then the default value (60) will be
assigned to 'to'.
