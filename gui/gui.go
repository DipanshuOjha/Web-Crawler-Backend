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
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Uptime    string `json:"uptime,omitempty"`
}

var startTime = time.Now()

func main() {
	http.HandleFunc("/api/crawl", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CrawlRqst

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.URL == "" || req.Depth < 0 || req.Concurrency < 1 || (req.Output != "console" && req.Output != "sql") {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}

		visited := &sync.Map{}
		uniqueLinks := &sync.Map{}
		var wg sync.WaitGroup
		linkChan := make(chan string, 100)
		parentURLs := make(map[string]string)
		sem := make(chan struct{}, req.Concurrency)

		start := time.Now()
		go func() {
			defer close(linkChan)
			wg.Add(1)
			go crawler.Crawl(req.URL, req.Depth, visited, &wg, linkChan, sem, parentURLs)
			wg.Wait()
		}()

		links := []string{}
		for link := range linkChan {
			if _, loaded := uniqueLinks.LoadOrStore(link, true); !loaded {
				links = append(links, link)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		resp := CrawlResponse{
			Links:      links,
			ParentURLs: parentURLs,
			Duration:   time.Since(start).Seconds(),
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			sendError(w, "Internal server error", http.StatusInternalServerError)
			log.Printf("Error encoding JSON: %v", err)
		}

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().Format(time.RFC3339),
			Uptime:    time.Since(startTime).String(),
		}); err != nil {
			sendError(w, "Internal server error", http.StatusInternalServerError)
			log.Printf("Error encoding JSON: %v", err)
		}

	})

	fmt.Println("Starting API server at http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func sendError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(CrawlResponse{Error: msg})
}
