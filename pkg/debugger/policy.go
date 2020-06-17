package debugger

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/certificate"
)



func (ds debugServer) getPolicies() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "hello" )
		
	})
}