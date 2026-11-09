package main

import (
	"bufio"
	"context"
	"errors"
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
	"github.com/minh/daily-bible/internal/parser"
)

var verseLine = regexp.MustCompile(`\s*\{\{(\d+)\}\}`)
var verseHTML = regexp.MustCompile(`<sup[^>]*>\s*(?:<[^>]+>\s*)*(\d+)\s*(?:</[^>]+>\s*)*</sup>`)

type crawlCounters struct {
	checked      int64
	matched      int64
	missingVerse int64
	failed       int64
}

type workerConfig struct {
	client        *http.Client
	jobs          <-chan string
	ctx           context.Context
	persist       func(crawlResult) error
	wg            *sync.WaitGroup
	total         int
	sleepDuration time.Duration
	counters      *crawlCounters
}

const defaultWorkerSleep = 300 * time.Millisecond

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

func writeURLLineAndFlush(w *bufio.Writer, url, label string) error {
	if _, err := fmt.Fprintln(w, url); err != nil {
		return fmt.Errorf("write %s: %w", label, err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush %s: %w", label, err)
	}
	return nil
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

func worker(cfg workerConfig) {
	defer cfg.wg.Done()

	ctx := cfg.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		var url string
		select {
		case <-ctx.Done():
			return
		case next, ok := <-cfg.jobs:
			if !ok || ctx.Err() != nil {
				return
			}
			url = next
		}
		func() {
			defer func() {
				c := atomic.AddInt64(&cfg.counters.checked, 1)
				if cfg.total <= 0 || c%cst.Progress != 0 {
					return
				}
				log.Printf(
					"Progress: %d / %d (%.2f%%) | Matches: %d | Missing Verse markers: %d | Failed: %d\n",
					c,
					cfg.total,
					float64(c)*100/float64(cfg.total),
					atomic.LoadInt64(&cfg.counters.matched),
					atomic.LoadInt64(&cfg.counters.missingVerse),
					atomic.LoadInt64(&cfg.counters.failed),
				)
			}()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				atomic.AddInt64(&cfg.counters.failed, 1)
				return
			}
			resp, err := cfg.client.Do(req)
			if err != nil {
				atomic.AddInt64(&cfg.counters.failed, 1)
				return
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				atomic.AddInt64(&cfg.counters.failed, 1)
				return
			}

			body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
			resp.Body.Close()
			if err != nil {
				atomic.AddInt64(&cfg.counters.failed, 1)
				return
			}

			html := string(body)

			// Vatican-only parsing: require Vatican markers and paragraph extraction.
			if idx := parser.FindReadingStartVatican(html); idx == -1 {
				log.Printf("Skipping URL (no Vatican markers found): %s\n", url)
				atomic.AddInt64(&cfg.counters.failed, 1)
				cfg.persist(crawlResult{url: url, failed: true})
				return
			}
			article, ref, err := parser.ExtractGospel(html)
			if err != nil {
				log.Printf("Failed to extract gospel from URL %s: %v\n", url, err)
				atomic.AddInt64(&cfg.counters.failed, 1)
				cfg.persist(crawlResult{url: url, failed: true})
				return
			}
			idx2 := parser.FindReadingStartVatican(article)
			if idx2 == -1 {
				log.Printf("Skipping URL (no Vatican markers found in article): %s\n", url)
				atomic.AddInt64(&cfg.counters.failed, 1)
				cfg.persist(crawlResult{url: url, failed: true})
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
			missing := !strings.Contains(content, "{{")
			if err := cfg.persist(crawlResult{url: url, payload: b.String(), missing: missing}); err != nil {
				return
			}
			if missing {
				atomic.AddInt64(&cfg.counters.missingVerse, 1)
			} else {
				atomic.AddInt64(&cfg.counters.matched, 1)
			}
			timer := time.NewTimer(cfg.sleepDuration)
			select {
			case <-ctx.Done():
				timer.Stop()
			case <-timer.C:
			}
		}()
	}
}

func writeLinksToFile(filename string, links []string) error {
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return writeLinksAndClose(out, links)
}

func writeLinksAndClose(out io.WriteCloser, links []string) (err error) {
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close links file: %w", closeErr))
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

func runCrawler(totalUrls int, outFile, processedPath, failedPath, missingPath, sitemapURL, prefix string, sleepDuration time.Duration) error {
	return runCrawlerWithSinkOpener(totalUrls, outFile, processedPath, failedPath, missingPath, sitemapURL, prefix, sleepDuration, openCrawlSink)
}

func runCrawlerWithSinkOpener(totalUrls int, outFile, processedPath, failedPath, missingPath, sitemapURL, prefix string, sleepDuration time.Duration, openSink func(string, string, string, string, context.CancelFunc) (*crawlSink, error)) (err error) {
	counters := &crawlCounters{}

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

	defer client.CloseIdleConnections()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sink, err := openSink(outFile, processedPath, failedPath, missingPath, cancel)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, sink.close()) }()

	jobs := make(chan string, cst.Workers*2)
	var workerWG sync.WaitGroup

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
		go worker(workerConfig{
			client:        client,
			jobs:          jobs,
			ctx:           ctx,
			persist:       sink.persist,
			wg:            &workerWG,
			total:         len(toProcess),
			sleepDuration: sleepDuration,
			counters:      counters,
		})
	}

	// Cancellation interrupts enqueueing; workers also cancel in-flight HTTP requests.
enqueue:
	for _, url := range toProcess {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- url:
		}
	}

	close(jobs)
	workerWG.Wait()
	if sink.err != nil {
		return sink.err
	}

	log.Println()
	log.Println("Crawl finished")
	log.Println("Checked pages:", atomic.LoadInt64(&counters.checked))
	log.Println("Matched pages:", atomic.LoadInt64(&counters.matched))
	log.Println("Missing verse number pages:", atomic.LoadInt64(&counters.missingVerse))
	log.Println("Failed pages:", atomic.LoadInt64(&counters.failed))
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

	if err := runCrawler(totalUrls, outFile, processedPath, failedPath, missingPath, sitemapURL, prefix, defaultWorkerSleep); err != nil {
		log.Fatalf("crawl failed: %v", err)
	}
}
