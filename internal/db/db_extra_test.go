package db

import "testing"

func TestOpenEmptyPath(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected non-nil error when opening empty path")
	}
}
