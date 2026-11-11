package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var bibleRefRe = regexp.MustCompile(`((?:[1-3]\s*)?[A-Za-zÀ-ỹ]{1,10}\s*\d+\s*,\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?(?:\s*,\s*\d+[A-Za-z]?)?)?(?:\s*\.\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?(?:\s*,\s*\d+[A-Za-z]?)?)?)*)`)
var pageDateRe = regexp.MustCompile(`ngày\s+(\d{1,2})\s+tháng\s+(\d{1,2})\s+(\d{4})`)
var chapterVerseRe = regexp.MustCompile(`(\d+\s*,\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?(?:\s*,\s*\d+[A-Za-z]?)?)?(?:\s*\.\s*\d+[A-Za-z]?(?:\s*-\s*\d+[A-Za-z]?(?:\s*,\s*\d+[A-Za-z]?)?)?)*)`)
var bookChapterSplitRe = regexp.MustCompile(`^((?:[1-3]\s*)?[A-Za-zÀ-ỹ]{1,10})\s*(\d.*)$`)
var commaSpacingRe = regexp.MustCompile(`\s*,\s*`)
var dashSpacingRe = regexp.MustCompile(`\s*-\s*`)
var dotSpacingRe = regexp.MustCompile(`\s*\.\s*`)

func FindLastNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.LastChild; c != nil; c = c.PrevSibling {
		if res := FindLastNode(c, match); res != nil {
			return res
		}
	}
	return nil
}

func HasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" && strings.Contains(attr.Val, class) {
			return true
		}
	}
	return false
}

func GetText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(GetText(c))
	}
	return b.String()
}

func FindNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if res := FindNode(c, match); res != nil {
			return res
		}
	}
	return nil
}

func extractGospelRef(doc *html.Node) (string, error) {
	content := FindSectionContent(doc)
	if content == nil {
		return "", fmt.Errorf("div with class 'section__content' not found")
	}

	p := FindNode(content, func(n *html.Node) bool {
		if n.Type != html.ElementNode || n.Data != "p" {
			return false
		}
		return IsGospelHeaderText(GetText(n))
	})

	if p == nil {
		p = FindNode(doc, func(n *html.Node) bool {
			if n.Type != html.ElementNode || n.Data != "p" {
				return false
			}
			return IsGospelHeaderText(GetText(n))
		})
	}
	if p == nil {
		return "", fmt.Errorf("paragraph containing 'Tin Mừng' not found in content")
	}

	text := NormalizeSpaces(GetText(p))
	if match := bibleRefRe.FindString(text); match != "" {
		return canonicalizeReference(match), nil
	}
	if book := inferBookFromHeader(text); book != "" {
		if cv := chapterVerseRe.FindString(text); cv != "" {
			return canonicalizeReference(book + " " + cv), nil
		}
		// Some Vatican pages put the reference in a separate Gospel label,
		// followed by at most one summary paragraph and the liturgical header.
		// Stay in this sibling block: never borrow a first-reading/acclamation
		// reference, another Gospel's book, or an unrelated distant paragraph.
		count := 0
		for prev := p.PrevSibling; prev != nil && count < 2; prev = prev.PrevSibling {
			if prev.Type == html.TextNode && strings.TrimSpace(prev.Data) == "" {
				continue
			}
			if prev.Type != html.ElementNode || prev.Data != "p" {
				break
			}
			count++
			label := NormalizeSpaces(GetText(prev))
			if !strings.HasPrefix(label, "Tin Mừng ") && !strings.HasPrefix(label, "Tin Mừng:") {
				continue
			}
			raw := strings.TrimSpace(strings.TrimPrefix(label, "Tin Mừng"))
			raw = strings.TrimSpace(strings.TrimPrefix(raw, ":"))
			match := bibleRefRe.FindString(raw)
			ref := canonicalizeReference(match)
			if match != "" && match == raw && strings.HasPrefix(ref, book+" ") {
				return ref, nil
			}
			return "", fmt.Errorf("separate Gospel reference disagrees with header: %q", label)
		}
	}
	return "", fmt.Errorf("no Bible reference found in text: %q", text)
}

func IsVerseParagraph(p *html.Node) bool {
	for c := p.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "sup" {
			return true
		}
	}
	return false
}

// continuesVerse reports an unnumbered paragraph (verse continuation, e.g. quoted
// poetry) directly followed by a numbered one. A trailing unnumbered paragraph is not taken.
func continuesVerse(p *html.Node) bool {
	if strings.TrimSpace(GetText(p)) == "" || IsGospelHeader(p) {
		return false
	}
	for next := p.NextSibling; next != nil; next = next.NextSibling {
		if next.Type == html.ElementNode {
			return next.Data == "p" && IsVerseParagraph(next)
		}
	}
	return false
}

func IsGospelHeader(p *html.Node) bool {
	return IsGospelHeaderText(GetText(p))
}

func ExtractGospelSection(content *html.Node) string {
	var b strings.Builder
	started := false

	for c := content.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data != "p" {
			continue
		}

		if !started {
			if IsGospelHeader(c) {
				started = true
				html.Render(&b, c)
			}
			continue
		}

		if IsVerseParagraph(c) || continuesVerse(c) {
			html.Render(&b, c)
			continue
		}

		break
	}

	return b.String()
}

func FindSectionContent(doc *html.Node) *html.Node {
	section := FindNode(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "section" {
			return strings.Contains(GetText(n), "Tin Mừng ngày hôm nay")
		}
		return false
	})
	if section != nil {
		if content := FindNode(section, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "div" && HasClass(n, "section__content")
		}); content != nil {
			return content
		}
	}
	return FindLastNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && HasClass(n, "section__content")
	})
}

func NormalizeSpaces(s string) string {
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\u2013", "-") // en-dash
	s = strings.ReplaceAll(s, "\u2014", "-") // em-dash
	return strings.Join(strings.Fields(s), " ")
}

func canonicalizeReference(ref string) string {
	ref = NormalizeSpaces(ref)
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

func IsGospelHeaderText(text string) bool {
	txt := NormalizeSpaces(text)
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

func extract(htmlStr string) (main, ref string, err error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	ref, err = extractGospelRef(doc)
	if err != nil {
		return "", "", fmt.Errorf("failed to extract gospel reference: %w", err)
	}

	if div := FindLastNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && HasClass(n, "section__content")
	}); div != nil {
		main = ExtractGospelSection(div)
		return
	}
	return "", "", fmt.Errorf("div with class 'section__content' not found")
}

func FindReadingStartVatican(s string) int {
	ls := strings.ToLower(s)
	if i := strings.Index(ls, "tin mừng"); i != -1 {
		return i
	}
	if i := strings.Index(ls, "lời chúa"); i != -1 {
		return i
	}
	return -1
}

// ExtractPageDate reads the date the page itself claims to be for (from its
// title, e.g. "... ngày 20 tháng 8 2026 ...") and returns it as "YYYY-MM-DD".
// This lets callers verify a fetched page actually covers the date they
// requested, rather than trusting the request URL alone.
func ExtractPageDate(htmlStr string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return "", fmt.Errorf("parse page: %w", err)
	}
	title := FindNode(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "title" })
	if title == nil {
		return "", fmt.Errorf("page has no title date")
	}
	m := pageDateRe.FindStringSubmatch(strings.ToLower(NormalizeSpaces(GetText(title))))
	if m == nil {
		return "", fmt.Errorf("no date found in page title")
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[3])
	date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", fmt.Errorf("invalid page date: %w", err)
	}
	return date, nil
}

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
