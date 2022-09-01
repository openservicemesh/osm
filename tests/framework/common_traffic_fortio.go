package framework

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	fortioResDurationHistKey  = "DurationHistogram"
	fortioResPercentilesKey   = "Percentiles"
	fortioResPercentileKey    = "Percentile"
	fortioResValueKey         = "Value"
	fortioResReturnCodesKey   = "RetCodes"
	fortioResRequestsCountKey = "Count"
)

// FortioLoadTestSpec defines the Fortio load test specification. Request definition is not included.
type FortioLoadTestSpec struct {
	// QPS is the number of requests per second. Negative number means no wait maximum rate. The default is 8
	QPS int

	// Connections is the number of connections/goroutines. The default is 4
	Connections int

	// Calls is the number of requests. Default is 0, which uses duration.
	Calls int

	// Duration is the duration of the test. Default is 5 seconds.
	Duration string
}

// FortioHTTPLoadTestDef defines a Fortio HTTP load test intent
type FortioHTTPLoadTestDef struct {
	HTTPRequestDef
	FortioLoadTestSpec
}

// FortioTCPLoadTestDef defines a Fortio TCP load test intent
type FortioTCPLoadTestDef struct {
	TCPRequestDef
	FortioLoadTestSpec
}

// FortioGRPCLoadTestDef defines a Fortio GRPC load test intent
type FortioGRPCLoadTestDef struct {
	GRPCRequestDef
	FortioLoadTestSpec
}

// FortioLoadResult represents Fortio load test result
type FortioLoadResult struct {
	ReturnCodes   map[string]FortioReturnCodeEntry
	DurationHist  map[float64]float64
	TotalRequests int32

	Err error
}

// FortioReturnCodeEntry is a data entry for a single Fortio load test return code with related stats.
type FortioReturnCodeEntry struct {
	ReturnCode string
	Count      int
	Percentage float64
}

// HasFailedHTTPRequests checks if there is return code not smaller than 400. Non-numeric return code will be skipped. This is suitable for checking the result of Fortio HTTP load test.
func (result *FortioLoadResult) HasFailedHTTPRequests() bool {
	for statusCodeStr := range result.ReturnCodes {
		statusCode, err := strconv.Atoi(statusCodeStr)
		if err != nil {
			continue
		}

		if statusCode >= 400 && result.ReturnCodes[statusCodeStr].Count > 0 {
			return true
		}
	}
	return false
}

// AllReturnCodes returns all the load test return codes as an array of strings.
func (result *FortioLoadResult) AllReturnCodes() []string {
	var codes []string
	for retCode := range result.ReturnCodes {
		codes = append(codes, retCode)
	}
	return codes
}

// FortioHTTPLoadTest runs a Fortio load test with HTTP protocol according to given FortioHTTPLoadTestDef and returns a FortioLoadResult
func (td *OsmTestData) FortioHTTPLoadTest(ht FortioHTTPLoadTestDef) FortioLoadResult {
	var command []string
	if td.ClusterOS == constants.OSWindows {
		// -s silent progress, -o output to devnull, '-D -' dump headers to "-" (stdout), -i Status code
		// -I skip body download, '-w StatusCode:%{http_code}' prints Status code label-like for easy parsing
		// -L follow redirects
		command = strings.Fields(fmt.Sprintf("curl.exe -s -o NUL -D - -I -w %s:%%{http_code} -L %s", StatusCodeWord, ht.Destination))
	} else {
		command = buildFortioLoadCommandWithArgs(ht.FortioLoadTestSpec)
		command = append(command, ht.Destination)
	}

	stdout, stderr, err := td.RunRemote(ht.SourceNs, ht.SourcePod, ht.SourceContainer, command)
	if err != nil {
		// Error codes from the execution come through err
		return FortioLoadResult{
			Err: fmt.Errorf("Remote exec err: %v | stderr: %s", err, stderr),
		}
	}
	if len(stderr) > 0 {
		// no error from execution and proper exit code, we got some stderr though
		td.T.Logf("[warn] Stderr: %v", stderr)
	}

	return mapFortioOutputToResult(stdout)
}

