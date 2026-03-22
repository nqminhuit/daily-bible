package main

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

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
		return n.Type == html.ElementNode &&
			n.Data == "div" &&
			hasClass(n, "section__content")
	})

	if content == nil {
		return "", fmt.Errorf("div with class 'section__content' not found in section")
	}

	// 3. find <p> containing "Tin Mừng"
	p := findNode(content, func(n *html.Node) bool {
		return n.Type == html.ElementNode &&
			n.Data == "p" &&
			strings.Contains(getText(n), "Tin Mừng")
	})

	if p == nil {
		return "", fmt.Errorf("paragraph containing 'Tin Mừng' not found in content")
	}

	// 4. extract last meaningful text node
	var result string
	for c := p.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			text := strings.TrimSpace(c.Data)
			text = strings.ReplaceAll(text, "\u00A0", "")
			if text != "" {
				result = text
			}
		}
	}

	return result, nil
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
	return strings.Contains(getText(p), "Tin Mừng")
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

	// fallback: <main class="content">
	if mainEl := findLastNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode &&
			n.Data == "main" &&
			hasClass(n, "content")
	}); mainEl != nil {
		var b strings.Builder
		html.Render(&b, mainEl)
		main = b.String()
		return
	}

	return
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

	if section == "" || ref == "" {
		return "", "", fmt.Errorf("page missing content or reference")
	}
	section = strings.TrimSpace(section)
	ref = strings.TrimSpace(ref)
	return
}
