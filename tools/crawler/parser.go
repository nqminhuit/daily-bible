package main

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var bibleRefRe = regexp.MustCompile(`((?:[1-3]\s*)?[A-Za-zÀ-ỹ]{1,10}\s*\d+\s*,\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?)?(?:\s*\.\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?)?)*)`)
var chapterVerseRe = regexp.MustCompile(`(\d+\s*,\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?)?(?:\s*\.\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?)?)*)`)
var bookChapterSplitRe = regexp.MustCompile(`^((?:[1-3]\s*)?[A-Za-zÀ-ỹ]{1,10})\s*(\d.*)$`)
var commaSpacingRe = regexp.MustCompile(`\s*,\s*`)
var dashSpacingRe = regexp.MustCompile(`\s*-\s*`)
var dotSpacingRe = regexp.MustCompile(`\s*\.\s*`)

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
	content := findSectionContent(doc)
	if content == nil {
		return "", fmt.Errorf("div with class 'section__content' not found")
	}

	// 3. find <p> containing the gospel reference line
	p := findNode(content, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "p" {
			return false
		}
		return isGospelHeaderText(getText(n))
	})

	if p == nil {
		// Fallback for templates where the header is outside the section__content.
		p = findNode(doc, func(n *html.Node) bool {
			if n.Type != html.ElementNode || n.Data != "p" {
				return false
			}
			return isGospelHeaderText(getText(n))
		})
	}
	if p == nil {
		return "", fmt.Errorf("paragraph containing 'Tin Mừng' not found in content")
	}

	// 4. extract Bible reference from that paragraph
	text := normalizeSpaces(getText(p))
	if match := bibleRefRe.FindString(text); match != "" {
		return canonicalizeReference(match), nil
	}
	// Some pages omit book abbreviation in the reference (e.g. "2,1-12") while
	// still stating the evangelist in the same header line.
	if book := inferBookFromHeader(text); book != "" {
		if cv := chapterVerseRe.FindString(text); cv != "" {
			return canonicalizeReference(book + " " + cv), nil
		}
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
	return isGospelHeaderText(getText(p))
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

func findSectionContent(doc *html.Node) *html.Node {
	section := findNode(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "section" {
			return strings.Contains(getText(n), "Tin Mừng ngày hôm nay")
		}
		return false
	})
	if section != nil {
		if content := findNode(section, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "section__content")
		}); content != nil {
			return content
		}
	}
	return findLastNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "section__content")
	})
}

func normalizeSpaces(s string) string {
	s = strings.ReplaceAll(s, "\u00A0", " ")
	return strings.Join(strings.Fields(s), " ")
}

func canonicalizeReference(ref string) string {
	ref = normalizeSpaces(ref)
	if parts := bookChapterSplitRe.FindStringSubmatch(ref); parts != nil {
		ref = strings.TrimSpace(parts[1]) + " " + strings.TrimSpace(parts[2])
	}
	ref = commaSpacingRe.ReplaceAllString(ref, ",")
	ref = dashSpacingRe.ReplaceAllString(ref, "-")
	ref = dotSpacingRe.ReplaceAllString(ref, ".")
	return ref
}

func inferBookFromHeader(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "mát-thêu"):
		return "Mt"
	case strings.Contains(lower, "mác-cô"):
		return "Mc"
	case strings.Contains(lower, "lu-ca"):
		return "Lc"
	case strings.Contains(lower, "gio-an"):
		return "Ga"
	default:
		return ""
	}
}

func isGospelHeaderText(text string) bool {
	txt := normalizeSpaces(text)
	if txt == "" {
		return false
	}
	if !strings.Contains(txt, "Tin Mừng") || !strings.Contains(txt, "theo thánh") {
		return false
	}
	return strings.HasPrefix(txt, "✠") ||
		strings.HasPrefix(txt, "Tin Mừng") ||
		strings.HasPrefix(txt, "Khởi đầu Tin Mừng") ||
		strings.HasPrefix(txt, "Bài trích Tin Mừng")
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
