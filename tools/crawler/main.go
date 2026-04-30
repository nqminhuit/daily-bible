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
var failed int64
var verseLine = regexp.MustCompile(`\s*\{\{(\d+)\}\}`)
var verseHTML = regexp.MustCompile(`<sup[^>]*>\s*(?:<[^>]+>\s*)*(\d+)\s*(?:</[^>]+>\s*)*</sup>`)

// workerSleep is the time the worker pauses between requests. Overridable in tests.
var workerSleep = 300 * time.Millisecond

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
	return loadURLSet(filename)
}

func loadFailed(filename string) map[string]bool {
	return loadURLSet(filename)
}

func filterPendingURLs(urls []string, processed, failed map[string]bool) []string {
	pending := make([]string, 0, len(urls))
	for _, url := range urls {
		if processed[url] || failed[url] {
			continue
		}
		pending = append(pending, url)
	}
	return pending
}

func loadURLSet(filename string) map[string]bool {
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

func writeURLLineAndFlush(w *bufio.Writer, url, label string) {
	if _, err := fmt.Fprintln(w, url); err != nil {
		log.Printf("Failed writing %s entry for URL %s: %v\n", label, url, err)
		return
	}
	if err := w.Flush(); err != nil {
		log.Printf("Failed flushing %s writer: %v\n", label, err)
	}
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
	failedURLs chan<- string,
	wg *sync.WaitGroup,
	total int) {
	defer wg.Done()

	for url := range jobs {
		func() {
			defer func() {
				c := atomic.AddInt64(&checked, 1)
				if total <= 0 || c%cst.Progress != 0 {
					return
				}
				log.Printf(
					"Progress: %d / %d (%.2f%%) | Matches: %d | Missing Verse markers: %d | Failed: %d\n",
					c,
					total,
					float64(c)*100/float64(total),
					atomic.LoadInt64(&matched),
					atomic.LoadInt64(&missingVerse),
					atomic.LoadInt64(&failed),
				)
			}()

			resp, err := client.Get(url)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				atomic.AddInt64(&failed, 1)
				return
			}

			buf := bufPool.Get().(*[]byte)
			*buf = (*buf)[:0]
			*buf, err = io.ReadAll(io.LimitReader(resp.Body, 10<<20))
			resp.Body.Close()
			if err != nil {
				bufPool.Put(buf)
				atomic.AddInt64(&failed, 1)
				return
			}

			html := string(*buf)
			bufPool.Put(buf)

			// Vatican-only parsing: require Vatican markers and paragraph extraction.
			if idx := findReadingStartVatican(html); idx == -1 {
				log.Printf("Skipping URL (no Vatican markers found): %s\n", url)
				atomic.AddInt64(&failed, 1)
				if failedURLs != nil {
					failedURLs <- url
				}
				return
			}
			article, ref, err := ExtractGospel(html)
			if err != nil {
				log.Printf("Failed to extract gospel from URL %s: %v\n", url, err)
				atomic.AddInt64(&failed, 1)
				if failedURLs != nil {
					failedURLs <- url
				}
				return
			}
			idx2 := findReadingStartVatican(article)
			if idx2 == -1 {
				log.Printf("Skipping URL (no Vatican markers found in article): %s\n", url)
				atomic.AddInt64(&failed, 1)
				if failedURLs != nil {
					failedURLs <- url
				}
				return
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
		}()
	}
}

func processedWriter(filename string, ch <-chan string) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for url := range ch {
		writeURLLineAndFlush(w, url, "processed")
	}
}

func missingVerseNumWriter(filename string, ch <-chan string) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for url := range ch {
		writeURLLineAndFlush(w, url, "missing")
	}
}

func failedWriter(filename string, ch <-chan string) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for url := range ch {
		writeURLLineAndFlush(w, url, "failed")
	}
}

