package maestro

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openservicemesh/osm/demo/cmd/common"
)

// We have 45 seconds to get 200 OK from OSM Ingress
// This is upper bound of the time we assume it would take for an ingress controller to program the ingress proxy.
const maxIngressBudget = 45 * time.Second

// TestIngress ensures that traffic flows from the ingressIP + hostname to the bookstore (backend) pods.
func TestIngress(ingressIP string, hostname string, outcome chan<- TestResult) {
	// Set the value of the ingressIP environment variable to a blank string to disable Ingress integration test.
	if ingressIP == "" {
		log.Warn().Msgf("Will NOT test OSM Ingress - no %s env var defined; Define this variable to start testing Ingress functionality", common.IngressIPEnvVar)
		outcome <- TestsPassed
		return
	}

	log.Info().Msgf("Testing OSM Ingress with IP=%q and Host=%q", ingressIP, hostname)

	startTime := time.Now()
	finishBy := startTime.Add(maxIngressBudget)
	go func() {
		for {
			resp, err := getIndex(ingressIP, hostname)
			if err != nil {
				log.Error().Err(err).Msgf("Error testing OSM Ingress with IP=%q and Host=%q", ingressIP, hostname)
				outcome <- TestsFailed
				return
			}

			statusCode := resp.StatusCode
			_ = resp.Body.Close()

			if statusCode == 200 {
				log.Info().Msgf("HTTP GET %s successful", hostname)
				outcome <- TestsPassed
				return
			}

			log.Warn().Msgf("Ingress test got status=%d on HTTP GET %s; Will retry", statusCode, hostname)

			overBudget := time.Now().After(finishBy)
			if overBudget {
				log.Error().Msgf("It has been %+v since we started testing OSM Ingress with HTTP GET %s and still no 200 OK; Giving up", startTime.Sub(time.Now()), hostname)
				outcome <- TestsFailed
				return
			}

			// Wait a bit before we retry HTTP GET on the ingress hostname.
			time.Sleep(5 * time.Second)
		}
	}()
}

func getIndex(ingressIP string, hostname string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s/", ingressIP)
	client := &http.Client{}
	requestBody := strings.NewReader(strconv.Itoa(1))

	req, err := http.NewRequest("GET", url, requestBody)
	if err != nil {
		log.Error().Err(err).Msgf("HTTP GET %s failed", url)
		return nil, err
	}

	req.Header.Add("Host", hostname)

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("HTTP GET %s", url)
		return nil, err
	}

	return resp, nil
}
