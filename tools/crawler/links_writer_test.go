package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type testWriteCloser struct {
	io.Writer
	closeFunc func() error
}

func (w testWriteCloser) Close() error { return w.closeFunc() }

func TestWriteLinksAndCloseErrors(t *testing.T) {
	writeFailure := errors.New("write failed")
	closeFailure := errors.New("close failed")
	for _, size := range []int{0, 10, 8192} {
		closed := false
		out := testWriteCloser{
			Writer:    writerFunc(func([]byte) (int, error) { return 0, writeFailure }),
			closeFunc: func() error { closed = true; return closeFailure },
		}
		var links []string
		if size > 0 {
			links = []string{strings.Repeat("x", size)}
		}
		err := writeLinksAndClose(out, links)
		if !closed || !errors.Is(err, closeFailure) {
			t.Fatalf("close failure lost: %v", err)
		}
		if size > 0 && !errors.Is(err, writeFailure) {
			t.Fatalf("write/flush failure lost: %v", err)
		}
	}
}
