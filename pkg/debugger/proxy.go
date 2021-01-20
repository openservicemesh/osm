package debugger

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	specificProxyQueryKey = "proxy"
	proxyConfigQueryKey   = "cfg"
)

func (ds DebugConfig) getProxies() http.Handler {
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
		w.Header().Set("Content-Type", "text/html")
		if proxyConfigDump, ok := r.URL.Query()[proxyConfigQueryKey]; ok {
			ds.getConfigDump(certificate.CommonName(proxyConfigDump[0]), w)
		} else if specificProxy, ok := r.URL.Query()[specificProxyQueryKey]; ok {
			ds.getProxy(certificate.CommonName(specificProxy[0]), w)
		} else {
			printProxies(w, listConnected(), "Connected")
			printProxies(w, ds.meshCatalogDebugger.ListExpectedProxies(), "Expected")
			printProxies(w, ds.meshCatalogDebugger.ListDisconnectedProxies(), "Disconnected")
		}
	})
}

func printProxies(w http.ResponseWriter, proxies map[certificate.CommonName]time.Time, category string) {
	var commonNames []string
	for cn := range proxies {
		commonNames = append(commonNames, cn.String())
	}

	sort.Strings(commonNames)

	_, _ = fmt.Fprintf(w, "<h1>%s Proxies (%d):</h1>", category, len(proxies))
	_, _ = fmt.Fprint(w, `<table>`)
	_, _ = fmt.Fprint(w, "<tr><td>#</td><td>Envoy's certificate CN</td><td>Connected At</td><td>How long ago</td><td>tools</td></tr>")
	for idx, cn := range commonNames {
		ts := proxies[certificate.CommonName(cn)]
		_, _ = fmt.Fprintf(w, `<tr><td>%d:</td><td>%s</td><td>%+v</td><td>(%+v ago)</td><td><a href="/debug/proxy?%s=%s">certs</a></td><td><a href="/debug/proxy?%s=%s">cfg</a></td></tr>`,
			idx, cn, ts, time.Since(ts), specificProxyQueryKey, cn, proxyConfigQueryKey, cn)
	}
	_, _ = fmt.Fprint(w, `</table>`)
}

func (ds DebugConfig) getConfigDump(cn certificate.CommonName, w http.ResponseWriter) {
	pod, err := catalog.GetPodFromCertificate(cn, ds.kubeController)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod from certificate with CN=%s", cn)
	}
	w.Header().Set("Content-Type", "application/json")
	envoyConfig := ds.getEnvoyConfig(pod, "config_dump")
	_, _ = fmt.Fprintf(w, "%s", envoyConfig)
}

func (ds DebugConfig) getProxy(cn certificate.CommonName, w http.ResponseWriter) {
	pod, err := catalog.GetPodFromCertificate(cn, ds.kubeController)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod from certificate with CN=%s", cn)
	}
	w.Header().Set("Content-Type", "application/json")
	envoyConfig := ds.getEnvoyConfig(pod, "certs")
	_, _ = fmt.Fprintf(w, "%s", envoyConfig)
}
