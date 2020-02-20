package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	counter = "http://bookstore.mesh/counter"
	incremt = "http://bookstore.mesh/incrementcounter"
)

func main() {
	iteration := 0
	for {
		fmt.Printf("---[ %d ]-----------------------------------------\n", iteration)
		for _, url := range []string{counter, incremt} {
			fetch(url)
		}
		fmt.Println("")
		time.Sleep(3 * time.Second)
	}
}

func fetch(url string) {
	fmt.Printf("Fetching %s\n", url)
	if resp, err := http.Get(url); err != nil {
		fmt.Printf("Error fetching %s: %s\n", url, err)
	} else {
		for _, hdr := range []string{"Identity", "Counter", "Server", "Date"} {
			fmt.Printf("%s: %s\n", hdr, getHeader(resp.Header, hdr))
		}
		fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
	}
}

func getHeader(headers map[string][]string, header string) string {
	val, ok := headers[header]
	if !ok {
		val = []string{"n/a"}
	}
	return strings.Join(val, ", ")
}
