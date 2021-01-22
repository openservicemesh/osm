package framework

import (
	"bufio"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
)

const (
	// StatusCodeWord is an identifier used on curl commands to print and parse REST Status codes
	StatusCodeWord = "StatusCode"
)

// HTTPRequestDef defines a remote HTTP request intent
type HTTPRequestDef struct {
	// Source pod where to run the HTTP request from
	SourceNs        string
	SourcePod       string
	SourceContainer string

	// The entire destination URL processed by curl, including host name and
	// optionally protocol, port, and path
	Destination string
}

// TCPRequestDef defines a remote TCP request intent
type TCPRequestDef struct {
	// Source pod where to run the HTTP request from
	SourceNs        string
	SourcePod       string
	SourceContainer string

	// The destination server host (FQDN or IP address) and port the request is directed to
	DestinationHost string
	DestinationPort int

	// Message to send as a part of the request
	Message string
}

// GRPCRequestDef defines a remote GRPC request intent
type GRPCRequestDef struct {
	// Source pod where to run the HTTP request from
	SourceNs        string
	SourcePod       string
	SourceContainer string

	// The entire destination URL processed by curl, including host name and
	// optionally protocol, port, and path
	Destination string

	// JSONRequest is the JSON request body
	JSONRequest string

	// Symbol is the fully qualified grpc service name, ex. hello.HelloService/SayHello
	Symbol string
}

// HTTPRequestResult represents results of an HTTPRequest call
type HTTPRequestResult struct {
	StatusCode int
	Headers    map[string]string
	Err        error
}

// TCPRequestResult represents the result of a TCPRequest call
type TCPRequestResult struct {
	Response string
	Err      error
}

// GRPCRequestResult represents the result of a GRPCRequest call
type GRPCRequestResult struct {
	Response string
	Err      error
}

// HTTPRequest runs a synchronous call to run the HTTPRequestDef and return a HTTPRequestResult
func (td *OsmTestData) HTTPRequest(ht HTTPRequestDef) HTTPRequestResult {
	// -s silent progress, -o output to devnull, '-D -' dump headers to "-" (stdout), -i Status code
	// -I skip body download, '-w StatusCode:%{http_code}' prints Status code label-like for easy parsing
	// -L follow redirects
	commandStr := fmt.Sprintf("/usr/bin/curl -s -o /dev/null -D - -I -w %s:%%{http_code} -L %s", StatusCodeWord, ht.Destination)
	command := strings.Fields(commandStr)

	stdout, stderr, err := td.RunRemote(ht.SourceNs, ht.SourcePod, ht.SourceContainer, command)
	if err != nil {
		// Error codes from the execution come through err
		// Curl 'Connection refused' err code = 7
		return HTTPRequestResult{
			0,
			nil,
			fmt.Errorf("Remote exec err: %v | stderr: %s", err, stderr),
		}
	}
	if len(stderr) > 0 {
		// no error from execution and proper exit code, we got some stderr though
		td.T.Logf("[warn] Stderr: %v", stderr)
	}

	// Expect predictable output at this point from the curl we executed
	curlMappedReturn := mapCurlOuput(stdout)
	statusCode, err := strconv.Atoi(curlMappedReturn[StatusCodeWord])
	if err != nil {
		return HTTPRequestResult{
			0,
			nil,
			fmt.Errorf("Could not read status code as integer: %v", err),
		}
	}
	delete(curlMappedReturn, StatusCodeWord)

	return HTTPRequestResult{
		statusCode,
		curlMappedReturn,
		nil,
	}
}

// TCPRequest runs a synchronous TCP request to run the TCPRequestDef and return a TCPRequestResult
func (td *OsmTestData) TCPRequest(req TCPRequestDef) TCPRequestResult {
	commandArgs := fmt.Sprintf("echo \"%s\" | nc %s %d", req.Message, req.DestinationHost, req.DestinationPort)
	command := []string{"sh", "-c", commandArgs}

	stdout, stderr, err := td.RunRemote(req.SourceNs, req.SourcePod, req.SourceContainer, command)
	if err != nil {
		// Error codes from the execution come through err
		return TCPRequestResult{
			stdout,
			fmt.Errorf("Remote exec err: %v | stderr: %s | cmd: %s", err, stderr, command),
		}
	}
	if len(stderr) > 0 {
		// no error from execution and proper exit code, we got some stderr though
		td.T.Logf("[warn] Stderr: %v", stderr)
	}

	return TCPRequestResult{
		stdout,
		nil,
	}
}

