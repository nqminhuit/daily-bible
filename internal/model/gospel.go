package model

type Gospel struct {
	Book        string `json:"book"`
	Chapter     int    `json:"chapter"`
	Verse       int    `json:"verse"`
	VerseSuffix string `json:"verse_suffix,omitempty"`
	Text        string `json:"text"`
}
