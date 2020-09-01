package common

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/openservicemesh/osm/pkg/logger"
)

const (
	// RestockWarehouseURL is a header string constant.
	RestockWarehouseURL = "restock-books"

	// httpEgressURL is the URL used to test HTTP egress.
	// The HTTP request will result in an HTTPS redirect which will be handled by the HTTP client.
	httpEgressURL = "http://github.com"

	// httpsEgressURL is the URL used to test HTTPS egress
	httpsEgressURL = "https://github.com"

	// bookstorePort is the bookstore service's port
	bookstorePort = 80
)

var (
	sleepDurationBetweenRequestsSecondsStr = GetEnv("CI_SLEEP_BETWEEN_REQUESTS_SECONDS", "1")
	minSuccessThresholdStr                 = GetEnv("CI_MIN_SUCCESS_THRESHOLD", "1")
	maxIterationsStr                       = GetEnv("CI_MAX_ITERATIONS_THRESHOLD", "0") // 0 for unlimited
	bookstoreServiceName                   = GetEnv("BOOKSTORE_SVC", "bookstore")
	bookstoreNamespace                     = os.Getenv(BookstoreNamespaceEnvVar)
	warehouseServiceName                   = "bookwarehouse"
	bookwarehouseNamespace                 = os.Getenv(BookwarehouseNamespaceEnvVar)
	enableEgress                           = os.Getenv(EnableEgressEnvVar) == "true"

	bookstoreService = fmt.Sprintf("%s.%s:%d", bookstoreServiceName, bookstoreNamespace, bookstorePort) // FQDN
	warehouseService = fmt.Sprintf("%s.%s", warehouseServiceName, bookwarehouseNamespace)               // FQDN
	booksBought      = fmt.Sprintf("http://%s/books-bought", bookstoreService)
	buyBook          = fmt.Sprintf("http://%s/buy-a-book/new", bookstoreService)
	chargeAccountURL = fmt.Sprintf("http://%s/%s", warehouseService, RestockWarehouseURL)

	interestingHeaders = []string{
		IdentityHeader,
		BooksBoughtHeader,
		"Server",
		"Date",
	}

	urlHeadersMap = map[string]map[string]string{
		booksBought: {
			"client-app": "bookbuyer", // this is a custom header
			"user-agent": "Go-http-client/1.1",
		},
		buyBook: nil,
	}
)

var log = logger.NewPretty("demo")

// RestockBooks restocks the bookstore with certain amount of books from the warehouse.
func RestockBooks(amount int) {
	log.Info().Msgf("Restocking from book warehouse with %d books", amount)

	client := &http.Client{}
	requestBody := strings.NewReader(strconv.Itoa(1))
	req, err := http.NewRequest("POST", chargeAccountURL, requestBody)
	if err != nil {
		log.Error().Err(err).Msgf("RestockBooks: error posting to %s", chargeAccountURL)
		return
	}

	log.Info().Msgf("RestockBooks: Posted to %s with headers %v", req.URL, req.Header)

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("RestockBooks: Error posting to %s", chargeAccountURL)
		return
	}

	defer resp.Body.Close()
	for _, hdr := range interestingHeaders {
		log.Info().Msgf("RestockBooks (%s) adding header {%s: %s}", chargeAccountURL, hdr, getHeader(resp.Header, hdr))
	}
	log.Info().Msgf("RestockBooks (%s) finished w/ status: %s %d ", chargeAccountURL, resp.Status, resp.StatusCode)
}

