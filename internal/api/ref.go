package api

import (
	"context"
	"database/sql"

	"github.com/minh/daily-bible/internal/model"
	"github.com/minh/daily-bible/internal/scripture"
)

// Compatibility aliases keep existing API callers on the shared parser.
type VerseRange = scripture.VerseRange
type ChapterSegment = scripture.ChapterSegment
type ParsedRef = scripture.ParsedRef

const maxRefLength = scripture.MaxRefLength
const maxRefSegments = scripture.MaxRefSegments

func ParseRef(ref string) (string, int, []VerseRange, error) {
	return scripture.ParseRef(ref)
}

func ParseRefMultiChapter(ref string) ([]ParsedRef, error) {
	return scripture.ParseRefMultiChapter(ref)
}

func QueryByRef(ctx context.Context, db *sql.DB, ref string) ([]model.Gospel, error) {
	return scripture.QueryByRef(ctx, db, ref)
}

func queryParsedRef(ctx context.Context, db *sql.DB, parsed []ParsedRef) ([]model.Gospel, error) {
	return scripture.QueryParsedRef(ctx, db, parsed)
}
