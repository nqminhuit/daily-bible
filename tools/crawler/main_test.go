package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/html"
)

// RoundTripper that always returns an error for testing client GET failures
type badRT struct{}

func (badRT) RoundTrip(_ *http.Request) (*http.Response, error) { return nil, fmt.Errorf("boom") }

func TestLoadLinksAndProcessed(t *testing.T) {
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)

	// create links file
	links := []string{"http://a/1", "http://a/2", ""}
	n := filepath.Join("links.txt")
	_ = os.WriteFile(n, []byte(strings.Join(links, "\n")), 0644)
	got, err := loadLinks(n)
	if err != nil {
		t.Fatalf("loadLinks error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 links, got %d", len(got))
	}

	// processed
	p := filepath.Join("processed.txt")
	_ = os.WriteFile(p, []byte("http://a/2\n"), 0644)
	m := loadProcessed(p)
	if !m["http://a/2"] {
		t.Fatalf("loadProcessed missing entry")
	}
}

func TestWritersAndWorkerIntegration(t *testing.T) {
	// Serve simple Vatican-style HTML pages (main.content) for tests
	b1, err := os.ReadFile("../../test-data/22mar2026.html")
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}

	b2, err := os.ReadFile("../../test-data/12mar2026.html")
	if err != nil {
		t.Fatalf("read html fixture: %v", err)
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.Contains(p, "page1") || strings.HasSuffix(p, "/1") {
			w.Write(b1)
			return
		}
		if strings.Contains(p, "page2") || strings.HasSuffix(p, "/2") {
			w.Write(b2)
			return
		}
		w.Write([]byte("<html></html>"))
	}))
	defer s.Close()

	// chdir to temp to avoid touching repo build files
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)

	// ensure build dir exists (writers create files inside build)
	if err := os.MkdirAll("build", 0755); err != nil {
		t.Fatalf("mkdir build: %v", err)
	}

	// start writers
	resultsCh := make(chan string, 10)
	doneCh := make(chan string, 10)
	missingCh := make(chan string, 10)

	go resultsWriter(resultsCh)
	go processedWriter(doneCh)
	go missingVerseNumWriter(missingCh)

	// prepare http client
	client := &http.Client{Timeout: 2 * time.Second}

	// run worker for two fake URLs
	jobs := make(chan string, 2)
	jobs <- s.URL + "/page1"
	jobs <- s.URL + "/page2"
	close(jobs)

	// set workerSleep to zero to speed up test
	workerSleep = 0

	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 2)

	// wait for worker
	wg.Wait()

	// close writer channels so files flush
	close(resultsCh)
	close(missingCh)
	close(doneCh)

	// give writers a moment
	time.Sleep(10 * time.Millisecond)

	// check files exist
	if _, err := os.Stat("build/gospels.txt"); err != nil {
		t.Fatalf("results file missing: %v", err)
	}
	if _, err := os.Stat("build/missing_verse_number.txt"); err != nil {
		t.Fatalf("missing verse file missing: %v", err)
	}
	if _, err := os.Stat("build/processed.txt"); err != nil {
		t.Fatalf("processed file missing: %v", err)
	}

	// basic contents check
	b, err := os.ReadFile("build/processed.txt")
	if err != nil {
		t.Fatalf("read processed file: %v", err)
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		t.Fatalf("processed file seems empty")
	}
}

func TestWorkerSkipsNon200(t *testing.T) {
	// server that returns 500
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("err"))
	}))
	defer s.Close()

	// chdir to temp
	temp := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(temp)
	_ = os.MkdirAll("build", 0755)

	resultsCh := make(chan string, 10)
	doneCh := make(chan string, 10)
	missingCh := make(chan string, 10)
	go resultsWriter(resultsCh)
	go processedWriter(doneCh)
	go missingVerseNumWriter(missingCh)

	client := &http.Client{Timeout: 2 * time.Second}
	jobs := make(chan string, 1)
	jobs <- s.URL + "/bad"
	close(jobs)

	workerSleep = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 1)
	wg.Wait()

	close(resultsCh)
	close(doneCh)
	close(missingCh)
	// wait
	time.Sleep(10 * time.Millisecond)
	// processed file should exist but be empty
	b, _ := os.ReadFile("build/processed.txt")
	if len(strings.TrimSpace(string(b))) != 0 {
		t.Fatalf("expected processed file to be empty when worker skipped non-200; got %q", string(b))
	}
}

