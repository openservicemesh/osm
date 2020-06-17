package debugger

import (
	"fmt"
	"net/http"
)

func (ds debugServer) getPolicies() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "hello")
	})
}
