package main

import (
	"flag"
	"log"
	"net"
	"os"

	"github.com/juju/errgo"
	"github.com/myitcv/neovim"
)

var fDebug = flag.Bool("debug", false, "enable debug logging")
var fDebugAST = flag.Bool("debugAST", false, "enable print of AST")

func main() {
	c, err := neovim.NewUnixClient("unix", nil, &net.UnixAddr{Name: os.Getenv("NEOVIM_LISTEN_ADDRESS")})
	if err != nil {
		log.Fatalf("Could not setup client: %v", errgo.Details(err))
	}
	c.PanicOnError = true
}