func TestWriteLinksToFile(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmp := t.TempDir()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(tmp)
	_ = os.MkdirAll("build", 0755)

	links := []string{"https://a/1", "https://a/2"}
	if err := writeLinksToFile(links); err != nil {
		t.Fatalf("writeLinksToFile failed: %v", err)
	}
	b, err := os.ReadFile("build/bible-links.txt")
	if err != nil {
		t.Fatalf("read links file: %v", err)
	}
	if !strings.Contains(string(b), "https://a/1") || !strings.Contains(string(b), "https://a/2") {
		t.Fatalf("links not written correctly: %q", string(b))
	}
}

func TestResultsWriterFlushes(t *testing.T) {
	oldWd, _ := os.Getwd()
	tmp := t.TempDir()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(tmp)
	_ = os.MkdirAll("build", 0755)

	resCh := make(chan string, 50)
	go resultsWriter(resCh)
	for i := range 25 {
		resCh <- fmt.Sprintf("entry-%d\n", i)
	}
	close(resCh)
	// wait for writer to create the file and write content (with timeout)
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for gospels file to be written")
		default:
			if fi, err := os.Stat("build/gospels.txt"); err == nil && fi.Size() > 0 {
				b, err := os.ReadFile("build/gospels.txt")
				if err != nil {
					t.Fatalf("read gospels file: %v", err)
				}
				if len(b) == 0 {
					t.Fatalf("gospels file empty after write")
				}
				return
			}
			// short sleep
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// TestMainRun runs main() end-to-end with a local test server to exercise main flow.
func TestMainRun(t *testing.T) {
	// simple pages that contain gospel and verse
	page := `<html><body>
	<section>
	  <div class="section__content">
	    <p>Tin Mừng ngày hôm nay</p>
	    <p><sup>1</sup> Verse one</p>
	  </div>
	</section>
	</body></html>`

	// We'll create a server that serves sitemap.xml and two pages
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sitemap.xml") {
			// use r.Host so we can compose absolute URLs without referencing srv in its own initializer
			host := r.Host
			sx := `<?xml version="1.0"?><urlset>` +
				`<url><loc>http://` + host + `/page1</loc></url>` +
				`<url><loc>http://` + host + `/page2</loc></url>` +
				`</urlset>`
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(sx))
			return
		}
		w.Write([]byte(page))
	}))
	defer srv.Close()

	oldWd, _ := os.Getwd()
	tmp := t.TempDir()
	defer func() { _ = os.Chdir(oldWd) }()
	_ = os.Chdir(tmp)
	_ = os.MkdirAll("build", 0755)

	// set env vars so main uses local server and prefix (include http:// to match generated locs)
	workerSleep = 0

	// call main - should complete
	sitemapURL = srv.URL + "/sitemap.xml"
	main()

	// verify outputs exist
	if _, err := os.Stat("build/gospels.txt"); err != nil {
		t.Fatalf("expected gospels output file created: %v", err)
	}
	if _, err := os.Stat("build/processed.txt"); err != nil {
		t.Fatalf("expected processed file created: %v", err)
	}
}

// Test stripHtmlTags removes tags but preserves text
func TestStripHtmlTags(t *testing.T) {
	in := "<p>Hello <b>World</b> &amp; <span>Go</span></p>"
	out := stripHtmlTags(in)
	if !strings.Contains(out, "Hello World") || !strings.Contains(out, "Go") {
		t.Fatalf("unexpected stripHtmlTags output: %q", out)
	}
}