func resultsWriter(filename string, ch <-chan string) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	count := 0
	for r := range ch {
		if _, err := w.WriteString(r); err != nil {
			log.Printf("Failed writing crawler result payload: %v\n", err)
			continue
		}
		count++
		if count%10 == 0 {
			if err := w.Flush(); err != nil {
				log.Printf("Failed flushing crawler results writer: %v\n", err)
			}
		}
	}
	if err := w.Flush(); err != nil {
		log.Printf("Failed final flush for crawler results writer: %v\n", err)
	}
}

func writeLinksToFile(filename string, links []string) error {
	out, err := os.Create(filename)
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
		if _, err := writer.WriteString(m + "\n"); err != nil {
			return fmt.Errorf("failed to write link %q: %w", m, err)
		}
	}
	return writer.Flush()
}

func runCrawler(totalUrls int, outFile, processedPath, failedPath, missingPath, sitemapURL, prefix string) error {
	atomic.StoreInt64(&checked, 0)
	atomic.StoreInt64(&matched, 0)
	atomic.StoreInt64(&missingVerse, 0)
	atomic.StoreInt64(&failed, 0)

	startTime := time.Now()

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
	failedCh := make(chan string, 100)

	var workerWG sync.WaitGroup
	var writerWG sync.WaitGroup

	writerWG.Add(4)
	go func() {
		defer writerWG.Done()
		resultsWriter(outFile, results)
	}()
	go func() {
		defer writerWG.Done()
		processedWriter(processedPath, done)
	}()
	go func() {
		defer writerWG.Done()
		missingVerseNumWriter(missingPath, missing)
	}()
	go func() {
		defer writerWG.Done()
		failedWriter(failedPath, failedCh)
	}()

	// Fetch sitemap and build links
	urls, err := FetchSitemapAndParse(totalUrls, sitemapURL, prefix, client)
	if err != nil {
		return fmt.Errorf("fetch and parse sitemap: %w", err)
	}
	if err := writeLinksToFile(cst.LinkFile, urls); err != nil {
		return fmt.Errorf("write links file: %w", err)
	}

	urls, err = loadLinks(cst.LinkFile)
	if err != nil {
		return fmt.Errorf("load links file: %w", err)
	}

	total := len(urls)
	processed := loadProcessed(processedPath)
	failedBefore := loadFailed(failedPath)
	toProcess := filterPendingURLs(urls, processed, failedBefore)
	log.Printf(
		"Loaded %d links, %d already processed, %d previously failed; %d remaining\n",
		total,
		len(processed),
		len(failedBefore),
		len(toProcess),
	)

	// start workers
	for range cst.Workers {
		workerWG.Add(1)
		go worker(client, jobs, results, done, missing, failedCh, &workerWG, len(toProcess))
	}

	// enqueue jobs
	for _, url := range toProcess {
		jobs <- url
	}

	close(jobs)
	workerWG.Wait()
	close(results)
	close(missing)
	close(done)
	close(failedCh)
	writerWG.Wait()

	log.Println()
	log.Println("Crawl finished")
	log.Println("Checked pages:", checked)
	log.Println("Matched pages:", matched)
	log.Println("Missing verse number pages:", missingVerse)
	log.Println("Failed pages:", failed)
	log.Println("Time elapsed:", time.Since(startTime))
	return nil
}

func main() {
	var totalUrls int
	outFile := cst.OutFilename
	processedPath := cst.ProcessedFile
	failedPath := cst.FailedFile
	missingPath := cst.MissingVerseF
	sitemapURL := cst.SitemapURL
	prefix := cst.VaticanPrefix

	flag.IntVar(&totalUrls, "totalUrls", 0, "if >0, process N latest URLs from the sitemap")
	flag.StringVar(&outFile, "out", outFile, "output gospels file")
	flag.StringVar(&processedPath, "processed", processedPath, "processed URLs file")
	flag.StringVar(&failedPath, "failed", failedPath, "failed URLs file")
	flag.StringVar(&missingPath, "missing", missingPath, "missing verse file")
	flag.Parse()

	if err := runCrawler(totalUrls, outFile, processedPath, failedPath, missingPath, sitemapURL, prefix); err != nil {
		log.Fatalf("crawl failed: %v", err)
	}
}
