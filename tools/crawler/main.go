package main

import (
	"bufio"
	"flag"
	"fmt"
	h "html"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cst "github.com/minh/daily-bible/internal/constants"
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

// workerSleep is the time the worker pauses between requests. Overridable in tests.
var workerSleep = 300 * time.Millisecond

// output file paths (overrideable via flags)
var outFilename = cst.OutFilename
var processedFile = cst.ProcessedFile
var missingFile = cst.MissingVerseF
var sitemapURL = cst.SitemapURL
var prefix = cst.VaticanPrefix

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

		// Vatican-only parsing: require vatican markers and paragraph extraction
		if idx := findReadingStartVatican(html); idx == -1 {
			log.Printf("Skipping URL (no Vatican markers found): %s\n", url)
			continue
		}
		article, ref, err := ExtractGospel(html)
		if err != nil {
			log.Printf("Failed to extract gospel from URL %s: %v\n", url, err)
			continue
		}
		idx2 := findReadingStartVatican(article)
		if idx2 == -1 {
			log.Printf("Skipping URL (no Vatican markers found in article): %s\n", url)
			continue
		}
		content := article[idx2:]
		content = cleanText(content)
		var b strings.Builder
		b.WriteString("-------\n")
		b.WriteString("URL: ")
		b.WriteString(url)
		b.WriteString("\n")
		b.WriteString("__ref__: ")
		b.WriteString(ref)
		b.WriteString("\n")
		b.WriteString(content)
		b.WriteString("\n")
		if hasVerseNumber := strings.Contains(content, "{{"); !hasVerseNumber {
			missing <- b.String()
			atomic.AddInt64(&missingVerse, 1)
		} else {
			results <- b.String()
			atomic.AddInt64(&matched, 1)
		}
		done <- url
		time.Sleep(workerSleep)
		if c := atomic.AddInt64(&checked, 1); c%cst.Progress == 0 {
			log.Printf(
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
	f, err := os.OpenFile(processedFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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
	f, err := os.OpenFile(missingFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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

func writeLinksToFile(links []string) error {
	out, err := os.Create(cst.LinkFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			panic(fmt.Errorf("failed to close file: %w", err))
		}
	}()

	writer := bufio.NewWriter(out)
	for _, m := range links {
		writer.WriteString(m + "\n")
	}
	return writer.Flush()
}

func main() {
	startTime := time.Now()

	var totalUrls int
	flag.IntVar(&totalUrls, "totalUrls", 0, "if >0, process N latest URLs from the sitemap")
	flag.StringVar(&outFilename, "out", outFilename, "output gospels file")
	flag.StringVar(&processedFile, "processed", processedFile, "processed URLs file")
	flag.StringVar(&missingFile, "missing", missingFile, "missing verse file")
	flag.Parse()

	log.Printf("Starting crawl with sitemap: %s, totalUrls: %d\n", sitemapURL, totalUrls)

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

	// Fetch sitemap and build links
	if urls, err := FetchSitemapAndParse(totalUrls, sitemapURL, prefix, client); err != nil {
		log.Fatalf("Failed to fetch and parse sitemap: %v", err)
	} else {
		writeLinksToFile(urls)
	}

	urls, err := loadLinks(cst.LinkFile)
	if err != nil {
		log.Fatalf("Failed to load links: %v", err)
	}

	total := len(urls)
	processed := loadProcessed(processedFile)
	log.Printf("Loaded %d links, %d already processed\n", total, len(processed))

	// start workers
	for range cst.Workers {
		wg.Add(1)
		go worker(client, jobs, results, done, missing, &wg, total)
	}

	// enqueue jobs
	for _, url := range urls {
		if !processed[url] {
			jobs <- url
		}
	}

	close(jobs)
	wg.Wait()
	close(results)
	close(missing)
	close(done)

	log.Println()
	log.Println("Crawl finished")
	log.Println("Checked pages:", checked)
	log.Println("Matched pages:", matched)
	log.Println("Missing verse number pages:", missingVerse)
	log.Println("Time elapsed:", time.Since(startTime))
}
