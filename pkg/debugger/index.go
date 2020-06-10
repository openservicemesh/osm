package debugger

import (
	"fmt"
	"net/http"
)

func (ds debugServer) getDebugIndex(handlers map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for url := range handlers {
			_, _ = fmt.Fprintf(w, "%s\n", url)
		}
	})
}
