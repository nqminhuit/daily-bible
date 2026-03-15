package constants

const (
	DBPath       = "build/bible.db"
	ServerAddr   = ":8080"
)

// crawler constants
const (
	Workers        = 1
	Progress       = 2
	OutFilename    = "build/gospels.txt"
	LinkFile       = "build/bible-links.txt"
	ProcessedFile  = "build/processed.txt"
	MissingVerseF  = "build/missing_verse_number.txt"
	OutTsvFilename = "build/gospels.tsv"
)
