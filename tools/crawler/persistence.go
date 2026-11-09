package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

type crawlResult struct {
	url, payload    string
	missing, failed bool
}

// crawlSink serializes payload and state writes. A processed entry is written
// only after its payload flush succeeds. This is not an atomic transaction or
// an fsync guarantee: interruptions may leave partial payloads or duplicates on
// retry. A persistence failure stops the run rather than skipping unwritten data.
type crawlSink struct {
	mu                                  sync.Mutex
	results, processed, failed, missing *bufio.Writer
	files                               []io.Closer
	cancel                              context.CancelFunc
	err                                 error
}

func openCrawlSink(out, processed, failed, missing string, cancel context.CancelFunc) (*crawlSink, error) {
	s := &crawlSink{cancel: cancel}
	writers := []**bufio.Writer{&s.results, &s.processed, &s.failed, &s.missing}
	for i, path := range []string{out, processed, failed, missing} {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("open crawler output %q: %w", path, err), s.close())
		}
		s.files = append(s.files, f)
		*writers[i] = bufio.NewWriter(f)
	}
	return s, nil
}

func (s *crawlSink) persist(r crawlResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	if err := s.write(r); err != nil {
		s.err = fmt.Errorf("persist %s: %w", r.url, err)
		s.cancel()
	}
	return s.err
}

func (s *crawlSink) write(r crawlResult) error {
	if r.failed {
		return writeURLLineAndFlush(s.failed, r.url, "failed")
	}
	w := s.results
	if r.missing {
		w = s.missing
	}
	if _, err := w.WriteString(r.payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush payload: %w", err)
	}
	return writeURLLineAndFlush(s.processed, r.url, "processed")
}

// Called only after workers finish (or before they start on setup failure).
// Successful records are already flushed; do not retry buffers after a failure.
func (s *crawlSink) close() error {
	var err error
	for _, f := range s.files {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close crawler output: %w", closeErr))
		}
	}
	return err
}
