package constants

const (
	DBPath     = "build/bible.db"
	ServerAddr = ":8080"
)

// crawler constants
const (
	Workers        = 1
	Progress       = 1
	OutFilename    = "build/gospels.txt"
	LinkFile       = "build/bible-links.txt"
	ProcessedFile  = "build/processed.txt"
	MissingVerseF  = "build/missing_verse_number.txt"
	OutTsvFilename = "build/gospels.tsv"
)

const (
	VaticanPrefix = "https://www.vaticannews.va/vi/loi-chua-hang-ngay/"
	SitemapURL    = "https://www.vaticannews.va/sitemap.vi.xml"
)
