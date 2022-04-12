package verifier

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// Status is a type describing the status of a verification
type Status string

const (
	// Success indicates the verification succeeded
	Success Status = "Success"

	// Failure indicates the verification failed
	Failure Status = "Failure"

	// Unknown indicates the result of the verification could not be determined
	Unknown Status = "Unknown"
)

// Result defines the result returned by a Verifier instance
type Result struct {
	Context       string
	Status        Status
	Reason        string
	Suggestion    string
	NestedResults []*Result
}

// Verifier defines the interface to perform a verification
type Verifier interface {
	Run() Result
}

func marker(count int) string {
	return "[" + strings.Repeat("+", count) + "]"
}

// Print prints the Result
func Print(result Result, w io.Writer, markerDepth int) {
	fmt.Fprintf(w, "%s Context: %s\n", marker(markerDepth), result.Context)
	fmt.Fprintf(w, "Status: %s\n", result.Status.Color())
	if result.Reason != "" {
		fmt.Fprintf(w, "Reason: %s\n", result.Reason)
	}
	if result.Suggestion != "" {
		fmt.Fprintf(w, "Suggestion: %s\n", result.Suggestion)
	}
	fmt.Fprintln(w)

	if result.Status == Success {
		return
	}

	for _, res := range result.NestedResults {
		Print(*res, w, markerDepth+1)
	}
}

// Color returns a color coded string for the verification status
func (s Status) Color() string {
	if s == Success {
		return color.GreenString("%s", s)
	} else if s == Failure {
		return color.RedString("%s", s)
	} else {
		return color.YellowString("%s", s)
	}
}

// Set is a collection of Verifier objects
type Set []Verifier

// Run executes runs the verification for each verifier in the list
func (verifiers Set) Run(ctx string) Result {
	result := Result{
		Context: ctx,
	}
	for _, verification := range verifiers {
		res := verification.Run()
		if res.Status == Failure {
			result.Status = Failure
			result.Reason = "A verification step failed"
			result.Suggestion = "Please follow the suggestions listed in the failed steps below to resolve the issue"
		}
		result.NestedResults = append(result.NestedResults, &res)
	}

	if result.Status != Failure && result.Status != Unknown {
		result.Status = Success
	}

	return result
}
