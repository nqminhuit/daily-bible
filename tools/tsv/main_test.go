package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTSVMain_BasicConversionAndDedup(t *testing.T) {
	dir := t.TempDir()

	in := `-------
	URL: https://www.vaticannews.va/vi/loi-chua-hang-ngay/2026/03/19.html
	__ref__: Mt 1,16.18-21.24a
	Tin Mừng Chúa Giê-su Ki-tô theo thánh Mát-thêu.    Mt 1,16.18-21.24a
	{{16}}  Ông Gia-cóp sinh ông Giu-se, chồng của bà Ma-ri-a, bà là mẹ Đức Giê-su cũng gọi là Đấng Ki-tô.
	{{18}}  Sau đây là gốc tích Đức Giê-su Ki-tô : bà Ma-ri-a, mẹ Người, đã thành hôn với ông Giu-se. Nhưng trước khi hai ông bà về chung sống, bà đã có thai do quyền năng Chúa Thánh Thần.
	{{16}}  Ông Giu-se, chồng bà, là người công chính và không muốn tố giác bà, nên mới định tâm bỏ bà cách kín đáo.
	`
	inputPath := filepath.Join(dir, "gospels.txt")
	if err := os.WriteFile(inputPath, []byte(in), 0644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "gospels.tsv")
	if err := convert(inputPath, outputPath); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (deduped), got %d: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "Mt\t1\t16\t") {
		t.Fatalf("unexpected first line prefix: %q", lines[0])
	}
	if strings.Contains(lines[0], "\\t") {
		t.Fatalf("tabs should be replaced in text: %q", lines[0])
	}
}

func TestTSVMain_NormalizesReferenceWithoutBookSpace(t *testing.T) {
	dir := t.TempDir()

	in := `-------
	URL: https://www.vaticannews.va/vi/loi-chua-hang-ngay/2026/03/19.html
	__ref__: Ga10,1-10
	{{1}} Câu một
	{{2}} Câu hai
	`
	inputPath := filepath.Join(dir, "gospels.txt")
	if err := os.WriteFile(inputPath, []byte(in), 0644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "gospels.tsv")
	if err := convert(inputPath, outputPath); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "Ga\t10\t1\t") {
		t.Fatalf("unexpected first line prefix: %q", lines[0])
	}
}

func TestTSVMain_SkipsUnknownBookReferences(t *testing.T) {
	dir := t.TempDir()

	in := `-------
	URL: https://example.invalid/a
	__ref__: Xa 1,1-2
	{{1}} Câu không hợp lệ
	-------
	URL: https://example.invalid/b
	__ref__: Mt 1,1-2
	{{1}} Câu hợp lệ
	`
	inputPath := filepath.Join(dir, "gospels.txt")
	if err := os.WriteFile(inputPath, []byte(in), 0644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "gospels.tsv")
	if err := convert(inputPath, outputPath); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected only 1 line from valid canonical book, got %d: %v", len(lines), lines)
	}
	if !strings.HasPrefix(lines[0], "Mt\t1\t1\t") {
		t.Fatalf("unexpected output line: %q", lines[0])
	}
}
