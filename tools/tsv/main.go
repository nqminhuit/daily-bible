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
			ref := after
			ref = strings.TrimSpace(ref)
			// Tin mừng: Lc 18,9-14 or Tin mừng: Lc 18, 9-14
			parts := strings.Fields(ref)
			book = parts[0]
			if len(parts) > 1 {
				chapter = strings.Split(parts[1], ",")[0]
			}
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
