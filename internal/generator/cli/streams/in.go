//nolint:dupl
package streams

import (
	"io"

	"github.com/moby/term"
)

type nopReadCloser struct {
	io.Reader
}

func (nopReadCloser) Close() error { return nil }

// In is an input stream to read user input. It implements [io.ReadCloser].
type In struct {
	isTerminal bool
	in         io.ReadCloser
}

// NewIn returns a new [In] from an [io.Reader].
func NewIn(in io.Reader) *In {
	i := &In{}

	if readCloser, ok := in.(io.ReadCloser); ok {
		i.in = readCloser
	} else {
		i.in = nopReadCloser{in}
	}

	_, i.isTerminal = term.GetFdInfo(in)

	return i
}

// Read implements the [io.Reader] interface.
func (i *In) Read(p []byte) (int, error) {
	return i.in.Read(p) //nolint:wrapcheck
}

// Close implements the [io.Closer] interface.
func (i *In) Close() error {
	return i.in.Close() //nolint:wrapcheck
}

// IsTerminal returns true if this stream is connected to a terminal.
func (i *In) IsTerminal() bool {
	return i.isTerminal
}
