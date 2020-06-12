package debugger

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"

	v1 "k8s.io/api/core/v1"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

func (ds debugServer) getEnvoyConfig(pod *v1.Pod, cn certificate.CommonName) string {
	log.Info().Msgf("Getting Envoy config for CN=%s, podIP=%s", cn, pod.Status.PodIP)

	minPort := 16000
	maxPort := 18000
	portFwdRequest := portForward{
		Pod:       pod,
		LocalPort: rand.Intn(maxPort-minPort) + minPort,
		PodPort:   15000,
		Stop:      make(chan struct{}),
		Ready:     make(chan struct{}),
	}
	go ds.forwardPort(portFwdRequest)

	<-portFwdRequest.Ready

	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/certs", "localhost", portFwdRequest.LocalPort))
	if err != nil {
		log.Error().Err(err).Msgf("Error getting pod with CN=%s and podIP=%s", cn, pod.Status.PodIP)
		return fmt.Sprintf("Error: %s", err)
	}

	defer func() {
		portFwdRequest.Stop <- struct{}{}
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Error getting Envoy config for Pod with CN=%s and IP=%s; HTTP Error %d", cn, pod.Status.PodIP, resp.StatusCode)
		portFwdRequest.Stop <- struct{}{}
		return fmt.Sprintf("Error: %s", err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting pod with CN=%s", cn)
		return fmt.Sprintf("Error: %s", err)
	}

	return string(bodyBytes)
}
