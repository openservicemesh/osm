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
	for {
		iteration++
		fmt.Printf("---Bookthief:[ %d ]-----------------------------------------\n", iteration)
		var responses []int
		for _, url := range []string{booksBought, buyBook} {
			response := fetch(url)
			fmt.Println("")
			responses = append(responses, response)
		}
		if waitForOK != 0 {
			//since bookthief doesn't have any traffic policies setup to talk to bookstore it will get a 404
			if responses[0] == http.StatusNotFound {
				fmt.Printf(common.Success)
			} else {
				fmt.Printf("Error, response code = %d", responses[0])
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
