package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	cst "github.com/minh/daily-bible/internal/constants"
)

var verseRE = regexp.MustCompile(`\{\{(\d+[A-Za-z]?)\}\}\s*(.*)`)
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
	if err := convert(cst.OutFilename, cst.OutTsvFilename); err != nil {
		panic(err)
	}
}

// convert reads gospels from inputPath and writes TSV to outputPath.
func convert(inputPath, outputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	defer writer.Flush()
	scanner := bufio.NewScanner(file)

	book := ""
	chapter := ""

	seen := make(map[string]struct{})

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
			verseRaw := m[1]
			text := m[2]
			text = strings.ReplaceAll(text, "\t", " ")

			verseNum := ""
			verseSuffix := ""
			for _, r := range verseRaw {
				if r >= '0' && r <= '9' {
					verseNum += string(r)
				} else {
					verseSuffix += string(r)
				}
			}

			key := fmt.Sprintf("%s\t%s\t%s\t%s", book, chapter, verseNum, verseSuffix)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			fmt.Fprintf(writer, "%s\t%s\n", key, text)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
