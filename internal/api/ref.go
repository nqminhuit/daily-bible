package api

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/minh/daily-bible/internal/model"
)

type VerseRange struct {
	Start       int
	StartSuffix string
	End         int
	EndSuffix   string
}

type ChapterSegment struct {
	Chapter int
	Ranges  []VerseRange
}

// crossChapterRe matches "21-19,1" where 19 is the next chapter and 1 is the end verse
var crossChapterRe = regexp.MustCompile(`^(\d+)([a-zA-Z]?)-(\d+),(\d+)([a-zA-Z]?)$`)

var refPattern = regexp.MustCompile(`^([A-Za-zÀ-ỹ]{1,3})\s+(\d+),(.+)$`)
var segPattern = regexp.MustCompile(`(\d+)([a-zA-Z]?)(?:-(\d+)([a-zA-Z]?))?`)

func ParseRef(ref string) (book string, chapter int, ranges []VerseRange, err error) {
	segments, err := ParseRefMultiChapter(strings.TrimSpace(ref))
	if err != nil {
		return "", 0, nil, err
	}
	// For backward compatibility, return the first chapter's data
	return segments[0].Book, segments[0].Segments[0].Chapter, segments[0].Segments[0].Ranges, nil
}

type ParsedRef struct {
	Book     string
	Segments []ChapterSegment
}

func ParseRefMultiChapter(ref string) ([]ParsedRef, error) {
	m := refPattern.FindStringSubmatch(strings.TrimSpace(ref))
	if m == nil {
		return nil, fmt.Errorf("invalid ref: %q", ref)
	}
	book := m[1]
	chapter, err := strconv.Atoi(m[2])
	if err != nil {
		return nil, fmt.Errorf("invalid chapter in ref %q", ref)
	}

	versePart := m[3]

	// Split by dots for non-contiguous segments
	dotParts := strings.Split(versePart, ".")

	var segments []ChapterSegment
	currentChapter := chapter

	for _, part := range dotParts {
		part = strings.TrimSpace(part)
		// Check for cross-chapter range like "21-19,1"
		if cm := crossChapterRe.FindStringSubmatch(part); cm != nil {
			startVerse, _ := strconv.Atoi(cm[1])
			startSuffix := cm[2]
			endChapter, _ := strconv.Atoi(cm[3])
			endVerse, _ := strconv.Atoi(cm[4])
			endSuffix := cm[5]

			// Add range from start verse to end of current chapter (use 999 as max)
			segments = append(segments, ChapterSegment{
				Chapter: currentChapter,
				Ranges:  []VerseRange{{Start: startVerse, StartSuffix: startSuffix, End: 999}},
			})

			// Add range from verse 1 to end verse in the new chapter
			segments = append(segments, ChapterSegment{
				Chapter: endChapter,
				Ranges:  []VerseRange{{Start: 1, End: endVerse, EndSuffix: endSuffix}},
			})
			currentChapter = endChapter
		} else {
			// Normal single-chapter segment
			for _, seg := range segPattern.FindAllStringSubmatch(part, -1) {
				start, _ := strconv.Atoi(seg[1])
				end := start
				startSuffix := seg[2]
				endSuffix := seg[4]
				if seg[3] != "" {
					end, _ = strconv.Atoi(seg[3])
				}

				found := false
				for i := range segments {
					if segments[i].Chapter == currentChapter {
						segments[i].Ranges = append(segments[i].Ranges, VerseRange{
							Start: start, StartSuffix: startSuffix, End: end, EndSuffix: endSuffix,
						})
						found = true
						break
					}
				}
				if !found {
					segments = append(segments, ChapterSegment{
						Chapter: currentChapter,
						Ranges:  []VerseRange{{Start: start, StartSuffix: startSuffix, End: end, EndSuffix: endSuffix}},
					})
				}
			}
		}
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("no verse ranges in ref %q", ref)
	}

	return []ParsedRef{{Book: book, Segments: segments}}, nil
}

func QueryByRef(ctx context.Context, db *sql.DB, ref string) ([]model.Gospel, error) {
	parsed, err := ParseRefMultiChapter(ref)
	if err != nil {
		return nil, err
	}

	var allResults []model.Gospel

	for _, p := range parsed {
		for _, seg := range p.Segments {
			results, err := queryChapterSegment(ctx, db, p.Book, seg)
			if err != nil {
				return nil, err
			}
			allResults = append(allResults, results...)
		}
	}

	if allResults == nil {
		allResults = make([]model.Gospel, 0)
	}
	return allResults, nil
}

func queryChapterSegment(ctx context.Context, db *sql.DB, book string, seg ChapterSegment) ([]model.Gospel, error) {
	var conditions []string
	var args []any
	args = append(args, book, seg.Chapter)

	for _, r := range seg.Ranges {
		if r.StartSuffix == "" && r.EndSuffix == "" {
			conditions = append(conditions, "(verse BETWEEN ? AND ? AND (verse_suffix = '' OR verse_suffix IS NULL))")
			args = append(args, r.Start, r.End)
		} else if r.Start == r.End {
			if r.StartSuffix != "" && r.EndSuffix != "" {
				conditions = append(conditions, "(verse = ? AND verse_suffix >= ? AND verse_suffix <= ?)")
				args = append(args, r.Start, r.StartSuffix, r.EndSuffix)
			} else if r.StartSuffix != "" {
				conditions = append(conditions, "(verse = ? AND verse_suffix >= ?)")
				args = append(args, r.Start, r.StartSuffix)
			} else {
				conditions = append(conditions, "(verse = ? AND verse_suffix <= ?)")
				args = append(args, r.Start, r.EndSuffix)
			}
		} else {
			var innerClauses []string
			var innerArgs []any
			for v := r.Start; v <= r.End; v++ {
				if v == r.Start && r.StartSuffix != "" {
					innerClauses = append(innerClauses, "(verse = ? AND verse_suffix >= ?)")
					innerArgs = append(innerArgs, v, r.StartSuffix)
				} else if v == r.End && r.EndSuffix != "" {
					innerClauses = append(innerClauses, "(verse = ? AND verse_suffix <= ?)")
					innerArgs = append(innerArgs, v, r.EndSuffix)
				} else {
					innerClauses = append(innerClauses, "(verse = ? AND (verse_suffix = '' OR verse_suffix IS NULL))")
					innerArgs = append(innerArgs, v)
				}
			}
			conditions = append(conditions, "("+strings.Join(innerClauses, " OR ")+")")
			args = append(args, innerArgs...)
		}
	}

	query := fmt.Sprintf(`
		SELECT book, chapter, verse, verse_suffix, text FROM verses
		WHERE book = ? AND chapter = ? AND (%s)
		ORDER BY verse, verse_suffix`,
		strings.Join(conditions, " OR "),
	)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.Gospel
	for rows.Next() {
		var g model.Gospel
		var suffix sql.NullString
		if err := rows.Scan(&g.Book, &g.Chapter, &g.Verse, &suffix, &g.Text); err != nil {
			return nil, err
		}
		if suffix.Valid {
			g.VerseSuffix = suffix.String
		}
		results = append(results, g)
	}
	return results, rows.Err()
}