// GetEnv is much  like os.Getenv() but with a default value.
func GetEnv(envVar string, defaultValue string) string {
	val := os.Getenv(envVar)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetBooks reaches out to the bookstore and buys/steals books. This is invoked by the bookbuyer and the bookthief.
func GetBooks(participantName string, meshExpectedResponseCode int, egressExpectedResponseCode int, booksCount *int64, booksCountV1 *int64, booksCountV2 *int64) {
	minSuccessThreshold, maxIterations, sleepDurationBetweenRequests := getEnvVars(participantName)

	// The URLs this participant will attempt to query from the bookstore service
	urlSuccessMap := map[string]bool{
		booksBought: false,
		buyBook:     false,
	}

	if enableEgress {
		urlSuccessMap[httpEgressURL] = false
		urlSuccessMap[httpsEgressURL] = false
	}

	urlExpectedRespCode := map[string]int{
		booksBought:    meshExpectedResponseCode,
		buyBook:        meshExpectedResponseCode,
		httpEgressURL:  egressExpectedResponseCode,
		httpsEgressURL: egressExpectedResponseCode,
	}

	// Count how many times we have reached out to the bookstore
	var iteration int64

	// Count how many times BOTH urls have returned the expected status code
	var successCount int64

	// Keep state of the previous success/failure so we know when things regress
	previouslySucceeded := false

	for {
		timedOut := maxIterations > 0 && iteration >= maxIterations
		iteration++

		fmt.Printf("\n\n--- %s:[ %d ] -----------------------------------------\n", participantName, iteration)

		startTime := time.Now()

		for url := range urlSuccessMap {
			// We only care about the response code of the HTTP call for the given URL
			responseCode, identity := fetch(url)

			expectedResponseCode := urlExpectedRespCode[url]
			succeeded := responseCode == expectedResponseCode
			if !succeeded {
				fmt.Printf("ERROR: response code for %q is %d;  expected %d\n", url, responseCode, expectedResponseCode)
			}
			urlSuccessMap[url] = succeeded

			// Regardless of what expect the response to be (depends on the policy) - in case of 200 OK - increase book counts.
			if responseCode == http.StatusOK {
				if url == buyBook {
					if strings.HasPrefix(identity, "bookstore-v1") {
						atomic.AddInt64(booksCountV1, 1)
						atomic.AddInt64(booksCount, 1)
						log.Info().Msgf("BooksCountV1=%d", booksCountV1)
					} else if strings.HasPrefix(identity, "bookstore-v2") {
						atomic.AddInt64(booksCountV2, 1)
						atomic.AddInt64(booksCount, 1)
						log.Info().Msgf("BooksCountV2=%d", booksCountV2)
					}
				}
			}

			// We are looking for a certain number of sequential successful HTTP requests.
			if previouslySucceeded && allUrlsSucceeded(urlSuccessMap) {
				successCount++
				goalReached := successCount >= minSuccessThreshold
				if goalReached && !timedOut {
					// Sending this string to STDOUT will inform the CI Maestro that this is a succeeded;
					// Maestro will stop tailing logs.
					fmt.Println(Success)
				}
			}

			if previouslySucceeded && !succeeded {
				// This is a regression. We had success previously, but now we are seeing a failure.
				// Reset the success counter.
				successCount = 0
			}

			// Keep track of the previous state so we can track a) sequential successes and b) regressions.
			previouslySucceeded = allUrlsSucceeded(urlSuccessMap)
		}

		if timedOut {
			// We are over budget!
			fmt.Printf("Threshold of %d iterations exceeded\n\n", maxIterations)
			fmt.Print(Failure)
		}

		fillerTime := sleepDurationBetweenRequests - time.Since(startTime)
		if fillerTime > 0 {
			time.Sleep(fillerTime)
		}
	}
}

func allUrlsSucceeded(urlSucceeded map[string]bool) bool {
	success := true
	for _, succeeded := range urlSucceeded {
		success = success && succeeded
	}
	return success
}

func fetch(url string) (responseCode int, identity string) {
	headersMap := urlHeadersMap[url]

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error requesting %s: %s\n", url, err)
	}

	for headerKey, headerValue := range headersMap {
		req.Header.Add(headerKey, headerValue)
	}

	fmt.Printf("\nFetching %s\n", req.URL)
	fmt.Printf("Request Headers: %v\n", req.Header)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching %s: %s\n", url, err)
	} else {
		defer resp.Body.Close()
		responseCode = resp.StatusCode
		for _, hdr := range interestingHeaders {
			fmt.Printf("%s: %s\n", hdr, getHeader(resp.Header, hdr))
		}
		fmt.Printf("Status: %s\n", resp.Status)
	}
	identity = "unknown"
	if resp != nil && resp.Header != nil {
		identity = getHeader(resp.Header, IdentityHeader)
	}

	return responseCode, identity
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

// GetExpectedResponseCodeFromEnvVar returns the expected response code based on the given environment variable
func GetExpectedResponseCodeFromEnvVar(envVar, defaultValue string) int {
	expectedRespCodeStr := GetEnv(envVar, defaultValue)
	expectedRespCode, err := strconv.ParseInt(expectedRespCodeStr, 10, 0)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not convert environment variable %s='%s' to int", envVar, expectedRespCodeStr)
	}
	return int(expectedRespCode)
}
