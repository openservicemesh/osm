package common

import (
	"fmt"

	"github.com/open-service-mesh/osm/pkg/logger"

	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	sleepDurationBetweenRequestsSecondsStr = GetEnv("CI_SLEEP_BETWEEN_REQUESTS_SECONDS", "3")
	minSuccessThresholdStr                 = GetEnv("CI_MIN_SUCCESS_THRESHOLD", "3")
	maxIterationsStr                       = GetEnv("CI_MAX_ITERATIONS_THRESHOLD", "30")
	bookstoreServiceName                   = GetEnv("BOOKSTORE_SVC", "bookstore-mesh")
	bookstoreNamespace                     = os.Getenv(BookstoreNamespaceEnvVar)

	bookstoreService = fmt.Sprintf("%s.%s", bookstoreServiceName, bookstoreNamespace) // FQDN
	booksBought      = fmt.Sprintf("http://%s/books-bought", bookstoreService)
	buyBook          = fmt.Sprintf("http://%s/buy-a-book/new", bookstoreService)

	interestingHeaders = []string{IdentityHeader, BooksBoughtHeader, "Server", "Date"}

	urlHeadersMap = map[string]map[string]string{
		booksBought: {
			"client-app": "bookbuyer", // this is a custom header
			"user-agent": "Go-http-client/1.1",
		},
		buyBook: nil,
	}
)

// GetEnv is much  like os.Getenv() but with a default value.
func GetEnv(envVar string, defaultValue string) string {
	val := os.Getenv(envVar)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetBooks reaches out to the bookstore and buys/steals books. This is invoked by the bookbuyer and the bookthief.
func GetBooks(participantName string, expectedResponseCode int) {
	minSuccessThreshold, maxIterations, sleepDurationBetweenRequests := getEnvVars(participantName)

	// The URLs this participant will attempt to query from the bookstore service
	urlSuccessMap := map[string]bool{
		booksBought: false,
		buyBook:     false,
	}

	// Count how many times we have reached out to the bookstore
	var iteration int64 = 0

	// Count how many times BOTH urls have returned the expected status code
	var successCount int64 = 0

	// Keep state of the previous success/failure so we know when things regress
	previouslySucceeded := false

	for {
		iteration++

		fmt.Printf("\n\n--- %s:[ %d ] -----------------------------------------\n", participantName, iteration)

		for url := range urlSuccessMap {

			// We only care about the response code of the HTTP call for the given URL
			responseCode := fetch(url)

			succeeded := responseCode == expectedResponseCode
			if !succeeded {
				fmt.Printf("ERROR: response code for %q is %d;  expected %d\n", url, responseCode, expectedResponseCode)
			}

			urlSuccessMap[url] = succeeded

			// We are looking for a certain number of sequential successful HTTP requests.
			if previouslySucceeded && allUrlsSucceeded(urlSuccessMap) {
				successCount++
				goalReached := successCount >= minSuccessThreshold
				if goalReached {
					// Sending this string to STDOUT will inform the CI Maestro that this is a succeeded;
					// Maestro will stop tailing logs.
					fmt.Println(Success)
				}

			}

			// Keep track of the previous state so we can track a) sequential successes and b) regressions.
			previouslySucceeded = allUrlsSucceeded(urlSuccessMap)
		}

		if iteration >= maxIterations {
			// We are over budget!
			fmt.Printf("Did not get expected response (%d) in %d iterations (max allowed)\n\n", expectedResponseCode, iteration)
			fmt.Print(Failure)
		}

		time.Sleep(sleepDurationBetweenRequests)
	}
}

func allUrlsSucceeded(urlSucceeded map[string]bool) bool {
	success := true
	for _, succeeded := range urlSucceeded {
		success = success && succeeded
	}
	return success
}

func fetch(url string) (responseCode int) {
	headersMap := urlHeadersMap[url]

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
	fmt.Printf("\nFetching %s\n", req.URL)
	fmt.Printf("Request Headers: %v\n", req.Header)
	if resp, err := client.Do(req); err != nil {
		fmt.Printf("Error fetching %s: %s\n", url, err)
	} else {
		responseCode = resp.StatusCode
		for _, hdr := range interestingHeaders {
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

func getEnvVars(participantName string) (minSuccessThreshold int64, maxIterations int64, sleepDurationBetweenRequests time.Duration) {
	log := logger.New(fmt.Sprintf("demo/%s", participantName))

	var err error

	minSuccessThreshold, err = strconv.ParseInt(minSuccessThresholdStr, 10, 32)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error parsing integer environment variable %q", minSuccessThresholdStr)
	}

	maxIterations, err = strconv.ParseInt(maxIterationsStr, 10, 32)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error parsing integer environment variable %q", maxIterationsStr)
	}

	sleepDurationBetweenRequestsInt, err := strconv.ParseInt(sleepDurationBetweenRequestsSecondsStr, 10, 32)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error parsing integer environment variable %q", sleepDurationBetweenRequestsSecondsStr)
	}

	return minSuccessThreshold, maxIterations, time.Duration(sleepDurationBetweenRequestsInt) * time.Second
}
