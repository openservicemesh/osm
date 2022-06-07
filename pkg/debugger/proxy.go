package debugger

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/openservicemesh/osm/pkg/envoy"
)

const (
	uuidQueryKey        = "uuid"
	proxyConfigQueryKey = "cfg"
)

func (ds DebugConfig) getProxies() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		proxyConfigDump := r.URL.Query()[proxyConfigQueryKey]
		uuid := r.URL.Query()[uuidQueryKey]

		switch {
		case len(uuid) == 0:
			ds.printProxies(w)
		case len(proxyConfigDump) > 0:
			ds.getConfigDump(uuid[0], w)
		default:
			ds.getProxy(uuid[0], w)
		}
	})
}

func (ds DebugConfig) printProxies(w http.ResponseWriter) {
	// This function is needed to convert the list of connected proxies to
	// the type (map) required by the printProxies function.
	proxyMap := ds.proxyRegistry.ListConnectedProxies()
	proxies := make([]*envoy.Proxy, 0, len(proxyMap))
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
		proxyURL := fmt.Sprintf("/debug/proxy?%s=%s", uuidQueryKey, proxy.UUID)
		configDumpURL := fmt.Sprintf("%s&%s=%t", proxyURL, proxyConfigQueryKey, true)
		_, _ = fmt.Fprintf(w, `<tr><td>%d:</td><td>%s</td><td>%s</td><td>%+v</td><td>(%+v ago)</td><td><a href="%s">certs</a></td><td><a href="%s">cfg</a></td></tr>`,
			idx+1, proxy.Identity, proxy.UUID, ts, time.Since(ts), proxyURL, configDumpURL)
	}
	_, _ = fmt.Fprint(w, `</table>`)
}

func (ds DebugConfig) getConfigDump(uuid string, w http.ResponseWriter) {
	proxy := ds.proxyRegistry.GetConnectedProxy(uuid)
	if proxy != nil {
		msg := fmt.Sprintf("Proxy for UUID %s not found, may have been disconnected", uuid)
		log.Error().Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	pod, err := ds.kubeController.GetPodForProxy(proxy)
	if err != nil {
		msg := fmt.Sprintf("Error getting Pod from proxy %s", proxy.GetName())
		log.Error().Err(err).Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	envoyConfig := ds.getEnvoyConfig(pod, "config_dump")
	_, _ = fmt.Fprintf(w, "%s", envoyConfig)
}

func (ds DebugConfig) getProxy(uuid string, w http.ResponseWriter) {
	proxy := ds.proxyRegistry.GetConnectedProxy(uuid)
	if proxy == nil {
		msg := fmt.Sprintf("Proxy for UUID %s not found, may have been disconnected", uuid)
		log.Error().Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	pod, err := ds.kubeController.GetPodForProxy(proxy)
	if err != nil {
		msg := fmt.Sprintf("Error getting Pod from proxy %s", proxy.GetName())
		log.Error().Err(err).Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	envoyConfig := ds.getEnvoyConfig(pod, "certs")
	_, _ = fmt.Fprintf(w, "%s", envoyConfig)
}