// Test wrapVerseHTML and cleanText convert <sup>..</sup> to verse markers and normalize whitespace
func TestWrapAndCleanText(t *testing.T) {
	in := "Line1<sup>12</sup>\u00A0\r\nLine2{{3}}<b>Ignore</b>"
	w := wrapVerseHTML(in)
	if !strings.Contains(w, "{{12}}") {
		t.Fatalf("wrapVerseHTML did not convert sup tag: %q", w)
	}

	c := cleanText(in)
	// cleanText should unescape, remove tags and produce verse markers
	if !strings.Contains(c, "{{12}}") || !strings.Contains(c, "{{3}}") {
		t.Fatalf("cleanText did not contain expected verse markers: %q", c)
	}
}

// Test extract returns error when ref exists but no content is found
func TestExtractMissingContentCausesError(t *testing.T) {
	h := `<html><body>
	<section>Tin Mừng ngày hôm nay</section>
	</body></html>`
	_, _, err := ExtractGospel(h)
	if err == nil {
		t.Fatalf("ExtractGospel expected to error when content missing")
	}
}

// Test parseSitemap error conditions
func TestParseSitemapErrors(t *testing.T) {
	// invalid XML
	if _, err := parseSitemap(0, strings.NewReader("<notxml>"), "https://x/"); err == nil {
		t.Fatalf("expected parseSitemap to fail on invalid XML")
	}

	// empty prefix should error when encountering a loc
	xml := `<?xml version="1.0"?><urlset><url><loc>https://a/1</loc></url></urlset>`
	if _, err := parseSitemap(0, strings.NewReader(xml), ""); err == nil {
		t.Fatalf("expected parseSitemap to error on empty prefix")
	}
}

// loadLinks should return an error when file does not exist
func TestLoadLinksFileMissing(t *testing.T) {
	if _, err := loadLinks("non-existent-file.txt"); err == nil {
		t.Fatalf("expected loadLinks to return error for missing file")
	}
}

// Test verse paragraph/header helpers
func TestVerseHelpers(t *testing.T) {
	// create a <p><sup>1</sup>..</p> node to check isVerseParagraph
	h := `<p><sup>1</sup> Text</p><p>No sup here</p>`
	doc, err := html.Parse(strings.NewReader(h))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	p1 := findNode(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "p" })
	if !isVerseParagraph(p1) {
		t.Fatalf("expected first p to be verse paragraph")
	}
	p2 := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "p" && strings.Contains(getText(n), "No sup")
	})
	if isVerseParagraph(p2) {
		t.Fatalf("expected second p to NOT be verse paragraph")
	}
}

// Test verse paragraph/header helpers
func TestGospelHeader(t *testing.T) {
	h := "<p><b>✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mác-cô.       </b>Mc 1,14-20<b></b></p>"
	d2, err := html.Parse(strings.NewReader(h))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}
	ph := findNode(d2, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "p" })
	if !isGospelHeader(ph) {
		t.Fatalf("isGospelHeader failed to detect header")
	}
}

// Test extractGospelSection includes header and verse paragraphs only
func TestExtractGospelSectionBehavior(t *testing.T) {
	h := `<div class="section__content">
	  <p>Tin Mừng header</p>
	  <p><sup>1</sup> Verse one</p>
	  <p><sup>2</sup> Verse two</p>
	  <p>Non-verse paragraph stops here</p>
	</div>`

	doc, _ := html.Parse(strings.NewReader(h))
	div := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "section__content")
	})
	res := extractGospelSection(div)
	if !strings.Contains(res, "Tin Mừng header") {
		t.Fatalf("expected header to be included: %q", res)
	}
	if !strings.Contains(res, "Verse one") || !strings.Contains(res, "Verse two") {
		t.Fatalf("expected verse paragraphs present: %q", res)
	}
	if strings.Contains(res, "Non-verse paragraph") {
		t.Fatalf("non-verse paragraph should stop extraction: %q", res)
	}
}

