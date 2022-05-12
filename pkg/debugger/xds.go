package debugger

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

func (ds DebugConfig) getXDSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xdsLog := ds.xdsDebugger.GetXDSLog()

		var proxies []string
		for proxyCN := range *xdsLog {
			proxies = append(proxies, proxyCN.String())
		}

		sort.Strings(proxies)

		for _, proxyCN := range proxies {
			xdsTypeWithTimestamps := (*xdsLog)[certificate.CommonName(proxyCN)]
			_, _ = fmt.Fprintf(w, "---[ %s\n", proxyCN)

			var xdsTypes []string
			for xdsType := range xdsTypeWithTimestamps {
				xdsTypes = append(xdsTypes, xdsType.String())
			}

			sort.Strings(xdsTypes)

			for _, xdsType := range xdsTypes {
				timeStamps := xdsTypeWithTimestamps[envoy.TypeURI(xdsType)]

				_, _ = fmt.Fprintf(w, "\t %s (%d):\n", xdsType, len(timeStamps))

				sort.Slice(timeStamps, func(i, j int) bool {
					return timeStamps[i].After(timeStamps[j])
				})
				for _, timeStamp := range timeStamps {
					_, _ = fmt.Fprintf(w, "\t\t%+v (%+v ago)\n", timeStamp, time.Since(timeStamp))
				}
				_, _ = fmt.Fprint(w, "\n")
			}
			_, _ = fmt.Fprint(w, "\n")
		}
	})
}
