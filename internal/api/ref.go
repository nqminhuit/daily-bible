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

type verseRange struct {
	start     int
	startSuffix string
	end       int
	endSuffix   string
}

var refPattern = regexp.MustCompile(`^([A-Za-zÀ-ỹ]{1,3})\s+(\d+),(.+)$`)
var segPattern = regexp.MustCompile(`(\d+)([a-z]?)(?:-(\d+)([a-z]?))?`)

func parseRef(ref string) (book string, chapter int, ranges []verseRange, err error) {
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
		ranges = append(ranges, verseRange{start, startSuffix, end, endSuffix})
	}
	if len(ranges) == 0 {
		return "", 0, nil, fmt.Errorf("no verse ranges in ref %q", ref)
	}
	return
}

func queryByRef(ctx context.Context, db *sql.DB, ref string) ([]model.Gospel, error) {
	book, chapter, ranges, err := parseRef(ref)
	if err != nil {
		return nil, err
	}

	var conditions []string
	var args []any
	args = append(args, book, chapter)
	for _, r := range ranges {
		if r.startSuffix == "" && r.endSuffix == "" {
			conditions = append(conditions, "(verse BETWEEN ? AND ? AND (verse_suffix = '' OR verse_suffix IS NULL))")
			args = append(args, r.start, r.end)
		} else if r.start == r.end {
			if r.startSuffix != "" && r.endSuffix != "" {
				conditions = append(conditions, "(verse = ? AND verse_suffix >= ? AND verse_suffix <= ?)")
				args = append(args, r.start, r.startSuffix, r.endSuffix)
			} else if r.startSuffix != "" {
				conditions = append(conditions, "(verse = ? AND verse_suffix >= ?)")
				args = append(args, r.start, r.startSuffix)
			} else {
				conditions = append(conditions, "(verse = ? AND verse_suffix <= ?)")
				args = append(args, r.start, r.endSuffix)
			}
		} else {
			var innerClauses []string
			var innerArgs []any
			for v := r.start; v <= r.end; v++ {
				if v == r.start && r.startSuffix != "" {
					innerClauses = append(innerClauses, "(verse = ? AND verse_suffix >= ?)")
					innerArgs = append(innerArgs, v, r.startSuffix)
				} else if v == r.end && r.endSuffix != "" {
					innerClauses = append(innerClauses, "(verse = ? AND verse_suffix <= ?)")
					innerArgs = append(innerArgs, v, r.endSuffix)
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
