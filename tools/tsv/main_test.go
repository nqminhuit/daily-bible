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

	in := `-------
	URL: https://www.vaticannews.va/vi/loi-chua-hang-ngay/2026/03/19.html
	__ref__: Mt 1,16.18-21.24a
	Tin Mừng Chúa Giê-su Ki-tô theo thánh Mát-thêu.    Mt 1,16.18-21.24a
	{{16}}  Ông Gia-cóp sinh ông Giu-se, chồng của bà Ma-ri-a, bà là mẹ Đức Giê-su cũng gọi là Đấng Ki-tô.
	{{18}}  Sau đây là gốc tích Đức Giê-su Ki-tô : bà Ma-ri-a, mẹ Người, đã thành hôn với ông Giu-se. Nhưng trước khi hai ông bà về chung sống, bà đã có thai do quyền năng Chúa Thánh Thần.
	{{16}}  Ông Giu-se, chồng bà, là người công chính và không muốn tố giác bà, nên mới định tâm bỏ bà cách kín đáo.
	`
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
	if !strings.HasPrefix(lines[0], "Mt\t1\t16\t") {
		t.Fatalf("unexpected first line prefix: %q", lines[0])
	}
	if strings.Contains(lines[0], "\\t") {
		t.Fatalf("tabs should be replaced in text: %q", lines[0])
	}
}
