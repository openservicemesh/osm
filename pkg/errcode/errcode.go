// Package errcode defines the error codes for error messages and an explanation
// of what the error signifies.
package errcode

import (
	"fmt"
)

type errCode int

const (
	// Kind defines the kind for the error code constants
	Kind = "error_code"
)

// Range 1000-1050 is reserved for errors related to application startup
const (
	// ErrInvalidCLIArgument refers to an invalid CLI argument being specified
	ErrInvalidCLIArgument errCode = iota + 1000
)

// String returns the error code as a string, ex. E1000
func (e errCode) String() string {
	return fmt.Sprintf("E%d", e)
}

//nolint: deadcode,varcheck,unused
var errCodeMap = map[errCode]string{
	ErrInvalidCLIArgument: `
An invalid comment line argument was passed to the application.
`,
}
