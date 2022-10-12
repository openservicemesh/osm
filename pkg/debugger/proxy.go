package debugger

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/openservicemesh/osm/pkg/models"
)

const (
	streamIDQueryKey    = "stream-id"
	proxyConfigQueryKey = "cfg"
)

func (ds DebugConfig) getProxies() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		proxyConfigDump := r.URL.Query()[proxyConfigQueryKey]
		streamIDQ := r.URL.Query()[streamIDQueryKey]

		if len(streamIDQ) == 0 {
			ds.printProxies(w)
			return
		}
		streamID, err := strconv.ParseInt(streamIDQ[0], 10, 64)
		if err != nil {
			msg := fmt.Sprintf("couldn't parse streamID %s", streamIDQ[0])
			log.Error().Msg(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		if len(proxyConfigDump) > 0 {
			ds.getConfigDump(streamID, w)
		} else {
			ds.getProxy(streamID, w)
		}
	})
}

func (ds DebugConfig) printProxies(w http.ResponseWriter) {
	// This function is needed to convert the list of connected proxies to
	// the type (map) required by the printProxies function.
	proxyMap := ds.proxyRegistry.ListConnectedProxies()
	proxies := make([]*models.Proxy, 0, len(proxyMap))
	for _, proxy := range proxyMap {
		proxies = append(proxies, proxy)
	}
	sort.Slice(proxies, func(i, j int) bool {
		return proxies[i].Identity.String() < proxies[j].Identity.String()
	})

	_, _ = fmt.Fprintf(w, "<h1>Connected Proxies (%d):</h1>", len(proxies))
	_, _ = fmt.Fprint(w, `<table>`)
	_, _ = fmt.Fprint(w, "<tr><td>#</td><td>Envoy's Service Identity</td><td>Envoy's UUID</td><td>Connected At</td><td>How long ago</td><td>tools</td></tr>")
	for idx, proxy := range proxies {
		ts := proxy.GetConnectedAt()
		proxyURL := fmt.Sprintf("/debug/proxy?%s=%d", streamIDQueryKey, proxy.GetConnectionID())
		configDumpURL := fmt.Sprintf("%s&%s=%t", proxyURL, proxyConfigQueryKey, true)
		_, _ = fmt.Fprintf(w, `<tr><td>%d:</td><td>%s</td><td>%s</td><td>%+v</td><td>(%+v ago)</td><td><a href="%s">certs</a></td><td><a href="%s">cfg</a></td></tr>`,
			idx+1, proxy.Identity, proxy.UUID, ts, time.Since(ts), proxyURL, configDumpURL)
	}
	_, _ = fmt.Fprint(w, `</table>`)
}

func (ds DebugConfig) getConfigDump(streamID int64, w http.ResponseWriter) {
	proxy := ds.proxyRegistry.GetConnectedProxy(streamID)
	if proxy == nil {
		msg := fmt.Sprintf("Proxy for Stream ID %d not found, may have been disconnected", streamID)
		log.Error().Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	envoyConfig, err := ds.computeClient.ConfigFromProxy(proxy, "config_dump", ds.kubeConfig)
	if err != nil {
		msg := fmt.Sprintf("Error getting envoy config from proxy %s", proxy)
		log.Error().Err(err).Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, "%s", envoyConfig)
}

func (ds DebugConfig) getProxy(streamID int64, w http.ResponseWriter) {
	proxy := ds.proxyRegistry.GetConnectedProxy(streamID)
	if proxy == nil {
		msg := fmt.Sprintf("Proxy for Stream ID %d not found, may have been disconnected", streamID)
		log.Error().Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	envoyConfig, err := ds.computeClient.ConfigFromProxy(proxy, "certs", ds.kubeConfig)
	if err != nil {
		msg := fmt.Sprintf("Error getting envoy config from proxy %s", proxy)
		log.Error().Err(err).Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, "%s", envoyConfig)
}
