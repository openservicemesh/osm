package kube

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport/spdy"

	"github.com/openservicemesh/osm/pkg/k8s"
)

func (c *client) getEnvoyConfig(pod *v1.Pod, envoyURL string, kubeConfig *rest.Config) string {
	log.Debug().Msgf("Getting Envoy config on Pod with UID=%s", pod.ObjectMeta.UID)

	minPort := 16000
	maxPort := 18000
	localPort := rand.Intn(maxPort-minPort) + minPort // #nosec G404
	podPort := 15000

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", pod.Namespace, pod.Name)
	hostIP := strings.TrimLeft(kubeConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(kubeConfig)
	if err != nil {
		log.Error().Err(err).Msg("error getting spdy RoundTripper")
	}

	client := &http.Client{Transport: transport}
	u := &url.URL{Scheme: "https", Path: path, Host: hostIP}

	pf, err := k8s.NewPortForwarder(spdy.NewDialer(upgrader, client, http.MethodPost, u), fmt.Sprintf("%d:%d", localPort, podPort))
	if err != nil {
		log.Error().Err(err).Msgf("Error creating portforwarder")
	}

	var resp *http.Response

	err = pf.Start(func(pf *k8s.PortForwarder) error {
		defer pf.Stop()

		resp, err := client.Get(fmt.Sprintf("http://%s:%d/%s", "localhost", localPort, envoyURL))
		if err != nil {
			log.Error().Err(err).Msgf("Error getting Pod with UID=%s", pod.ObjectMeta.UID)
			return err
		}

		//nolint: errcheck
		//#nosec G307
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Error().Msgf("Error getting Envoy config on Pod with UID=%s; HTTP Error %d", pod.ObjectMeta.UID, resp.StatusCode)
			return err
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod with UID=%s", pod.ObjectMeta.UID)
		return fmt.Sprintf("Error: %s", err)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Pod with UID=%s", pod.ObjectMeta.UID)
		return fmt.Sprintf("Error: %s", err)
	}

	return string(bodyBytes)
}
