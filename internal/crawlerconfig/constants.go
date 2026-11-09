// Package crawlerconfig defines Vatican News crawling defaults and artifact
// paths shared by the crawler and TSV converter. Runtime application defaults
// belong in internal/constants instead.
package crawlerconfig

const (
	Workers        = 1
	Progress       = 1
	OutFilename    = "build/gospels.txt"
	LinkFile       = "build/bible-links.txt"
	ProcessedFile  = "build/processed.txt"
	FailedFile     = "build/failed.txt"
	MissingVerseF  = "build/missing_verse_number.txt"
	OutTsvFilename = "build/gospels.tsv"
)

const (
	VaticanPrefix = "https://www.vaticannews.va/vi/loi-chua-hang-ngay/"
	SitemapURL    = "https://www.vaticannews.va/sitemap.vi.xml"
)
