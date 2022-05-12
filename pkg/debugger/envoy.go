package debugger

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"

	v1 "k8s.io/api/core/v1"
)

func (ds DebugConfig) getEnvoyConfig(pod *v1.Pod, url string) string {
	log.Debug().Msgf("Getting Envoy config on Pod with UID=%s", pod.ObjectMeta.UID)

	minPort := 16000
	maxPort := 18000

	// #nosec G404
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
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/%s", "localhost", portFwdRequest.LocalPort, url))
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod with UID=%s", pod.ObjectMeta.UID)
		return fmt.Sprintf("Error: %s", err)
	}

	defer func() {
		portFwdRequest.Stop <- struct{}{}
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Error getting Envoy config on Pod with UID=%s; HTTP Error %d", pod.ObjectMeta.UID, resp.StatusCode)
		portFwdRequest.Stop <- struct{}{}
		return fmt.Sprintf("Error: %s", err)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod with UID=%s", pod.ObjectMeta.UID)
		return fmt.Sprintf("Error: %s", err)
	}

	return string(bodyBytes)
}
