package debugger

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
)

const specificProxyQueryKey = "proxy"

func (ds debugServer) getProxies() http.Handler {
	// This function is needed to convert the list of connected proxies to
	// the type (map) required by the printProxies function.
	listConnected := func() map[certificate.CommonName]time.Time {
		proxies := make(map[certificate.CommonName]time.Time)
		for cn, proxy := range ds.meshCatalogDebugger.ListConnectedProxies() {
			proxies[cn] = (*proxy).GetConnectedAt()
		}
		return proxies
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if specificProxy, ok := r.URL.Query()[specificProxyQueryKey]; !ok {
			printProxies(w, listConnected(), "Connected")
			printProxies(w, ds.meshCatalogDebugger.ListExpectedProxies(), "Expected")
			printProxies(w, ds.meshCatalogDebugger.ListDisconnectedProxies(), "Disconnected")
		} else {
			ds.getProxy(certificate.CommonName(specificProxy[0]), w)
		}
	})
}

func printProxies(w http.ResponseWriter, proxies map[certificate.CommonName]time.Time, category string) {
	var commonNames []string
	for cn := range proxies {
		commonNames = append(commonNames, cn.String())
	}

	sort.Strings(commonNames)

	_, _ = fmt.Fprintf(w, "---| %s Proxies (%d):\n", category, len(proxies))
	for idx, cn := range commonNames {
		ts := proxies[certificate.CommonName(cn)]
		_, _ = fmt.Fprintf(w, "\t%d: \t %s \t %+v \t(%+v ago)\n", idx, cn, ts, time.Since(ts))
	}
	_, _ = fmt.Fprint(w, "\n")
}

func (ds debugServer) getProxy(cn certificate.CommonName, w http.ResponseWriter) {
	pod, err := catalog.GetPodFromCertificate(cn, ds.kubeClient)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod from certificate with CN=%s", cn)
	}
	envoyConfig := ds.getEnvoyConfig(pod, cn)
	_, _ = fmt.Fprintf(w, "%s\n", envoyConfig)
}
