# Web Crawler in Go

## Overview

This project is a high-performance, concurrent web crawler written in Go. It can crawl web pages starting from a given URL, follow links up to a specified depth, and store discovered URLs and their parent relationships in a PostgreSQL database. The project features both a command-line interface and a simple HTTP API (with CORS enabled) for crawling via JSON requests.

## Features

- **Concurrent Crawling:** Utilizes Go's goroutines and channels for fast, parallel crawling.
- **Configurable Depth & Concurrency:** Control how deep and how many concurrent requests the crawler makes.
- **PostgreSQL Storage:** Stores crawled URLs and their parent links for later analysis.
- **REST API:** Crawl via HTTP POST requests for easy integration with other tools or GUIs.
- **Health Check Endpoint:** Simple `/health` endpoint for monitoring.

## How It Works

The crawler starts from a user-specified URL and recursively follows links found on each page, up to a given depth. It uses Go's goroutines to fetch and parse multiple pages in parallel, dramatically speeding up the crawling process compared to a sequential approach.

### Go Concurrency & Goroutines (In-Depth)

Go is designed for concurrency. Its concurrency model is based on goroutines and channels:

- **Goroutines** are lightweight threads managed by the Go runtime. You can start a goroutine by prefixing a function call with the `go` keyword. For example:

  ```go
  go fetchURL("https://example.com")
  ```

  In this project, each time a new link is found, a new goroutine may be spawned to crawl that link, up to a user-defined concurrency limit.

- **Channels** are used for communication between goroutines. For example, discovered links are sent through a channel to be collected and processed.

- **sync.WaitGroup** is used to wait for all goroutines to finish before the program exits or returns a response.

- **sync.Map** provides a thread-safe map for tracking visited URLs and parent relationships, ensuring no race conditions occur.

- **Semaphores (Buffered Channels):** A buffered channel (e.g., `sem := make(chan struct{}, concurrency)`) is used as a semaphore to limit the number of concurrent HTTP requests. Each goroutine acquires a slot before making a request and releases it when done, preventing resource exhaustion.

#### Example from this project:

```go
wg.Add(1)
go func() {
    crawler.Crawl(url, depth, visited, &wg, linkChan, sem, parentURLs)
}()

// Inside Crawl:
for _, link := range links {
    if _, loaded := visited.Load(link); !loaded {
        select {
        case sem <- struct{}{}:
            wg.Add(1)
            go func(link string) {
                Crawl(link, depth-1, visited, wg, linkChan, sem, parentURLs)
                <-sem // release semaphore
            }(link)
        default:
            // Semaphore full, skip or wait
        }
    }
}
```

**Why is this fast?**  
Instead of waiting for each HTTP request to finish before starting the next, the crawler launches many requests in parallel. This means it can crawl large sites much faster, as it's not bottlenecked by network latency or slow pages. Go's concurrency model makes this both easy to write and efficient to run.

## Project Structure

```
web-crawler/
  ├── crawler/      # Core crawling logic (concurrent, recursive)
  │   └── crawler.go
  ├── db/           # Database connection and storage logic
  │   └── db.go
  ├── gui/          # HTTP API server (can be used for GUI integration)
  │   └── gui.go
  ├── main.go       # CLI entry point
  ├── go.mod        # Go module definition
  └── go.sum        # Go dependencies
```

## Usage

### 1. Install Go

If you don't have Go installed, download it from: https://go.dev/dl/

### 2. Clone the Repository

```sh
git clone https://github.com/DipanshuOjha/Web-crawler.git
cd Web-crawler
```

### 3. Install Dependencies

Go modules will handle dependencies automatically:

```sh
go mod tidy
```

### 4. Build and Run

#### Command-Line Usage

```sh
go run main.go --url="https://example.com" --depth=2 --concurrency=10
```

- `--url`: Starting URL to crawl
- `--depth`: How deep to follow links (non-negative integer)
- `--concurrency`: Max number of concurrent requests

#### HTTP API Usage

Start the API server:

```sh
go run gui/gui.go
```

By default, the server runs on port 8080.

**Crawl via API:**

```sh
curl -X POST http://localhost:8080/api/crawl \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com", "depth":2, "concurrency":10}'
```

**Health Check:**

```sh
curl http://localhost:8080/health
```

### 5. Database

The crawler uses PostgreSQL for storing URLs. Make sure you have a PostgreSQL instance running and set the connection string (e.g., via an `.env` file or environment variable).

## Contributing

Pull requests are welcome! For major changes, please open an issue first to discuss what you would like to change. 

---

By love, Dipanshu Ojha 