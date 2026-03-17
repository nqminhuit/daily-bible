package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTSVMain_BasicConversionAndDedup(t *testing.T) {
	temp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	dir := "build"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	in := "Tin mừng: Lc 18,9-14\n{{1}} Hello\tworld\n{{2}} Second line\n{{1}} Duplicate should be ignored\n"
	if err := os.WriteFile(filepath.Join(dir, "gospels.txt"), []byte(in), 0644); err != nil {
		t.Fatal(err)
	}

	// run main
	main()

	outPath := filepath.Join(dir, "gospels.tsv")
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (deduped), got %d: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "Lc\t18\t1\t") {
		t.Fatalf("unexpected first line prefix: %q", lines[0])
	}
	if strings.Contains(lines[0], "\\t") {
		t.Fatalf("tabs should be replaced in text: %q", lines[0])
	}
}

func TestTSVMain_ParsesChapterFromColonReference(t *testing.T) {
	temp := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("build", 0755); err != nil {
		t.Fatal(err)
	}

	in := "Tin mừng: Mc 6:17-29\n{{17}} one\n{{18}} two\n"
	if err := os.WriteFile(filepath.Join("build", "gospels.txt"), []byte(in), 0644); err != nil {
		t.Fatal(err)
	}

	main()

	b, err := os.ReadFile(filepath.Join("build", "gospels.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "Mc\t6\t17\t") {
		t.Fatalf("expected chapter 6 for colon reference, got %q", lines[0])
	}
}
