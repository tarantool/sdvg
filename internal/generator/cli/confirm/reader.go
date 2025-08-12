package confirm

import "io"

// cancelableReader wraps an io.Reader and can be closed to make future reads fail.
type cancelableReader struct {
	r      io.Reader
	closed chan struct{}
}

// newCancelableReader creates a ReadCloser from an io.Reader.
// Closing it will make subsequent Read() calls return io.EOF.
func newCancelableReader(r io.Reader) io.ReadCloser {
	return &cancelableReader{
		r:      r,
		closed: make(chan struct{}),
	}
}

func (c *cancelableReader) Read(p []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, io.EOF
	default:
		return c.r.Read(p) //nolint:wrapcheck
	}
}

func (c *cancelableReader) Close() error {
	select {
	case <-c.closed:
		// already closed
	default:
		close(c.closed)
	}

	return nil
}
