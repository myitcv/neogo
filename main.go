// Copyright 2014 Paul Jolly <paul@myitcv.org.uk>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
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

var fDebug = flag.Bool("debug", false, "enable debug logging")
var fDebugAST = flag.Bool("debugAST", false, "enable print of AST")

func main() {
	flag.Parse()

	c, err := neovim.NewUnixClient("unix", nil, &net.UnixAddr{Name: os.Getenv("NEOVIM_LISTEN_ADDRESS")})
	if err != nil {
		log.Fatalf("Could not setup client: %v", errgo.Details(err))
	}

	// whilst in development, we will simply bail out on errors
	c.PanicOnError = true

	// we want to know when the buffer changes. We do this in a few steps, which
	// are necessarily "out of order"

	// 1. Link the autocmd events TextChanged and TextChangedI to send an event on a topic
	topic := "text_changed"
	com := fmt.Sprintf(`au TextChanged,TextChangedI <buffer> call send_event(0, "%v", [])`, topic)
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
	sg := NewSynGenerator()
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
			f, err := parser.ParseFile(fset, tf.Name(), nil, parser.AllErrors|parser.ParseComments)
			if f == nil && err != nil {
				fmt.Println("We got an error on the parse")
			}

			if *fDebugAST {
				ast.Print(fset, f)
			}

			// TODO better way? Need to reparse each time?
			sg.fset = fset
			sg.f = f

			// generate our highlight positions
			ast.Walk(sg, f)

			for _, c := range f.Comments {
				ast.Walk(sg, c)
			}

			// set the highlights
			sg.sweepMap(c)
		}
	}
}

type position struct {
	l    int
	line int
	col  int
	t    nodeType
}

type action uint32

type nodeType uint32

const (
	_ADD action = iota
	_KEEP
	_DELETE
)

const (
	_KEYWORD nodeType = iota
	_STATEMENT
	_STRING
	_TYPE
	_CONDITIONAL
	_FUNCTION
	_COMMENT
)

func (n nodeType) String() string {
	switch n {
	case _KEYWORD:
		return "Keyword"
	case _STATEMENT:
		return "Statement"
	case _STRING:
		return "String"
	case _TYPE:
		return "Type"
	case _CONDITIONAL:
		return "Conditional"
	case _FUNCTION:
		return "Function"
	case _COMMENT:
		return "Comment"
	default:
		panic("Unknown const mapping")
	}
	return ""
}

type match struct {
	id int64
	a  action
}

type synGenerator struct {
	fset  *token.FileSet
	f     *ast.File
	nodes map[position]*match
}

func NewSynGenerator() *synGenerator {
	res := &synGenerator{
		nodes: make(map[position]*match),
	}
	return res
}

func (s *synGenerator) sweepMap(c *neovim.Client) {
	for pos, m := range s.nodes {
		switch m.a {
		case _ADD:
			com := fmt.Sprintf("matchadd('%v', '\\%%%vl\\%%%vc\\_.\\{%v\\}')", pos.t, pos.line, pos.col, pos.l)
			id_i, _ := c.Eval(com)
			if *fDebug {
				fmt.Printf("%v, res = %v\n", com, id_i)
			}
			id := id_i.(int64)
			m.id = id
			m.a = _DELETE
		case _DELETE:
			com := fmt.Sprintf("matchdelete(%v)", m.id)
			c.Eval(com)
			if *fDebug {
				fmt.Printf("%v\n", com)
			}
			delete(s.nodes, pos)
		case _KEEP:
			m.a = _DELETE
		}
	}
}

func (s *synGenerator) addNode(t nodeType, l int, p token.Position) {
	pos := position{t: t, l: l, line: p.Line, col: p.Column}
	if m, ok := s.nodes[pos]; ok {
		// when we call add, we mark the match as delete
		// for efficiency next time around, hence the need
		// to mark this as keep
		m.a = _KEEP
	} else {
		// we leave anything that needs to be deleted
		// and add a new match, with action == _ADD
		s.nodes[pos] = &match{a: _ADD}
	}
}

func (s *synGenerator) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.File:
		pos := s.fset.Position(node.Package)
		s.addNode(_STATEMENT, 7, pos)
	case *ast.BasicLit:
		pos := s.fset.Position(node.ValuePos)
		if node.Kind == token.STRING {
			s.addNode(_STRING, len(node.Value), pos)
		}
	case *ast.FuncType:
		pos := s.fset.Position(node.Func)
		s.addNode(_KEYWORD, 4, pos)
	case *ast.Comment:
		pos := s.fset.Position(node.Slash)
		s.addNode(_COMMENT, len(node.Text), pos)
	case *ast.GenDecl:
		pos := s.fset.Position(node.TokPos)
		switch node.Tok {
		case token.VAR:
			s.addNode(_KEYWORD, 3, pos)
		case token.IMPORT:
			s.addNode(_STATEMENT, 6, pos)
		case token.CONST:
			s.addNode(_KEYWORD, 5, pos)
		}
	case *ast.Ident:
		pos := s.fset.Position(node.NamePos)
		if node.Obj == nil {
			switch node.Name {
			case "bool", "string", "error", "int", "int8", "int16", "int32", "int64", "rune", "byte", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128":
				s.addNode(_TYPE, len(node.Name), pos)
			case "true", "false", "nil", "iota":
				s.addNode(_KEYWORD, len(node.Name), pos)
			}
		} else {
			switch node.Obj.Kind {
			case ast.Fun:
				s.addNode(_FUNCTION, len(node.Obj.Name), pos)
			}
		}
	case *ast.MapType:
		pos := s.fset.Position(node.Map)
		s.addNode(_TYPE, 3, pos)
	case *ast.IfStmt:
		pos := s.fset.Position(node.If)
		s.addNode(_CONDITIONAL, 2, pos)
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
