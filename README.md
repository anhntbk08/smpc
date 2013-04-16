SMPC Primitives for Go
=======
A simple SMPC library in Go.

DO NOT USE THIS FOR ANYTHING TRULY SECURE, THIS HAS NOT BEEN AUDITED

This library includes primitives for a set of SMPC functions. This is not really complete, nor
is it necessarily secure. We do not handle communication, allowing applications to handle communication,
since one can perhaps build a better communication medium.

For our purposes, we mostly care about the multiplication, and comparison protocols. We do not implement 
a large set of what is possible. This library is neither complete, nor secure.

Dependencies (for this is built on the shoulder of giants):<br/>
ZeroMQ 3.2
go-zmq (originally from https://github.com/vaughan0/go-zmq, forked into https://github.com/apanda/go-zmq)
Protobuf

In this branch we relax anonymity for preferences
