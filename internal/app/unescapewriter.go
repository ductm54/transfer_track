package app

import (
	"bytes"
	"fmt"
	"io"
)

// UnescapeWriter is a writer that unescapes escaped quotes in JSON.
type UnescapeWriter struct {
	w io.WriteCloser
}

// Write implements the io.Writer interface.
func (u UnescapeWriter) Write(p []byte) (int, error) {
	var (
		eq = []byte{'\\', '"'}
		qq = []byte{'"'}
	)

	nw := len(p)
	p = bytes.ReplaceAll(p, eq, qq)

	m, err := u.w.Write(p)
	if err != nil {
		return m, fmt.Errorf("writing to underlying writer: %w", err)
	}

	if m == len(p) {
		return nw, nil
	}

	return m, nil
}

// CloseWrite closes the underlying writer.
func (u UnescapeWriter) CloseWrite() error {
	if err := u.w.Close(); err != nil {
		return fmt.Errorf("closing underlying writer: %w", err)
	}

	return nil
}
