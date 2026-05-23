package query

import (
	"fmt"
	"strings"
)

// FtsPhraseQuery wraps the query in double quotes for exact phrase matching in FTS5.
// The escaped query is treated as a single phrase; individual token search is not supported.
func FtsPhraseQuery(q string) string {
	escaped := strings.ReplaceAll(q, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}