// GRPCRequest runs a GRPC request to run the GRPCRequestDef and return a GRPCRequestResult
func (td *OsmTestData) GRPCRequest(req GRPCRequestDef) GRPCRequestResult {
	command := []string{"/grpcurl", "-d", req.JSONRequest, "-plaintext", req.Destination, req.Symbol}

	stdout, stderr, err := td.RunRemote(req.SourceNs, req.SourcePod, req.SourceContainer, command)
	if err != nil {
		// Error codes from the execution come through err
		return GRPCRequestResult{
			stdout,
			fmt.Errorf("Remote exec err: %v | stderr: %s | cmd: %s", err, stderr, command),
		}
	}
	if len(stderr) > 0 {
		// no error from execution and proper exit code, we got some stderr though
		td.T.Logf("[warn] Stderr: %v", stderr)
	}

	return GRPCRequestResult{
		stdout,
		nil,
	}
}

// MapCurlOuput maps stdout from our specific curl,
// it expects headers on stdout like "<name>: <value...>"
func mapCurlOuput(curlOut string) map[string]string {
	var ret map[string]string = make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(curlOut))

	for scanner.Scan() {
		line := scanner.Text()

		// Expect at most 2 substrings, separating by only the first colon
		splitResult := strings.SplitN(line, ":", 2)

		if len(splitResult) != 2 {
			// other non-header data
			continue
		}
		ret[strings.TrimSpace(splitResult[0])] = strings.TrimSpace(splitResult[1])
	}
	return ret
}

// HTTPMultipleRequest takes multiple HTTP request defs to issue them concurrently
type HTTPMultipleRequest struct {
	// Request
	Sources []HTTPRequestDef
}

// HTTPMultipleResults represents results from a multiple HTTP request call
// results come back as a map["srcNs/srcPod"]["dstNs/dstPod"] -> HTTPResults
type HTTPMultipleResults map[string]map[string]HTTPRequestResult

// MultipleHTTPRequest will issue a list of requests concurrently and return results when all requests have returned
func (td *OsmTestData) MultipleHTTPRequest(requests *HTTPMultipleRequest) HTTPMultipleResults {
	results := HTTPMultipleResults{}
	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}

	// Prepare results
	for idx, r := range requests.Sources {
		srcKey := fmt.Sprintf("%s/%s", r.SourceNs, r.SourcePod)
		dstKey := r.Destination

		if _, ok := results[srcKey]; !ok {
			results[srcKey] = map[string]HTTPRequestResult{}
		}
		if _, ok := results[srcKey][dstKey]; !ok {
			results[srcKey][dstKey] = HTTPRequestResult{}
		} else {
			td.T.Logf("No support for more than one request from src to dst. (%s to %s).Ignoring.",
				srcKey, dstKey)
			continue
		}

		wg.Add(1)
		go func(ns string, podname string, htReq HTTPRequestDef) {
			defer wg.Done()
			r := td.HTTPRequest(htReq)

			// Need lock to avoid concurrent map writes
			mtx.Lock()
			results[ns][podname] = r
			mtx.Unlock()
		}(srcKey, dstKey, (*requests).Sources[idx])
	}
	wg.Wait()

	return results
}

// PrettyPrintHTTPResults prints pod results per namespace
func (td *OsmTestData) PrettyPrintHTTPResults(results *HTTPMultipleResults) {
	// We sort the keys to always walk the maps deterministically.
	namespaceKeys := []string{}
	for nsKey := range *results {
		namespaceKeys = append(namespaceKeys, nsKey)
	}
	sort.Strings(namespaceKeys)

	for _, ns := range namespaceKeys {
		podKeys := []string{}
		for podKey := range (*results)[ns] {
			podKeys = append(podKeys, podKey)
		}
		sort.Strings(podKeys)

		strLine := fmt.Sprintf("%s - ", color.CyanString(ns))
		for _, pod := range podKeys {
			strLine += fmt.Sprintf("%s: %s -", pod, getColoredStatusCode((*results)[ns][pod]))
		}
		td.T.Log(strLine)
	}
}

func getColoredStatusCode(res HTTPRequestResult) string {
	var coloredStatus string
	if res.Err != nil {
		coloredStatus = color.RedString("ERR")
	} else if res.StatusCode != 200 {
		coloredStatus = color.YellowString("%d ", res.StatusCode)
	} else {
		coloredStatus = color.HiGreenString("%d ", res.StatusCode)
	}

	return coloredStatus
}
