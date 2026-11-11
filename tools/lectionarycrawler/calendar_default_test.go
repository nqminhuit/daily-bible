package main

import (
	"os"
	"testing"
)

// These suites exercise the source-only contract; the production default is the Vietnam calendar.
func TestMain(m *testing.M) {
	os.Setenv("READING_CALENDAR", "vatican-vi")
	os.Exit(m.Run())
}