// Test parseSitemap respects totalUrls limit and skips empty locs
func TestParseSitemapLimitAndEmptyLoc(t *testing.T) {
	xml := `<?xml version="1.0"?><urlset>
	  <url><loc>https://www.vaticannews.va/vi/one</loc></url>
	  <url><loc>https://www.vaticannews.va/vi/two</loc></url>
	  <url><loc></loc></url>
	</urlset>`
	urls, err := parseSitemap(1, strings.NewReader(xml), "https://www.vaticannews.va/vi/")
	if err != nil {
		t.Fatalf("parseSitemap unexpected error: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("expected 1 url due to limit, got %d", len(urls))
	}
	// ensure empty loc skipped
	urls2, _ := parseSitemap(0, strings.NewReader(xml), "https://www.vaticannews.va/vi/")
	for _, u := range urls2 {
		if strings.TrimSpace(u) == "" {
			t.Fatalf("unexpected empty loc in result")
		}
	}
}

// Test worker handles client GET errors gracefully
func TestWorkerClientGetError(t *testing.T) {
	client := &http.Client{Transport: badRT{}}
	jobs := make(chan string, 1)
	resultsCh := make(chan string, 1)
	doneCh := make(chan string, 1)
	missingCh := make(chan string, 1)
	jobs <- "http://example.invalid/"
	close(jobs)

	workerSleep = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 1)
	wg.Wait()
	select {
	case <-resultsCh:
		t.Fatalf("expected no results when client GET errors")
	default:
	}
	select {
	case <-doneCh:
		t.Fatalf("expected no done when client GET errors")
	default:
	}
	select {
	case <-missingCh:
		t.Fatalf("expected no missing when client GET errors")
	default:
	}
}

// Test worker skips pages with no Vatican markers
func TestWorkerSkipsNoVaticanMarkers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>No markers here</body></html>"))
	}))
	defer srv.Close()

	client := srv.Client()
	jobs := make(chan string, 1)
	resultsCh := make(chan string, 1)
	doneCh := make(chan string, 1)
	missingCh := make(chan string, 1)
	jobs <- srv.URL + "/"
	close(jobs)

	workerSleep = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 1)
	wg.Wait()
	select {
	case <-resultsCh:
		t.Fatalf("expected no results for page without markers")
	default:
	}
}

