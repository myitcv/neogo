// Copyright 2014 Paul Jolly <paul@myitcv.org.uk>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package neogo

import (
	"fmt"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/myitcv/neovim"
)

type Neogo struct {
	c *neovim.Client
}

var fDebugAST bool
var fDebug bool

func (n *Neogo) Init(c *neovim.Client) error {
	n.c = c

	// we want to know when the buffer changes. We do this in a few steps, which
	// are necessarily "out of order"

	// Link the autocmd events TextChanged and TextChangedI to send an event on a topic
	topic := "text_changed"
	com := fmt.Sprintf(`au TextChanged,TextChangedI <buffer> call send_event(0, "%v", [])`, topic)
	c.Command(com)

	// Register a subscription event (and error) channel in our client on this topic
	sub, _ := c.Subscribe(topic)

	// Consume events, parse and send back commands to highlight
	sg := NewSynGenerator()
	for {
		select {
		case <-sub.Events:
			cb, _ := c.GetCurrentBuffer()
			bn, _ := cb.GetName()
			bc, _ := cb.GetSlice(0, -1, true, true)
			src := []byte(strings.Join(bc, "\n"))

			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, bn, src, parser.AllErrors|parser.ParseComments)
			if f == nil && err != nil {
				fmt.Println("We got an error on the parse")
			}

			if fDebugAST {
				ast.Print(fset, f)
			}

			// TODO better way? Do we really need to reparse each time?
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

	return nil
}

func (n *Neogo) Shutdown() error {
	return nil
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
	_LABEL
	_REPEAT
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
	case _LABEL:
		return "Label"
	case _REPEAT:
		return "Repeat"
	default:
		panic("Unknown const mapping")
	}
	return ""
}

type match struct {
	id uint64
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
			com := fmt.Sprintf("matchaddpos('%v', [[%v,%v,%v]])", pos.t, pos.line, pos.col, pos.l)
			id, _ := c.Eval(com)
			if fDebug {
				fmt.Printf("%v\n", com)
			}
			switch id := id.(type) {
			case uint64:
				m.id = id
			case int64:
				m.id = uint64(id)
			}
			m.a = _DELETE
		case _DELETE:
			com := fmt.Sprintf("matchdelete(%v)", m.id)
			c.Eval(com)
			if fDebug {
				fmt.Printf("%v\n", com)
			}
			delete(s.nodes, pos)
		case _KEEP:
			m.a = _DELETE
		}
	}
}

func (s *synGenerator) addNode(t nodeType, l int, _p token.Pos) {
	p := s.fset.Position(_p)
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
	var handleType func(ast.Expr)
	handleType = func(t ast.Expr) {
		switch node := t.(type) {
		case *ast.Ident:
			s.addNode(_TYPE, len(node.Name), node.NamePos)
		case *ast.FuncType:
			s.addNode(_KEYWORD, 4, node.Func)
		case *ast.ChanType:
			s.addNode(_TYPE, 4, node.Begin)
			// TODO add highligthing of chan arrow?
			handleType(node.Value)
		case *ast.MapType:
			s.addNode(_TYPE, 3, node.Map)
			handleType(node.Key)
			handleType(node.Value)
		}
	}
	switch node := node.(type) {
	case *ast.File:
		s.addNode(_STATEMENT, 7, node.Package)
	case *ast.BasicLit:
		if node.Kind == token.STRING {
			s.addNode(_STRING, len(node.Value), node.ValuePos)
		}
	case *ast.Comment:
		s.addNode(_COMMENT, len(node.Text), node.Slash)
	case *ast.GenDecl:
		switch node.Tok {
		case token.VAR:
			s.addNode(_KEYWORD, 3, node.TokPos)
		case token.IMPORT:
			s.addNode(_STATEMENT, 6, node.TokPos)
		case token.CONST:
			s.addNode(_KEYWORD, 5, node.TokPos)
		case token.TYPE:
			s.addNode(_KEYWORD, 4, node.TokPos)
		}
	case *ast.StructType:
		s.addNode(_KEYWORD, 6, node.Struct)
	case *ast.InterfaceType:
		s.addNode(_KEYWORD, 9, node.Interface)
	case *ast.ReturnStmt:
		s.addNode(_KEYWORD, 6, node.Return)
	case *ast.BranchStmt:
		s.addNode(_KEYWORD, len(node.Tok.String()), node.TokPos)
	case *ast.ForStmt:
		s.addNode(_REPEAT, 3, node.For)
	case *ast.GoStmt:
		s.addNode(_STATEMENT, 2, node.Go)
	case *ast.DeferStmt:
		s.addNode(_STATEMENT, 5, node.Defer)
	case *ast.FuncDecl:
		s.addNode(_FUNCTION, len(node.Name.Name), node.Name.NamePos)
		handleType(node.Type)
	case *ast.Field:
		handleType(node.Type)
	case *ast.ValueSpec:
		handleType(node.Type)
	case *ast.SwitchStmt:
		s.addNode(_CONDITIONAL, 6, node.Switch)
	case *ast.SelectStmt:
		s.addNode(_CONDITIONAL, 6, node.Select)
	case *ast.CaseClause:
		s.addNode(_LABEL, 4, node.Case)
	case *ast.RangeStmt:
		// TODO is this always safe to do?
		s.addNode(_REPEAT, 3, node.For)
		key := node.Key.(*ast.Ident)
		ass := key.Obj.Decl.(*ast.AssignStmt)
		rhs := ass.Rhs[0].(*ast.UnaryExpr)
		s.addNode(_REPEAT, 5, rhs.OpPos)
	case *ast.IfStmt:
		s.addNode(_CONDITIONAL, 2, node.If)
		// TODO need to find a way to add else highlighting
	}
	return s
}
