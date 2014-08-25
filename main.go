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

func main() {
	c, err := neovim.NewUnixClient("unix", nil, &net.UnixAddr{Name: os.Getenv("NEOVIM_LISTEN_ADDRESS")})
	if err != nil {
		log.Fatalf("Could not setup client: %v", errgo.Details(err))
	}

	// whilst in development, we will simply bail out on errors
	c.PanicOnError = true

	// we want to know when the buffer changes. We do this in a few steps, which
	// are necessarily "out of order"

	// 1. Link the autocmd events TextChanged and TextChangedI to send an event on a topic
	topic := "cursor_moved"
	com := fmt.Sprintf(`au TextChanged,TextChangedI <buffer> call send_event(0, "%v", [1])`, topic)
	c.Command(com)

	// 2. Register a subscription event (and error) channel in our client on this topic
	respChan := make(chan neovim.SubscriptionEvent)
	errChan := make(chan error)
	c.SubChan <- neovim.Subscription{
		Topic:  topic,
		Events: respChan,
		Error:  errChan,
	}

	// 3. Check the registration succeeded; errors here would mean we already have
	// a subscription setup for this topic
	err = <-errChan
	if err != nil {
		log.Fatalf("Could not setup subscription: %v\n", err)
	}

	// 4. Perform the subscribe on the topic; our client will now be told about events on this topic
	c.Subscribe(topic)

	// subscription done

	// Consume events, parse and send back commands to highlight
	for {
		select {
		case <-respChan:
			// write the current buffer to a temp file
			cb, _ := c.GetCurrentBuffer()
			tf, err := tempFile()
			if err != nil {
				log.Fatalf("Could not create temp file: %v\n", errgo.Details(err))
			}
			bc, _ := cb.GetSlice(0, -1, true, true)
			for _, v := range bc {
				_, err := tf.WriteString(v + "\n")
				if err != nil {
					log.Fatalf("Could not write to temp file: %v\n", errgo.Details(err))
				}
			}
			err = tf.Close()
			if err != nil {
				log.Fatalf("Could not close temp file: %v\n", errgo.Details(err))
			}

			// parse the temp file
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, tf.Name(), nil, parser.AllErrors)
			if f == nil && err != nil {
				fmt.Println("We got an error on the parse")
			}

			// ast.Print(fset, f)

			// generate our highlight positions
			sg := &synGenerator{fset: fset, f: f}
			ast.Walk(sg, f)

			// clear current matches
			c.Command("call clearmatches()")

			// set the highlights
			c.Command(sg.genList("Keyword", sg.keywords))
			c.Command(sg.genList("Statement", sg.statements))
			c.Command(sg.genList("String", sg.strings))
		}
	}
}

type position struct {
	l    int
	line int
	col  int
}

type synGenerator struct {
	fset       *token.FileSet
	f          *ast.File
	keywords   []position
	statements []position
	strings    []position
}

func (s *synGenerator) genList(prefix string, l []position) string {
	list := ""
	join := ""
	for i := range l {
		pos := l[i]
		list = fmt.Sprintf("%v%v\\%%%vl\\%%%vc.\\{%v\\}", list, join, pos.line, pos.col, pos.l)
		join = "\\|"
	}
	res := fmt.Sprintf("call matchadd('%v', '%v')", prefix, list)
	// fmt.Printf("%v: %v\n", prefix, res)
	return res
}

func (s *synGenerator) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.File:
		pos := s.fset.Position(node.Package)
		s.statements = append(s.statements, position{l: 7, line: pos.Line, col: pos.Column})
	case *ast.BasicLit:
		pos := s.fset.Position(node.ValuePos)
		if node.Kind == token.STRING {
			s.strings = append(s.strings, position{l: len(node.Value), line: pos.Line, col: pos.Column})
		}
	case *ast.FuncType:
		pos := s.fset.Position(node.Func)
		s.keywords = append(s.keywords, position{l: 4, line: pos.Line, col: pos.Column})
	case *ast.GenDecl:
		pos := s.fset.Position(node.TokPos)
		if node.Tok == token.VAR {
			s.keywords = append(s.keywords, position{l: 3, line: pos.Line, col: pos.Column})
		} else if node.Tok == token.IMPORT {
			s.statements = append(s.statements, position{l: 6, line: pos.Line, col: pos.Column})
		}
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
