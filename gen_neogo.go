//go:generate msgp
package neogo

import (
	"github.com/myitcv/neovim"
	"github.com/tinylib/msgp/msgp"
)

// **************************
// BufferUpdate
func (n *Neogo) newBufferUpdateResponder() neovim.AsyncDecoder {
	return &bufferUpdateWrapper{
		Neogo: n,
		args:  &BufferUpdateArgs{},
	}
}

func (n *bufferUpdateWrapper) Args() msgp.Decodable {
	return n.args
}

type bufferUpdateWrapper struct {
	*Neogo
	args *BufferUpdateArgs
}

//msgp:tuple BufferUpdateArgs
type BufferUpdateArgs struct {
	FunctionArgs [0]int64
}

func (g *bufferUpdateWrapper) Run() error {
	err := g.Neogo.BufferUpdate()
	return err
}
