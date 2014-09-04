## `neovim-go`

[![Build Status](https://travis-ci.org/myitcv/neovim-go.svg?branch=master)](https://travis-ci.org/myitcv/neovim-go)

A proof of concept Neovim plugin written against the [`neovim` Go package](http://godoc.org/github.com/myitcv/neovim)
to support Go development in Neovim.

Very very alpha.

## Running the plugin

Start a Neovim instance:

```bash
$ NEOVIM_LISTEN_ADDRESS=/tmp/neovim nvim
```
In another terminal:

```bash
$ go build
$ NEOVIM_LISTEN_ADDRESS=/tmp/neovim ./neovim-go
```

Switch back to the Neovim instance, and start writing some Go!

## Features implemented

* syntax highlighting via [`go/parser`](http://godoc.org/go/parser) (partial)

## Features TODO list

* complete syntax highlighting
* support for syntax-based commands (e.g. fold a function, struct, Tagbar), exposed via commands that
ultimately take advantage of the `go/parser` integration
* completion via some integration of [`gocode`](https://github.com/nsf/gocode); reuse the `go/parser`
part of the plugin here
* integration of [Go oracle](https://docs.google.com/a/myitcv.org.uk/document/d/1SLk36YRjjMgKqe490mSRzOPYEDe0Y_WQNRv-EiFYUyw/view) - again,
reuse the `go/parser` part of the plugin here. Go Oracle will be exposed via a number of
language-specific commands, e.g, pointsto
* integration of the Go toolset, e.g. `gofmt`, `godoc`, `godef` (some overlap here with oracle), `test`, `govet`, etc. Again, where
possible reusing the `go/parser` part of the plugin

