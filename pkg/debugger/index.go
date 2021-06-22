package debugger

import (
	"fmt"
	"net/http"
)

func (ds DebugConfig) getDebugIndex(handlers map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, `<ul>`)
		for url := range handlers {
			_, _ = fmt.Fprintf(w, `<li><a href="%s">%s</a><br/>`, url, url)
		}
		_, _ = fmt.Fprint(w, `</ul>`)
	})
}
