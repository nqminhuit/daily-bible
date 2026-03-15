package main

import (
	"bufio"
	"fmt"
	cst "github.com/minh/daily-bible/internal/constants"
	h "html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 256*1024)
		return &b
	},
}

var checked int64
var matched int64
var missingVerse int64
var verseLine = regexp.MustCompile(`\s*\{\{(\d+)\}\}`)
var verseHTML = regexp.MustCompile(`<sup[^>]*>\s*(?:<[^>]+>\s*)*(\d+)\s*(?:</[^>]+>\s*)*</sup>`)

func wrapVerseHTML(s string) string {
	return verseHTML.ReplaceAllString(s, "{{$1}}")
}

func loadLinks(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var links []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			links = append(links, url)
		}
	}

	return links, scanner.Err()
}

func loadProcessed(filename string) map[string]bool {
	done := map[string]bool{}

	f, err := os.Open(filename)
	if err != nil {
		return done
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		done[scanner.Text()] = true
	}

	return done
}

func cutBeforeSuyNiem(s string) string {
	before, _, ok := strings.Cut(s, "<h2")
	if !ok {
		return s
	}
	return before
}

func stripHtmlTags(s string) string {
	var b strings.Builder
	inTag := false

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteByte(s[i])
			}
		}
	}
	return b.String()
}

func cleanText(s string) string {
	s = h.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00A0", " ")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	s = wrapVerseHTML(s)
	s = verseLine.ReplaceAllString(s, "\n{{$1}} ")

	s = stripHtmlTags(s)

	lines := strings.Split(s, "\n")
	var out []string

	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}

	return strings.Join(out, "\n")
}
func extractH1(html string) string {
	start := strings.Index(html, "<h1")
	if start == -1 {
		return ""
	}

	start = strings.Index(html[start:], ">") + start + 1
	end := strings.Index(html[start:], "</h1>")
	if end == -1 {
		return ""
	}

	return strings.TrimSpace(html[start : start+end])
}

func extractArticleDetail(html string) string {
	start := strings.Index(html, `<div class="article-detail"`)
	if start == -1 {
		return ""
	}

	// move to end of start tag
	start = strings.Index(html[start:], ">") + start + 1

	depth := 1
	i := start

	for i < len(html) {
		if strings.HasPrefix(html[i:], "<div") {
			depth++
		} else if strings.HasPrefix(html[i:], "</div>") {
			depth--
			if depth == 0 {
				return html[start:i]
			}
		}
		i++
	}

	return ""
}

func worker(
	client *http.Client,
	jobs <-chan string,
	results chan<- string,
	done chan<- string,
	missing chan<- string,
	wg *sync.WaitGroup,
	total int) {
	defer wg.Done()

	for url := range jobs {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			continue
		}

		buf := bufPool.Get().(*[]byte)
		*buf = (*buf)[:0]
		*buf, err = io.ReadAll(io.LimitReader(resp.Body, 10<<20))
		resp.Body.Close()
		if err != nil {
			bufPool.Put(buf)
			continue
		}

		html := string(*buf)
		bufPool.Put(buf)
		if !strings.Contains(html, "Tin mừng:") {
			continue
		}

		title := extractH1(html)

		article := extractArticleDetail(html)
		article = cutBeforeSuyNiem(article)
		if article == "" {
			continue
		}

		if idx := strings.Index(article, "Tin mừng:"); idx != -1 {
			content := article[idx:]
			content = cleanText(content)
			var b strings.Builder
			b.WriteString("TITLE: ")
			b.WriteString(h.UnescapeString(title))
			b.WriteString("\nURL: ")
			b.WriteString(url)
			b.WriteString("\n\n")
			b.WriteString(content)
			b.WriteString("\n\n-----------------------\n\n")
			if hasVerseNumber := strings.Contains(content, "{{"); !hasVerseNumber {
				missing <- b.String()
				atomic.AddInt64(&missingVerse, 1)
			} else {
				results <- b.String()
				atomic.AddInt64(&matched, 1)
			}
		}
		done <- url
		time.Sleep(300 * time.Millisecond)
		if c := atomic.AddInt64(&checked, 1); c%cst.Progress == 0 {
			fmt.Printf(
				"Progress: %d / %d (%.2f%%) | Matches: %d | Missing Verse markers: %d\n",
				c,
				total,
				float64(c)*100/float64(total),
				atomic.LoadInt64(&matched),
				atomic.LoadInt64(&missingVerse),
			)
		}
	}
}

func processedWriter(ch <-chan string) {
	f, err := os.OpenFile(cst.ProcessedFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for url := range ch {
		fmt.Fprintln(w, url)
		w.Flush()
	}
}

func missingVerseNumWriter(ch <-chan string) {
	f, err := os.OpenFile(cst.MissingVerseF, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for url := range ch {
		fmt.Fprintln(w, url)
		w.Flush()
	}
}

func resultsWriter(ch <-chan string) {
	f, err := os.OpenFile(cst.OutFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	count := 0
	for r := range ch {
		w.WriteString(r)
		count++
		if count%10 == 0 {
			w.Flush()
		}
	}
	w.Flush()
}

func main() {
	startTime := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			DisableCompression:  false,
		},
	}

	jobs := make(chan string, cst.Workers*2)
	results := make(chan string, 100)
	done := make(chan string, 100)
	missing := make(chan string, 100)

	var wg sync.WaitGroup

	go resultsWriter(results)
	go processedWriter(done)
	go missingVerseNumWriter(missing)

	links, err := loadLinks(cst.LinkFile)
	if err != nil {
		panic(err)
	}

	total := len(links)
	processed := loadProcessed(cst.ProcessedFile)
	fmt.Printf("Loaded %d links, %d already processed\n", total, len(processed))

	// start workers
	for range cst.Workers {
		wg.Add(1)
		go worker(client, jobs, results, done, missing, &wg, total)
	}

	// enqueue jobs
	for _, url := range links {
		if !processed[url] {
			jobs <- url
		}
	}

	close(jobs)
	wg.Wait()
	close(results)
	close(missing)
	close(done)

	fmt.Println()
	fmt.Println("Crawl finished")
	fmt.Println("Checked pages:", checked)
	fmt.Println("Matched pages:", matched)
	fmt.Println("Missing verse number pages:", missingVerse)
	fmt.Println("Time elapsed:", time.Since(startTime))
}
