package neogo

import (
	"github.com/myitcv/neovim"
	"github.com/tinylib/msgp/msgp"
)

// **************************
// BufferUpdate
func (n *Neogo) newBufferUpdateResponder() neovim.AsyncDecoder {
	return &bufferUpdateWrapper{Neogo: n}
}

func (n *bufferUpdateWrapper) Args() msgp.Decodable {
	return new(neovim.NilDeocdable)
}

func (n *bufferUpdateWrapper) Params() *neovim.MethodOptionParams {
	return nil
}

func (n *bufferUpdateWrapper) Eval() msgp.Decodable {
	return nil
}

type bufferUpdateWrapper struct {
	*Neogo
}

func (g *bufferUpdateWrapper) Run() error {
	err := g.Neogo.BufferUpdate(nil)
	return err
}
