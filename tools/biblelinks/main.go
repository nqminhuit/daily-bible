package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	cst "github.com/minh/daily-bible/internal/constants"
)

var linkRe = regexp.MustCompile(`/bai-viet/[^"' ]+`)

func main() {
	out, err := os.Create(cst.LinkFile)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	writer := bufio.NewWriter(out)
	seen := map[string]bool{}
	page := 1

	for {
		url := fmt.Sprintf("https://tgpsaigon.net/diem-tin/x-10?page=%d", page)
		fmt.Println("Fetching:", url)
		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			panic(err)
		}

		html := string(body)
		matches := linkRe.FindAllString(html, -1)

		if len(matches) == 0 {
			fmt.Println("No more links. Stop.")
			break
		}

		newLinks := 0
		for _, m := range matches {
			full := "https://tgpsaigon.net" + m
			if seen[full] {
				continue
			}
			seen[full] = true
			newLinks++
			writer.WriteString(full + "\n")
		}

		fmt.Println("Page", page, "found", newLinks, "new links")
		page++
		time.Sleep(500 * time.Millisecond) // be nice to the server
	}
	writer.Flush()

	fmt.Println("Total links:", len(seen))
	fmt.Println("Saved to:", cst.LinkFile)
}
