package model

// Gospel represents a reading entry.
type Gospel struct {
	Reference    string `json:"reference"`
	Book         string `json:"book"`
	ChapterStart int    `json:"chapter_start"`
	VerseStart   int    `json:"verse_start"`
	ChapterEnd   int    `json:"chapter_end"`
	VerseEnd     int    `json:"verse_end"`
	Text         string `json:"text"`
}
