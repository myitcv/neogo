package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/juju/errgo"
	"github.com/myitcv/neovim"
	. "gopkg.in/check.v1"
)

type NeovimGoTest struct {
	client *neovim.Client
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&NeovimGoTest{})

func (t *NeovimGoTest) SetUpTest(c *C) {
	la := os.Getenv("NEOVIM_LISTEN_ADDRESS")
	client, err := neovim.NewUnixClient("unix", nil, &net.UnixAddr{Name: la})
	if err != nil {
		log.Fatalf("Could not setup client: %v", errgo.Details(err))
	}
	client.PanicOnError = true
	t.client = client
}

func (t *NeovimGoTest) BenchmarkBufferGetSlice(c *C) {
	// TODO this needs Neovim to be started with the right file
	// can be updated when we can use headless testing
	cb, _ := t.client.GetCurrentBuffer()
	for i := 0; i < c.N; i++ {
		bc, _ := cb.GetSlice(0, -1, true, true)
		_ = []byte(strings.Join(bc, "\n"))
	}
}

func (t *NeovimGoTest) BenchmarkParse(c *C) {
	data, err := ioutil.ReadFile("_testfiles/20k_lines.go")
	if err != nil {
		panic(err)
	}
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		if _, err := parser.ParseFile(token.NewFileSet(), "", data, parser.AllErrors|parser.ParseComments); err != nil {
			c.Fatalf("benchmark failed due to parse error: %s", err)
		}
	}
}

func (t *NeovimGoTest) BenchmarkWalk(c *C) {
	data, err := ioutil.ReadFile("_testfiles/20k_lines.go")
	if err != nil {
		panic(err)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "dummy", data, parser.AllErrors|parser.ParseComments)
	if err != nil {
		c.Fatalf("benchmark failed due to parse error: %s", err)
	}
	sg := NewSynGenerator()
	sg.fset = fset
	sg.f = f
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		sg.nodes = make(map[position]*match)
		ast.Walk(sg, f)
		for _, c := range f.Comments {
			ast.Walk(sg, c)
		}
	}
}
