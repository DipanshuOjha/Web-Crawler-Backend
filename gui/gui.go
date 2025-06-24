package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/DipanshuOjha/Web-crawler/crawler"
)

type CrawlRqst struct {
	URL         string `json:"url"`
	Depth       int    `json:"depth"`
	Concurrency int    `json:"concurrency"`
	Output      string `json:"output"`
}

type CrawlResponse struct {
	Links      []string          `json:"links"`
	ParentURLs map[string]string `json:"parent_urls"`
	Duration   float64           `json:"duration_seconds"`
	Error      string            `json:"error,omitempty"`
}

func main() {
	http.HandleFunc("/api/crawl", enableCORS(crawlHandler))

	fmt.Println("Starting API server at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func crawlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CrawlRqst
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" || req.Depth < 0 || req.Concurrency < 1 {
		sendError(w, "Invalid input", http.StatusBadRequest)
		return
	}

	visited := &sync.Map{}
	uniqueLinks := &sync.Map{}
	var wg sync.WaitGroup
	linkChan := make(chan string, 1000)
	parentURLs := &sync.Map{}
	links := []string{}
	sem := make(chan struct{}, req.Concurrency)

	start := time.Now()

	var collectWg sync.WaitGroup
	collectWg.Add(1)
	go func() {
		defer collectWg.Done()
		for link := range linkChan {
			if _, loaded := uniqueLinks.LoadOrStore(link, true); !loaded {
				links = append(links, link)
			}
		}
	}()

	wg.Add(1)
	go func() {
		crawler.Crawl(req.URL, req.Depth, visited, &wg, linkChan, sem, parentURLs)
	}()

	wg.Wait()
	close(linkChan)
	collectWg.Wait()

	parentMap := make(map[string]string)
	parentURLs.Range(func(key, value interface{}) bool {
		parentMap[key.(string)] = value.(string)
		return true
	})

	w.Header().Set("Content-Type", "application/json")
	resp := CrawlResponse{
		Links:      links,
		ParentURLs: parentMap,
		Duration:   time.Since(start).Seconds(),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		sendError(w, "Internal server error", http.StatusInternalServerError)
		log.Printf("Error encoding JSON: %v", err)
	}
}

func sendError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(CrawlResponse{Error: msg})
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins (adjust for production)
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
