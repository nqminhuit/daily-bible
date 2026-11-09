package scripture

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var referenceNumber = regexp.MustCompile(`[0-9]+`)

// NormalizeGospelRef uses the shared bounded parser and canonical Gospel bounds.
func NormalizeGospelRef(ref string) (string, error) {
	ref = strings.Join(strings.Fields(strings.NewReplacer("–", "-", "—", "-").Replace(ref)), " ")
	p, err := ParseRefMultiChapter(ref)
	if err != nil {
		return "", err
	}
	chapters, ok := gospelChapters[p[0].Book]
	if !ok {
		return "", fmt.Errorf("not a Gospel reference: %q", ref)
	}
	// No chapter or verse in a Gospel exceeds 80. Check explicit tokens so an
	// explicit MaxInt endpoint cannot masquerade as an open chapter sentinel.
	for _, token := range referenceNumber.FindAllString(ref, -1) {
		n, err := strconv.Atoi(token)
		if err != nil || n > 80 {
			return "", fmt.Errorf("invalid Gospel number %q", token)
		}
	}
	for _, segment := range p[0].Segments {
		if segment.Chapter > len(chapters) {
			return "", fmt.Errorf("invalid Gospel chapter")
		}
		last := chapters[segment.Chapter-1]
		for _, span := range segment.Ranges {
			if span.Start > last || (span.End != EndOfChapter && span.End > last) {
				return "", fmt.Errorf("verse exceeds Gospel chapter")
			}
		}
	}
	return ref, nil
}

func Equivalent(a, b string) bool {
	a, err := NormalizeGospelRef(a)
	if err != nil {
		return false
	}
	b, err = NormalizeGospelRef(b)
	if err != nil {
		return false
	}
	pa, _ := ParseRefMultiChapter(a)
	pb, _ := ParseRefMultiChapter(b)
	ja, _ := json.Marshal(pa)
	jb, _ := json.Marshal(pb)
	return string(ja) == string(jb)
}

// HasText checks structural coverage: empty text, endpoints, gaps and chapter tails.
// It cannot detect an already-truncated legacy base-verse string; source audits
// must validate passage content. It does not download text or hold transactions.
func HasText(ctx context.Context, db *sql.DB, ref string) (bool, error) {
	ref, err := NormalizeGospelRef(ref)
	if err != nil {
		return false, err
	}
	p, err := ParseRefMultiChapter(ref)
	if err != nil {
		return false, err
	}
	rows, err := QueryParsedRef(ctx, db, p)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	for _, row := range rows {
		if strings.TrimSpace(row.Text) == "" {
			return false, nil
		}
	}
	for _, book := range p {
		for _, segment := range book.Segments {
			for _, span := range segment.Ranges {
				end := span.End
				if end == EndOfChapter {
					end = gospelChapters[book.Book][segment.Chapter-1]
				}
				seen := map[int]bool{}
				startSuffix, endSuffix := false, false
				parts := map[string]bool{}
				for _, row := range rows {
					if row.Book == book.Book && row.Chapter == segment.Chapter && row.Verse >= span.Start && row.Verse <= end && strings.TrimSpace(row.Text) != "" {
						seen[row.Verse] = true
						if row.Verse == span.Start {
							parts[row.VerseSuffix] = true
						}
						if row.Verse == span.Start && (row.VerseSuffix == "" || row.VerseSuffix == span.StartSuffix) {
							startSuffix = true
						}
						if row.Verse == end && (row.VerseSuffix == "" || row.VerseSuffix == span.EndSuffix) {
							endSuffix = true
						}
					}
				}
				if span.Start == end && span.StartSuffix != "" && span.EndSuffix != "" && !parts[""] {
					for letter := span.StartSuffix[0]; letter <= span.EndSuffix[0]; letter++ {
						if !parts[string(letter)] {
							return false, nil
						}
					}
				}
				if !seen[span.Start] {
					return false, nil
				}
				if len(seen) != end-span.Start+1 || (span.StartSuffix != "" && !startSuffix) || (span.EndSuffix != "" && !endSuffix) {
					return false, nil
				}
			}
		}
	}
	return true, nil
}
