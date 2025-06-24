package crawler

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

func Crawl(url string, depth int, visited *sync.Map, wg *sync.WaitGroup, linkChan chan<- string, sem chan struct{}, parentURLs *sync.Map) error {
	defer wg.Done()
	//fmt.Printf("Crawling: %s (depth=%d)\n", url, depth)

	if depth <= 0 {
		return nil
	}

	if _, loaded := visited.LoadOrStore(url, true); loaded {
		return nil
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Request error for %s: %v\n", url, err)
		return nil // Skip
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Fetch error for %s: %v\n", url, err)
		return nil // Skip
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Bad status for %s: %s\n", url, resp.Status)
		return nil // Skip
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Printf("Parse error for %s: %v\n", url, err)
		return nil // Skip
	}

	var links []string
	var node func(n *html.Node, links *[]string)
	node = func(n *html.Node, links *[]string) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := strings.TrimSpace(attr.Val)
					if link != "" && (strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")) {
						*links = append(*links, link)
						//fmt.Printf("Found link: %s\n", link)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			node(c, links)
		}
	}

	node(doc, &links)
	//fmt.Printf("Total links found on %s: %d\n", url, len(links))

	for _, link := range links {
		select {
		case linkChan <- link:
			parentURLs.Store(link, url)
		default:
			//fmt.Printf("linkChan full, skipping link: %s\n", link)
		}
	}

	// Remove link limit
	// if len(links) > 10 {
	// 	links = links[:10]
	// }

	for _, link := range links {
		if _, loaded := visited.Load(link); !loaded {
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(link string) {
					Crawl(link, depth-1, visited, wg, linkChan, sem, parentURLs)
					<-sem
					//fmt.Printf("Finished crawling: %s\n", link)
				}(link)
			default:
				//fmt.Printf("Semaphore full, skipping crawl: %s\n", link)
			}
		}
	}

	return nil
}
