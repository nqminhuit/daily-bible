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
var refRE = regexp.MustCompile(`^\s*([A-Za-zÀ-ỹ]{1,12})\s*(\d+)\s*,`)

var canonicalBooks = map[string]struct{}{
	"Mt": {},
	"Mc": {},
	"Lc": {},
	"Ga": {},
}

func canonicalBookCode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "mt", "mát-thêu":
		return "Mt"
	case "mc", "mác-cô":
		return "Mc"
	case "lc", "lu-ca":
		return "Lc"
	case "ga", "gio-an":
		return "Ga"
	default:
		return strings.TrimSpace(raw)
	}
}

func parseReference(ref string) (book, chapter string, ok bool) {
	m := refRE.FindStringSubmatch(ref)
	if m == nil {
		return "", "", false
	}
	book = canonicalBookCode(m[1])
	if _, valid := canonicalBooks[book]; !valid {
		return "", "", false
	}
	return book, m[2], true
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
		if after, ok := strings.CutPrefix(line, "__ref__: "); ok {
			ref := strings.TrimSpace(after)
			var valid bool
			book, chapter, valid = parseReference(ref)
			if !valid {
				book = ""
				chapter = ""
			}
			continue
		}

		m := verseRE.FindStringSubmatch(line)
		if m != nil {
			if book == "" || chapter == "" {
				continue
			}
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
