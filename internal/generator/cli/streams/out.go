//nolint:dupl
package streams

import (
	"io"

	"github.com/moby/term"
	"github.com/tarantool/sdvg/internal/generator/cli/utils"
)

// Out is an output stream to write normal program output. It implements an [io.WriteCloser].
type Out struct {
	isTerminal bool
	out        io.WriteCloser
}

// NewOut returns a new [Out] from an [io.Writer].
func NewOut(out io.Writer) *Out {
	o := &Out{}

	if writeCloser, ok := out.(io.WriteCloser); ok {
		o.out = writeCloser
	} else {
		o.out = utils.DummyReadWriteCloser{Writer: out}
	}

	_, o.isTerminal = term.GetFdInfo(out)

	return o
}

// Write implements the [io.Writer] interface.
func (o *Out) Write(p []byte) (int, error) {
	return o.out.Write(p) //nolint:wrapcheck
}

// Close implements the [io.Closer] interface.
func (o *Out) Close() error {
	return o.out.Close() //nolint:wrapcheck
}

// IsTerminal returns true if this stream is connected to a terminal.
func (o *Out) IsTerminal() bool {
	return o.isTerminal
}
