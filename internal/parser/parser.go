package parser

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

		if IsVerseParagraph(c) {
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
	if i := strings.Index(ls, "tin mừng:"); i != -1 {
		return i
	}
	if i := strings.Index(ls, "lời chúa:"); i != -1 {
		return i
	}
	return -1
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
