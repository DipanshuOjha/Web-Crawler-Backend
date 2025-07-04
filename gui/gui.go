package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
	// Initialize with detailed logging
	fmt.Println("🚀 Initializing server...")

	// Get port with fallback
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port
	fmt.Printf("🔌 Attempting to bind to %s\n", addr)

	// 1. Add pre-bind delay (helps with container initialization)
	time.Sleep(2 * time.Second)

	// 2. Explicit port binding with validation
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("❌ Bind failed: %v", err)
	}
	fmt.Printf("✅ Successfully bound to %s\n", addr)

	// Register handlers
	http.HandleFunc("/api/crawl", enableCORS(crawlHandler))
	http.HandleFunc("/health", enableCORS(healthcheck))

	// 3. Add post-bind delay (ensures Render detects the port)
	fmt.Println("⏳ Waiting for port detection...")
	time.Sleep(3 * time.Second)

	// Start server with logging
	fmt.Println("🌐 Server starting...")
	log.Fatal(http.Serve(ln, nil))
}

func healthcheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	res := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("Error encoding health response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

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
