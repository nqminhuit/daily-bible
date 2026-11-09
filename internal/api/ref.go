package api

import (
	"context"
	"database/sql"
	"fmt"
	"math"
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
var segPattern = regexp.MustCompile(`^(\d+)([a-zA-Z]?)(?:-(\d+)([a-zA-Z]?))?$`)

// Bound input parsing, generated predicates, and cross-chapter queries. The
// segment budget counts ranges after cross-chapter expansion, not just dots.
const maxRefLength = 1024 // bytes, including surrounding whitespace
const maxRefSegments = 32

// An open-ended chapter range uses the largest representable verse number.
// Unlike the old 999 sentinel, this does not impose an arbitrary verse cutoff.
const endOfChapter = math.MaxInt

func ParseRef(ref string) (book string, chapter int, ranges []VerseRange, err error) {
	segments, err := ParseRefMultiChapter(ref)
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
	if len(ref) > maxRefLength {
		return nil, fmt.Errorf("reference exceeds %d bytes", maxRefLength)
	}
	m := refPattern.FindStringSubmatch(strings.TrimSpace(ref))
	if m == nil {
		return nil, fmt.Errorf("invalid ref: %q", ref)
	}
	chapter, err := positiveRefNumber(m[2])
	if err != nil {
		return nil, fmt.Errorf("invalid chapter: %w", err)
	}
	if strings.Count(m[3], ".") >= maxRefSegments {
		return nil, fmt.Errorf("reference exceeds %d segments", maxRefSegments)
	}

	var segments []ChapterSegment
	rangeCount := 0
	addRange := func(ch int, r VerseRange) error {
		if rangeCount == maxRefSegments {
			return fmt.Errorf("reference exceeds %d segments", maxRefSegments)
		}
		rangeCount++
		if len(segments) == 0 || segments[len(segments)-1].Chapter != ch {
			segments = append(segments, ChapterSegment{Chapter: ch})
		}
		i := len(segments) - 1
		segments[i].Ranges = append(segments[i].Ranges, r)
		return nil
	}

	for _, part := range strings.Split(m[3], ".") {
		part = strings.TrimSpace(part)
		if cm := crossChapterRe.FindStringSubmatch(part); cm != nil {
			start, err := positiveRefNumber(cm[1])
			if err != nil {
				return nil, fmt.Errorf("invalid start verse: %w", err)
			}
			endChapter, err := positiveRefNumber(cm[3])
			if err != nil {
				return nil, fmt.Errorf("invalid end chapter: %w", err)
			}
			end, err := positiveRefNumber(cm[4])
			if err != nil {
				return nil, fmt.Errorf("invalid end verse: %w", err)
			}
			if endChapter <= chapter {
				return nil, fmt.Errorf("cross-chapter range must end in a later chapter")
			}
			// Check before expansion; subtraction is safe for positive ordered ints.
			if endChapter-chapter >= maxRefSegments-rangeCount {
				return nil, fmt.Errorf("reference exceeds %d segments", maxRefSegments)
			}
			for ch := chapter; ch < endChapter; ch++ {
				r := VerseRange{Start: 1, End: endOfChapter}
				if ch == chapter {
					r.Start, r.StartSuffix = start, cm[2]
				}
				if err := addRange(ch, r); err != nil {
					return nil, err
				}
			}
			if err := addRange(endChapter, VerseRange{Start: 1, End: end, EndSuffix: cm[5]}); err != nil {
				return nil, err
			}
			chapter = endChapter
			continue
		}

		seg := segPattern.FindStringSubmatch(part)
		if seg == nil {
			return nil, fmt.Errorf("invalid verse segment: %q", part)
		}
		start, err := positiveRefNumber(seg[1])
		if err != nil {
			return nil, fmt.Errorf("invalid start verse: %w", err)
		}
		end := start
		if seg[3] != "" {
			end, err = positiveRefNumber(seg[3])
			if err != nil {
				return nil, fmt.Errorf("invalid end verse: %w", err)
			}
		}
		if end < start || (end == start && seg[2] != "" && seg[4] != "" && seg[4] < seg[2]) {
			return nil, fmt.Errorf("reversed verse range: %q", part)
		}
		if err := addRange(chapter, VerseRange{Start: start, StartSuffix: seg[2], End: end, EndSuffix: seg[4]}); err != nil {
			return nil, err
		}
	}
	return []ParsedRef{{Book: m[1], Segments: segments}}, nil
}

func positiveRefNumber(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("expected a positive integer, got %q", s)
	}
	return n, nil
}

func QueryByRef(ctx context.Context, db *sql.DB, ref string) ([]model.Gospel, error) {
	parsed, err := ParseRefMultiChapter(ref)
	if err != nil {
		return nil, err
	}

	return queryParsedRef(ctx, db, parsed)
}

// queryParsedRef accepts only the validated output of ParseRefMultiChapter.
func queryParsedRef(ctx context.Context, db *sql.DB, parsed []ParsedRef) ([]model.Gospel, error) {
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
			conditions = append(conditions, "(verse BETWEEN ? AND ? AND verse_suffix = '')")
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
			// Three predicates regardless of span; no per-verse loop or endpoint
			// arithmetic, so even MaxInt endpoints cannot overflow or expand SQL.
			startOp, endOp := "=", "="
			if r.StartSuffix != "" {
				startOp = ">="
			}
			if r.EndSuffix != "" {
				endOp = "<="
			}
			conditions = append(conditions, fmt.Sprintf(`(
				(verse = ? AND verse_suffix %s ?) OR
				(verse > ? AND verse < ? AND verse_suffix = '') OR
				(verse = ? AND verse_suffix %s ?))`, startOp, endOp))
			args = append(args, r.Start, r.StartSuffix, r.Start, r.End, r.End, r.EndSuffix)
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
