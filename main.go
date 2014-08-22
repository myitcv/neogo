// Copyright 2014 Paul Jolly <paul@myitcv.org.uk>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/juju/errgo"
	"github.com/myitcv/neovim"
)

// WARNING: this code is rather messy right now..

func main() {
	c, err := neovim.NewUnixClient("unix", nil, &net.UnixAddr{Name: os.Getenv("NEOVIM_LISTEN_ADDRESS")})
	if err != nil {
		log.Fatalf("Could not setup client: %v", errgo.Details(err))
	}
	c.PanicOnError = true

	event := "cursor_moved"

	// cause go files to trigger the event
	com := fmt.Sprintf(`au TextChanged,TextChangedI <buffer> call send_event(0, "%v", [1])`, event)
	c.Command(com)

	// now register to process that event
	respChan := make(chan neovim.SubscriptionEvent)
	errChan := make(chan error)
	c.SubChan <- neovim.Subscription{
		Topic:  event,
		Events: respChan,
		Error:  errChan,
	}

	// was registration successful?
	err = <-errChan
	if err != nil {
		log.Fatalf("Could not setup subscription: %v\n", err)
	}

	// now perform the subscribe in Neovim
	c.Subscribe("cursor_moved")

	// now listen for that event
	for {
		select {
		case <-respChan:
			// write the current buffer to a temp file
			// TODO there is probably a better way of doing this...
			// that also avoids running the go tool as a separate process?
			cb, _ := c.GetCurrentBuffer()
			tf, err := tempFile()
			if err != nil {
				log.Fatalf("Could not create temp file: %v\n", errgo.Details(err))
			}
			bc, _ := cb.GetSlice(0, -1, true, true)
			for _, v := range bc {
				_, err := tf.WriteString(v + "\n")
				if err != nil {
					log.Fatalf("Could not write to temp file", errgo.Details(err))
				}
			}
			err = tf.Close()
			if err != nil {
				log.Fatalf("Could not close temp file: %v\n", errgo.Details(err))
			}

			fset := token.NewFileSet()

			f, err := parser.ParseFile(fset, tf.Name(), nil, parser.AllErrors)
			if f == nil && err != nil {
				fmt.Println("We got an error on the parse")
			}

			// ast.Print(fset, f)

			sg := &synGenerator{fset: fset, f: f}
			ast.Walk(sg, f)
			c.Command("call clearmatches()")
			if len(sg.functions) > 0 {
				synCom := ""
				join := ""
				for i := range sg.functions {
					pos := sg.functions[i]
					// fmt.Printf("We got a function at %v:%v\n", pos.Line, pos.Column)
					synCom = fmt.Sprintf("%v%v\\%%%vl\\%%%vc.\\{4\\}", synCom, join, pos.Line, pos.Column)
					join = "\\|"
				}
				resCom := fmt.Sprintf("match Keyword /%v/", synCom)
				// fmt.Println(resCom)
				c.Command(resCom)
			}
		}
	}
}

type synGenerator struct {
	fset      *token.FileSet
	f         *ast.File
	functions []*token.Position
}

func (s *synGenerator) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.FuncType:
		pos := s.fset.Position(node.Func)
		s.functions = append(s.functions, &pos)
	}
	return s
}

// Use a sledgehammer to crack a nut
func tempFile() (*os.File, error) {
	td := os.TempDir()
	f, err := os.OpenFile("/dev/urandom", os.O_RDONLY, 0)
	if err != nil {
		log.Fatalf("Could not open /dev/urandom: %v\n", err)
	}
	b := make([]byte, 16)
	_, err = f.Read(b)
	if err != nil {
		log.Fatalf("Could not read from urandom: %v\n", err)
	}
	f.Close()
	if err != nil {
		log.Fatalf("Could not close urandom: %v\n", err)
	}
	uuid := fmt.Sprintf("%v/%x-%x-%x-%x-%x.go", td, b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	res, err := os.Create(uuid)
	if err != nil {
		log.Fatalf("Could not create temp file: %v\n", err)
	}
	return res, nil
}

type buildError struct {
	file   string
	line   uint64
	errMsg string
}