// Test worker sends to missing channel when no verse markers in content
func TestWorkerSendsMissingWhenNoVerse(t *testing.T) {
	h := `<section class="section section--evidence section--isStatic">
			<div class="section__head"><h2>Tin Mừng ngày hôm nay</h2></div>
			<div class="section__wrapper">
				<div class="section__content">
					<p><i>Anh em hãy sám hối và tin vào Tin Mừng.</i></p>
					<p><b>✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mác-cô.&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; </b>Mc 1,14-20<b></b></p>
					<p>&nbsp;Sau khi ông Gio-an bị nộp, Đức Giê-su đến miền Ga-li-lê rao giảng Tin Mừng của Thiên Chúa.&nbsp;&nbsp;Người nói : “Thời kỳ đã mãn, và Triều Đại Thiên Chúa đã đến gần. Anh em hãy sám hối và tin vào Tin Mừng.”</p>
					<p>&nbsp;Người đang đi dọc theo biển hồ Ga-li-lê, thì thấy ông Si-môn với người anh là ông An-rê, đang quăng lưới xuống biển, vì các ông làm nghề đánh cá.&nbsp;&nbsp;Người bảo các ông : “Các anh hãy đi theo tôi, tôi sẽ làm cho các anh trở thành những kẻ lưới người như lưới cá.”&nbsp;&nbsp;Lập tức hai ông bỏ chài lưới mà theo Người.</p>
					<p>&nbsp;Đi xa hơn một chút, Người thấy ông Gia-cô-bê, con ông Dê-bê-đê, và người em là ông Gio-an. Hai ông này đang vá lưới ở trong thuyền.&nbsp;&nbsp;Người liền gọi các ông. Và các ông bỏ cha mình là ông Dê-bê-đê ở lại trên thuyền với những người làm công, mà đi theo Người.</p>
				</div>
			</div>
        </section>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(h))
	}))
	defer srv.Close()

	client := srv.Client()
	jobs := make(chan string, 1)
	resultsCh := make(chan string, 1)
	doneCh := make(chan string, 1)
	missingCh := make(chan string, 1)
	jobs <- srv.URL + "/"
	close(jobs)

	workerSleep = 0
	var wg sync.WaitGroup
	wg.Add(1)
	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 1)
	wg.Wait()

	select {
	case m := <-missingCh:
		if !strings.Contains(m, "__ref__: Mc 1,14-20") {
			t.Fatalf("missing output did not contain expected header: %q", m)
		}
	default:
		t.Fatalf("expected missing output")
	}
}

// Test findNode/findLastNode and helpers hasClass/getText work on parsed HTML
func TestFindNodeHelpers(t *testing.T) {
	h := `<html><body>
	<section>
	  <div class="section__content">
	    <p>Tin Mừng ngày hôm nay</p>
	    <p><sup>1</sup> First verse</p>
	  </div>
	</section>
	<main class="content"><p>Main here</p></main>
	</body></html>`

	doc, err := html.Parse(strings.NewReader(h))
	if err != nil {
		t.Fatalf("parse html: %v", err)
	}

	// getText on the first <p>
	p := findNode(doc, func(n *html.Node) bool { return n.Type == html.ElementNode && n.Data == "p" })
	if p == nil {
		t.Fatal("findNode did not find paragraph")
	}
	text := getText(p)
	if !strings.Contains(text, "Tin Mừng") {
		t.Fatalf("getText returned unexpected: %q", text)
	}

	// hasClass should detect section__content
	div := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "section__content")
	})
	if div == nil {
		t.Fatal("div.section__content not found")
	}

	// findLastNode should find main.content (last occurrence)
	last := findLastNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "main" && hasClass(n, "content")
	})
	if last == nil {
		t.Fatal("findLastNode failed to find main.content")
	}
}

// Test FetchSitemapAndParse handles non-200 responses
func TestFetchSitemapAndParseNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("nope"))
	}))
	defer srv.Close()

	client := srv.Client()
	if _, err := FetchSitemapAndParse(0, srv.URL+"/sitemap.xml", "https://x/", client); err == nil {
		t.Fatalf("expected FetchSitemapAndParse to fail on non-200")
	}
}

// func TestWorkerProgressLogging(t *testing.T) {
// 	// serve pages that contain Tin Mừng and verse markers
// 	h := `<section class="section section--evidence section--isStatic">
// 			<div class="section__head"><h2>Tin Mừng ngày hôm nay</h2></div>
// 			<div class="section__wrapper">
// 				<div class="section__content">
// 					<p><i>Anh em hãy sám hối và tin vào Tin Mừng.</i></p>
// 					<p><b>✠Tin Mừng Chúa Giê-su Ki-tô theo thánh Mác-cô.&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; </b>Mc 1,14-20<b></b></p>
// 					<p><sup>14</sup>&nbsp;Sau khi ông Gio-an bị nộp, Đức Giê-su đến miền Ga-li-lê rao giảng Tin Mừng của Thiên Chúa.&nbsp;<sup>15</sup>&nbsp;Người nói : “Thời kỳ đã mãn, và Triều Đại Thiên Chúa đã đến gần. Anh em hãy sám hối và tin vào Tin Mừng.”</p>
// 					<p><sup>16</sup>&nbsp;Người đang đi dọc theo biển hồ Ga-li-lê, thì thấy ông Si-môn với người anh là ông An-rê, đang quăng lưới xuống biển, vì các ông làm nghề đánh cá.&nbsp;<sup>17</sup>&nbsp;Người bảo các ông : “Các anh hãy đi theo tôi, tôi sẽ làm cho các anh trở thành những kẻ lưới người như lưới cá.”&nbsp;<sup>18</sup>&nbsp;Lập tức hai ông bỏ chài lưới mà theo Người.</p>
// 					<p><sup>19</sup>&nbsp;Đi xa hơn một chút, Người thấy ông Gia-cô-bê, con ông Dê-bê-đê, và người em là ông Gio-an. Hai ông này đang vá lưới ở trong thuyền.&nbsp;<sup>20</sup>&nbsp;Người liền gọi các ông. Và các ông bỏ cha mình là ông Dê-bê-đê ở lại trên thuyền với những người làm công, mà đi theo Người.</p>
// 				</div>
// 			</div>
//         </section>`
// 	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.Write([]byte(h))
// 	}))
// 	defer srv.Close()

// 	client := srv.Client()
// 	jobs := make(chan string, 10)
// 	resultsCh := make(chan string, 10)
// 	doneCh := make(chan string, 10)
// 	missingCh := make(chan string, 10)

// 	// enqueue 4 jobs to trigger progress logging (Progress==2)
// 	for i := range 4 {
// 		jobs <- srv.URL + "/page" + fmt.Sprint(i)
// 	}
// 	close(jobs)

// 	// reset counters
// 	atomic.StoreInt64(&checked, 0)
// 	atomic.StoreInt64(&matched, 0)
// 	atomic.StoreInt64(&missingVerse, 0)

// 	workerSleep = 0
// 	var wg sync.WaitGroup
// 	wg.Add(1)
// 	go worker(client, jobs, resultsCh, doneCh, missingCh, &wg, 4)
// 	wg.Wait()

// 	if atomic.LoadInt64(&checked) != 4 {
// 		t.Fatalf("expected checked==4, got %d", atomic.LoadInt64(&checked))
// 	}
// 	if atomic.LoadInt64(&matched) != 4 {
// 		t.Fatalf("expected matched==4, got %d", atomic.LoadInt64(&matched))
// 	}
// }

func TestProcessedAndMissingWriters(t *testing.T) {
	// ensure build dir exists
	os.MkdirAll("build", 0755)
	// cleanup files
	os.Remove("build/processed.txt")
	os.Remove("build/missing_verse_number.txt")
	os.Remove("build/gospels.txt")

	// processedWriter
	procCh := make(chan string, 2)
	var wg sync.WaitGroup
	wg.Go(func() {
		processedWriter(procCh)
	})

	procCh <- "u1"
	procCh <- "u2"
	close(procCh)
	wg.Wait()

	b, err := os.ReadFile("build/processed.txt")
	if err != nil {
		t.Fatalf("read processed file: %v", err)
	}
	if string(b) == "" {
		t.Fatalf("processed file empty")
	}

	// missingVerseNumWriter
	missCh := make(chan string, 2)
	wg.Go(func() {
		missingVerseNumWriter(missCh)
	})
	missCh <- "m1"
	close(missCh)
	wg.Wait()
	b2, err := os.ReadFile("build/missing_verse_number.txt")
	if err != nil {
		t.Fatalf("read missing file: %v", err)
	}
	if string(b2) == "" {
		t.Fatalf("missing file empty")
	}

	// resultsWriter
	resCh := make(chan string, 2)
	wg.Go(func() {
		resultsWriter(resCh)
	})
	resCh <- "entry1"
	resCh <- "entry2"
	close(resCh)
	wg.Wait()
	b3, err := os.ReadFile("build/gospels.txt")
	if err != nil {
		t.Fatalf("read gospels file: %v", err)
	}
	if len(b3) == 0 {
		t.Fatalf("gospels file empty")
	}

	// cleanup
	os.Remove("build/processed.txt")
	os.Remove("build/missing_verse_number.txt")
	os.Remove("build/gospels.txt")
}
