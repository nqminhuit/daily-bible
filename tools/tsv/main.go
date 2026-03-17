package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	cst "github.com/minh/daily-bible/internal/constants"
)

var verseRE = regexp.MustCompile(`\{\{(\d+)\}\}\s*(.*)`)
var chapterRE = regexp.MustCompile(`^(\d+)`)

func extractChapter(ref string) string {
	parts := strings.Fields(ref)
	if len(parts) < 2 {
		return ""
	}

	chapterPart := parts[1]
	if m := chapterRE.FindStringSubmatch(chapterPart); m != nil {
		return m[1]
	}

	return ""
}

func main() {
	file, err := os.Open(cst.OutFilename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	out, err := os.Create(cst.OutTsvFilename)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	defer writer.Flush()
	scanner := bufio.NewScanner(file)

	book := ""
	chapter := ""

	// deduplicate (book, chapter, verse) and keep the first occurrence
	seen := make(map[string]struct{})

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// detect gospel line
		if after, ok := strings.CutPrefix(line, "Tin mừng:"); ok {
			ref := strings.TrimSpace(after)
			parts := strings.Fields(ref)
			if len(parts) > 0 {
				book = parts[0]
			} else {
				book = ""
			}
			chapter = extractChapter(ref)
			continue
		}

		m := verseRE.FindStringSubmatch(line)
		if m != nil {
			verse := m[1]
			text := m[2]
			text = strings.ReplaceAll(text, "\t", " ")
			key := fmt.Sprintf("%s\t%s\t%s", book, chapter, verse)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			fmt.Fprintf(writer, "%s\t%s\n", key, text)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
