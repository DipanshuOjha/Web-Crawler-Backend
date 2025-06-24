package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/DipanshuOjha/Web-crawler/crawler"
	"github.com/DipanshuOjha/Web-crawler/db"
	"github.com/joho/godotenv"
)

func main() {
	urlPtr := flag.String("url", "https://example.com", "Starting URL to crawl")
	depthPtr := flag.Int("depth", 2, "Crawl depth (non-negative integer)")
	concurrencyPtr := flag.Int("concurrency", 10, "Max concurrent goroutines")
	outputPtr := flag.String("output", "console", "Output method (console,sql)")
	flag.Parse()

	if *depthPtr < 0 {
		fmt.Fprintln(os.Stderr, "Error: depth must be positive")
		os.Exit(1)
	}

	if *concurrencyPtr < 1 {
		fmt.Fprintln(os.Stderr, "Error: concurrency must be positive")
		os.Exit(1)
	}

	if *outputPtr != "console" && *outputPtr != "sql" {
		fmt.Fprintln(os.Stderr, "Error: output must be 'console' or 'sql'")
		os.Exit(1)
	}

	fmt.Printf("Starting to crawl %s (depth=%d, concurrency=%d, output=%s)...\n",
		*urlPtr, *depthPtr, *concurrencyPtr, *outputPtr)
	time.Sleep(time.Millisecond * 500)

	var dbConn *db.DB

	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error reading form env :- %v", err)
		os.Exit(1)
	}

	if *outputPtr == "sql" {
		connectionString := os.Getenv("DATABASE_URL")

		if connectionString == "" {
			log.Fatal("didnt find any database url in .env check out once")
		}
		var err error
		fmt.Println("Starting to connect to db......")
		dbConn, err = db.NewDB(connectionString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer dbConn.Close()
		fmt.Println("database connected successfully....!!")

	}

	visited := &sync.Map{}
	uniquelink := &sync.Map{}
	var wg sync.WaitGroup
	linkchan := make(chan string, 100)
	parentURLs := make(map[string]string)
	links := []string{}
	sem := make(chan struct{}, *concurrencyPtr) // Semaphore: goroutines to limit the gorotines
	start := time.Now()
	go func() {
		defer close(linkchan)
		wg.Add(1)
		go crawler.Crawl(*urlPtr, *depthPtr, visited, &wg, linkchan, sem, parentURLs)
		wg.Wait()
	}()

	for link := range linkchan {
		if _, loaded := uniquelink.LoadOrStore(link, true); !loaded {
			links = append(links, link)
		}
	}

	end := time.Now()
	diff := end.Sub(start).Seconds()

	fmt.Println("total time taken in secs before go rotines ", diff)

	if len(links) == 0 {
		fmt.Println("No links found on the page.")
		return
	}

	switch *outputPtr {
	case "console":
		fmt.Printf("Found %d unique links on %s:\n", len(links), *urlPtr)
		for i, link := range links {
			fmt.Printf("%d. %s\n", i+1, link)
			time.Sleep(time.Millisecond * 300)
		}
	case "sql":
		fmt.Println("starting to store links.....")
		if err := dbConn.StoreLinks(links, parentURLs); err != nil {
			fmt.Fprintf(os.Stderr, "Error storing links: %v\n", err)
			os.Exit(1)
		}
		//fmt.Println("Successfully stored all the links.....!!!")
		fmt.Printf("Stored %d unique links in Postgress database \n", len(links))

		// if err := dbConn.ShowStoreLinks(); err != nil {
		// 	fmt.Printf("Warning: Couldn't display results: %v\n", err)
		// }
	}

}