// FortioTCPLoadTest runs a Fortio load test with TCP protocol according to given FortioTCPLoadTestDef and returns a FortioLoadResult
func (td *OsmTestData) FortioTCPLoadTest(req FortioTCPLoadTestDef) FortioLoadResult {
	var command []string
	if td.ClusterOS == constants.OSWindows {
		powershellCommand := fmt.Sprintf("$IP = [System.Net.Dns]::GetHostAddresses('%s');", req.DestinationHost) +
			fmt.Sprintf("$Socket = New-Object System.Net.Sockets.TCPClient($IP, %d);", req.DestinationPort) +
			"$Stream = $Socket.GetStream(); $Writer = New-Object System.IO.StreamWriter($Stream);" +
			fmt.Sprintf(" $Writer.WriteLine('%s'); $Writer.Flush();", req.Message) +
			"$reader = New-Object System.IO.StreamReader($Stream); Write-Host -NoNewline $reader.ReadLine()"
		command = []string{"pwsh.exe", "-c", powershellCommand}
	} else {
		command = buildFortioLoadCommandWithArgs(req.FortioLoadTestSpec)
		if req.Message != "" {
			command = append(command, "-payload", fmt.Sprintf("'%s'", req.Message))
		}
		command = append(command, fmt.Sprintf("tcp://%s:%d", req.DestinationHost, req.DestinationPort))
	}

	stdout, stderr, err := td.RunRemote(req.SourceNs, req.SourcePod, req.SourceContainer, command)

	if err != nil {
		return FortioLoadResult{
			Err: fmt.Errorf("Remote exec err: %v | stderr: %s | cmd: %s", err, stderr, command),
		}
	}
	if len(stderr) > 0 {
		// no error from execution and proper exit code, we got some stderr though
		td.T.Logf("[warn] Stderr: %v", stderr)
	}

	return mapFortioOutputToResult(stdout)
}

// FortioGRPCLoadTest runs a Fortio load test with GRPC protocol according to given FortioGRPCLoadTestDef and returns a FortioLoadResult
func (td *OsmTestData) FortioGRPCLoadTest(req FortioGRPCLoadTestDef) FortioLoadResult {
	if req.UseTLS {
		return FortioLoadResult{Err: fmt.Errorf("UseTLS is not supported for Fortio GRPC load test")}
	}

	command := buildFortioLoadCommandWithArgs(req.FortioLoadTestSpec)
	command = append(command, "-grpc", req.Destination)

	stdout, stderr, err := td.RunRemote(req.SourceNs, req.SourcePod, req.SourceContainer, command)

	if err != nil {
		return FortioLoadResult{
			Err: fmt.Errorf("Remote exec err: %w | stderr: %s | cmd: %s", err, stderr, command),
		}
	}
	if len(stderr) > 0 {
		// no error from execution and proper exit code, we got some stderr though
		td.T.Logf("[warn] Stderr: %v", stderr)
	}

	return mapFortioOutputToResult(stdout)
}

// mapFortioOutputToResult maps stdout from single request fortio test
// output. It expects headers on stdout like "<name>: <value...>"
func mapFortioOutputToResult(execOut string) FortioLoadResult {
	var ret = FortioLoadResult{
		ReturnCodes:  make(map[string]FortioReturnCodeEntry),
		DurationHist: make(map[float64]float64),
	}

	var fortioResultJSON map[string]interface{}
	err := json.Unmarshal([]byte(execOut), &fortioResultJSON)
	if err != nil {
		ret.Err = err
		return ret
	}

	// Example Fortio HTTP JSON output:

	durationHist := fortioResultJSON[fortioResDurationHistKey].(map[string]interface{})
	totalRequests := int32(durationHist[fortioResRequestsCountKey].(float64))
	ret.TotalRequests = totalRequests

	// extract duration histogram
	for _, stat := range durationHist[fortioResPercentilesKey].([]interface{}) {
		statObj := stat.(map[string]interface{})
		percentile := statObj[fortioResPercentileKey].(float64)
		duration := statObj[fortioResValueKey].(float64)

		ret.DurationHist[percentile] = duration
	}

	// extract response stats
	for returnCode, count := range fortioResultJSON[fortioResReturnCodesKey].(map[string]interface{}) {
		ret.ReturnCodes[returnCode] = FortioReturnCodeEntry{
			ReturnCode: returnCode,
			Count:      int(count.(float64)),
			Percentage: count.(float64) / float64(totalRequests),
		}
	}

	return ret
}

// buildFortioLoadCommandWithArgs builds a fortio load CLI command given the load test spec and returns an array of strings.
func buildFortioLoadCommandWithArgs(spec FortioLoadTestSpec) []string {
	cmd := []string{"fortio", "load", "-json", "-"}
	if spec.QPS != 0 {
		if spec.QPS < 0 {
			cmd = append(cmd, "-qps", "0")
		} else {
			cmd = append(cmd, "-qps", strconv.Itoa(spec.QPS))
		}
	}
	if spec.Calls != 0 {
		cmd = append(cmd, "-n", strconv.Itoa(spec.Calls))
	}
	if spec.Connections != 0 {
		cmd = append(cmd, "-c", strconv.Itoa(spec.Connections))
	}
	if spec.Duration != "" {
		cmd = append(cmd, "-t", spec.Duration)
	}
	return cmd
}
