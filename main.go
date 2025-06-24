package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/DipanshuOjha/Web-crawler/crawler"
)

func main() {
	urlPtr := flag.String("url", "https://example.com", "Starting URL to crawl")
	depthPtr := flag.Int("depth", 2, "Crawl depth (non-negative integer)")
	concurrencyPtr := flag.Int("concurrency", 10, "Max concurrent goroutines")
	flag.Parse()

	if *depthPtr < 0 {
		fmt.Fprintln(os.Stderr, "Error: depth must be positive")
		os.Exit(1)
	}

	if *concurrencyPtr < 1 {
		fmt.Fprintln(os.Stderr, "Error: concurrency must be positive")
		os.Exit(1)
	}

	fmt.Printf("Starting to crawl %s (depth=%d, concurrency=%d)...\n", *urlPtr, *depthPtr, *concurrencyPtr)
	time.Sleep(time.Millisecond * 500)

	visited := &sync.Map{}
	uniqueLinks := &sync.Map{}
	var wg sync.WaitGroup
	linkChan := make(chan string, 1000)
	parentURLs := &sync.Map{}
	links := []string{}
	sem := make(chan struct{}, *concurrencyPtr)

	start := time.Now()

	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for link := range linkChan {
			if _, loaded := uniqueLinks.LoadOrStore(link, true); !loaded {
				links = append(links, link)
				//fmt.Printf("Collected link: %s\n", link)
			}
		}
		//fmt.Println("Finished collecting links")
	}()

	wg.Add(1)
	go func() {
		crawler.Crawl(*urlPtr, *depthPtr, visited, &wg, linkChan, sem, parentURLs)
		fmt.Println("Crawler goroutine done")
	}()

	wg.Wait()
	close(linkChan)

	collectWg.Wait()

	end := time.Now()
	diff := end.Sub(start).Seconds()

	fmt.Printf("Total time taken: %.2f seconds\n", diff)

	if len(links) == 0 {
		fmt.Println("No links found on the page.")
		return
	}

	fmt.Printf("Found %d unique links on %s:\n", len(links), *urlPtr)
	for i, link := range links {
		parent, _ := parentURLs.Load(link)
		fmt.Printf("%d. %s (parent: %v)\n", i+1, link, parent)
		time.Sleep(time.Millisecond * 300)
	}
}
