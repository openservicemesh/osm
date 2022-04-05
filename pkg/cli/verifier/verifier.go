package verifier

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

// VerificationStatus is a type describing the status of a verification
type VerificationStatus string

const (
	// Success indicates the verification succeeded
	Success VerificationStatus = "Success"

	// Failure indicates the verification failed
	Failure VerificationStatus = "Failure"

	// Unknown indicates the result of the verification could not be determined
	Unknown VerificationStatus = "Unknown"
)

// VerificationResult defines the result returned by a Verifier instance
type VerificationResult struct {
	Context       string
	Status        VerificationStatus
	Reason        string
	Suggestion    string
	NestedResults []*VerificationResult
}

// Verifier defines the interface to perform a verification
type Verifier interface {
	Run() VerificationResult
}

// Verify runs the given list of verifiers and returns a corresponding
// list of verification results.
func Verify(toVerify []Verifier) []*VerificationResult {
	var result []*VerificationResult
	for _, v := range toVerify {
		res := v.Run()
		result = append(result, &res)
	}
	return result
}

// Print prints the
func Print(result VerificationResult, w io.Writer) {
	fmt.Fprintf(w, "[+] Context: %s\n", result.Context)
	fmt.Fprintf(w, "Status: %s\n", result.Status.Color())
	fmt.Fprintf(w, "Reason: %s\n", result.Reason)
	fmt.Fprintf(w, "Suggestion: %s\n\n", result.Suggestion)

	if result.Status == Success {
		return
	}

	for _, res := range result.NestedResults {
		Print(*res, w)
	}
}

// Color returns a color coded string for the verification status
func (s VerificationStatus) Color() string {
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
func (verifiers Set) Run(ctx string) VerificationResult {
	result := VerificationResult{
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
