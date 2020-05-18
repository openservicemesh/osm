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
	bookstoreService := os.Getenv("BOOKSTORE_SVC")
	if bookstoreService == "" {
		bookstoreService = "bookstore.mesh"
	}
	booksBought := fmt.Sprintf("http://%s/books-bought", bookstoreService)
	buyBook := fmt.Sprintf("http://%s/buy-a-book/new", bookstoreService)
	waitForOK := getWaitForOK()
	iteration := 0
	successCount := 0
	hasSucceeded := false
	urlSuccessMap := map[string]bool{booksBought: false, buyBook: false}
	for {
		iteration++
		fmt.Printf("---Bookthief:[ %d ]-----------------------------------------\n", iteration)
		for _, url := range []string{booksBought, buyBook} {
			response := fetch(url)
			fmt.Println("")
			if waitForOK != 0 {
				//since bookthief doesn't have any traffic policies setup to talk to bookstore it will get a 404
				if response == http.StatusNotFound {
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

func fetch(url string) (responseCode int) {
	fmt.Printf("Fetching %s\n", url)
	if resp, err := http.Get(url); err != nil {
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
