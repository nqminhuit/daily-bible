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

var refPattern = regexp.MustCompile(`^([A-Za-zÀ-ỹ]{1,3})\s+(\d+),(.+)$`)
var segPattern = regexp.MustCompile(`(\d+)([a-zA-Z]?)(?:-(\d+)([a-zA-Z]?))?`)

func ParseRef(ref string) (book string, chapter int, ranges []VerseRange, err error) {
	m := refPattern.FindStringSubmatch(strings.TrimSpace(ref))
	if m == nil {
		return "", 0, nil, fmt.Errorf("invalid ref: %q", ref)
	}
	book = m[1]
	chapter, err = strconv.Atoi(m[2])
	if err != nil {
		return "", 0, nil, fmt.Errorf("invalid chapter in ref %q", ref)
	}

	for _, seg := range segPattern.FindAllStringSubmatch(m[3], -1) {
		start, _ := strconv.Atoi(seg[1])
		end := start
		startSuffix := seg[2]
		endSuffix := seg[4]
		if seg[3] != "" {
			end, _ = strconv.Atoi(seg[3])
		}
		ranges = append(ranges, VerseRange{Start: start, StartSuffix: startSuffix, End: end, EndSuffix: endSuffix})
	}
	if len(ranges) == 0 {
		return "", 0, nil, fmt.Errorf("no verse ranges in ref %q", ref)
	}
	return
}

func QueryByRef(ctx context.Context, db *sql.DB, ref string) ([]model.Gospel, error) {
	book, chapter, ranges, err := ParseRef(ref)
	if err != nil {
		return nil, err
	}

	var conditions []string
	var args []any
	args = append(args, book, chapter)
	for _, r := range ranges {
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

	results := make([]model.Gospel, 0)
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
