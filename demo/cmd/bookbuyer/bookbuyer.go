package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/open-service-mesh/osm/demo/cmd/common"
)

const (
	waitForEnvVar                = "WAIT_FOR_OK_SECONDS"
	sleepDurationBetweenRequests = 3 * time.Second
	minSuccessThreshold          = 10
)

func main() {
	bookstoreNamespace := os.Getenv(common.BookstoreNamespaceEnvVar)
	bookstoreService := os.Getenv("BOOKSTORE_SVC")
	if bookstoreService == "" {
		bookstoreService = "bookstore-mesh"
	}
	bookstoreService = fmt.Sprintf("%s.%s", bookstoreService, bookstoreNamespace) // FQDN
	booksBought := fmt.Sprintf("http://%s/books-bought", bookstoreService)
	buyBook := fmt.Sprintf("http://%s/buy-a-book/new", bookstoreService)
	waitForOK := getWaitForOK()
	iteration := 0
	successCount := 0
	hasSucceeded := false
	urlSuccessMap := map[string]bool{booksBought: false, buyBook: false}
	headersMap := map[string]string{
		"client-app": "bookbuyer", // this is a custom header
		"user-agent": "Go-http-client/1.1"}
	urlHeadersMap := map[string]map[string]string{booksBought: headersMap, buyBook: nil}
	for {
		iteration++
		fmt.Printf("---Bookbuyer:[ %d ]-----------------------------------------\n", iteration)
		for url, headersMap := range urlHeadersMap {
			response := fetch(url, headersMap)
			fmt.Println("")
			if waitForOK != 0 {
				if response == http.StatusOK {
					urlSuccessMap[url] = true
					if urlSuccessMap[booksBought] == true && urlSuccessMap[buyBook] == true {
						// All the queries have succeeded, test should succeed from this point
						hasSucceeded = true
					}
					successCount++
					if successCount >= minSuccessThreshold {
						fmt.Println(common.Success)
					}
				} else {
					fmt.Printf("Error, response code = %d\n", response)
					if hasSucceeded {
						fmt.Println(common.Failure)
					}
				}
			}
		}

		fmt.Print("\n\n")
		time.Sleep(sleepDurationBetweenRequests)
	}
}

func fetch(url string, headersMap map[string]string) (responseCode int) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error requesting %s: %s\n", url, err)
	}
	if headersMap != nil {
		for headerKey, headerValue := range headersMap {
			req.Header.Add(headerKey, headerValue)
		}
	}
	fmt.Printf("Fetching %s\n", req.URL)
	fmt.Printf("Request Headers: %v\n", req.Header)
	if resp, err := client.Do(req); err != nil {
		fmt.Printf("Error fetching %s: %s\n", url, err)
	} else {
		responseCode = resp.StatusCode
		for _, hdr := range []string{common.IdentityHeader, common.BooksBoughtHeader, "Server", "Date"} {
			fmt.Printf("%s: %s\n", hdr, getHeader(resp.Header, hdr))
		}
		fmt.Printf("Status: %s\n", resp.Status)
	}
	return responseCode
}

func getHeader(headers map[string][]string, header string) string {
	val, ok := headers[header]
	if !ok {
		val = []string{"n/a"}
	}
	return strings.Join(val, ", ")
}

func getWaitForOK() int64 {
	waitForOKString := os.Getenv(waitForEnvVar)
	waitForOK, err := strconv.ParseInt(waitForOKString, 10, 64)
	if err != nil {
		waitForOK = 0
	}
	return waitForOK
}
