package main

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var bibleRefRe = regexp.MustCompile(`([A-Za-zÀ-ỹ]{1,5}\s*\d+,\d+(?:-\d+)?)`)

// find last node matching condition
func findLastNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.LastChild; c != nil; c = c.PrevSibling {
		if res := findLastNode(c, match); res != nil {
			return res
		}
	}
	return nil
}

func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" && strings.Contains(attr.Val, class) {
			return true
		}
	}
	return false
}

// helper: get all text inside node
func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(getText(c))
	}
	return b.String()
}

// find node by condition
func findNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if res := findNode(c, match); res != nil {
			return res
		}
	}
	return nil
}

// main extractor
func extractGospelRef(doc *html.Node) (string, error) {
	// 1. find section with "Tin Mừng ngày hôm nay"
	section := findNode(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "section" {
			text := getText(n)
			return strings.Contains(text, "Tin Mừng ngày hôm nay")
		}
		return false
	})

	if section == nil {
		return "", fmt.Errorf("section with 'Tin Mừng ngày hôm nay' not found")
	}

	// 2. find section__content inside it
	content := findNode(section, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "section__content")
	})

	if content == nil {
		return "", fmt.Errorf("div with class 'section__content' not found in section")
	}

	// 3. find <p> containing the gospel reference line
	p := findNode(content, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "p" {
			return false
		}

		text := strings.TrimSpace(getText(n))
		text = strings.ReplaceAll(text, "\u00A0", " ")

		// must contain "Tin Mừng" AND a ✠ marker (strong signal)
		return strings.Contains(text, "Tin Mừng") && strings.Contains(text, "✠")
	})

	if p == nil {
		return "", fmt.Errorf("paragraph containing 'Tin Mừng' not found in content")
	}

	// 4. extract Bible reference from that paragraph
	text := getText(p)
	text = strings.ReplaceAll(text, "\u00A0", " ") // non-breaking space to regular space
	if match := bibleRefRe.FindString(text); match != "" {
		return match, nil
	}
	return "", fmt.Errorf("no Bible reference found in text: %q", text)
}

func isVerseParagraph(p *html.Node) bool {
	for c := p.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "sup" {
			return true
		}
	}
	return false
}

func isGospelHeader(p *html.Node) bool {
	txt := strings.TrimSpace(getText(p))
	// Strong signals: leading "Tin Mừng", the ✠ marker, or explicit "Tin Mừng Chúa"
	if strings.Contains(txt, "✠") || strings.Contains(txt, "Tin Mừng Chúa") {
		return true
	}
	// header paragraphs often start with "Tin Mừng"
	return strings.HasPrefix(txt, "Tin Mừng")
}

func extractGospelSection(content *html.Node) string {
	var b strings.Builder
	started := false

	for c := content.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "p" {
			continue
		}

		if !started {
			if isGospelHeader(c) {
				started = true
				html.Render(&b, c)
			}
			continue
		}

		// after start
		if isVerseParagraph(c) {
			html.Render(&b, c)
			continue
		}

		// STOP at first non-verse paragraph
		break
	}

	return b.String()
}

// extract finds the <div class="section__content"> block the HTML and Bible reference if present.
// fallback to <main class="content"> if the first block is not found.
// Returns empty strings if not found, or an error if the HTML cannot be parsed.
func extract(htmlStr string) (main, ref string, err error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	ref, err = extractGospelRef(doc)
	if err != nil {
		return "", "", fmt.Errorf("failed to extract gospel reference: %w", err)
	}

	// find: <div class="section__content">
	if div := findLastNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "section__content")
	}); div != nil {
		main = extractGospelSection(div)
		return
	}
	return "", "", fmt.Errorf("div with class 'section__content' not found")
}

// findReadingStartVatican searches for typical starting keywords in lowered text.
func findReadingStartVatican(s string) int {
	ls := strings.ToLower(s)
	if i := strings.Index(ls, "tin mừng"); i != -1 {
		return i
	}
	if i := strings.Index(ls, "lời chúa"); i != -1 {
		return i
	}
	if i := strings.Index(ls, "tin mừng:"); i != -1 {
		return i
	}
	if i := strings.Index(ls, "lời chúa:"); i != -1 {
		return i
	}
	return -1
}

// ExtractGospel extracts the gospel content and reference block (HTML) from a Vatican News article HTML.
func ExtractGospel(htmlInput string) (section, ref string, err error) {
	section, ref, err = extract(htmlInput)
	if err != nil {
		return "", "", fmt.Errorf("failed to extract content: %w", err)
	}
	if section == "" {
		return "", "", fmt.Errorf("page missing Bible content")
	}
	if ref == "" {
		return "", "", fmt.Errorf("page missing Bible reference")
	}
	section = strings.TrimSpace(section)
	ref = strings.TrimSpace(ref)
	return
}
