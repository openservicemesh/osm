package framework

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/openservicemesh/osm/pkg/kubernetes"
)

var (
	errEmptyResult = errors.Errorf("Empty result from prometheus")
)

// Prometheus is a simple handler to represent a target Prometheus endpoint to run queries against
type Prometheus struct {
	Client api.Client
	API    v1.API

	pfwd *kubernetes.PortForwarder
}

// Stop gracefully stops the port forwarding to Prometheus
func (p *Prometheus) Stop() {
	p.pfwd.Stop()
}

// VectorQuery runs a query at time <t>, expects single vector type and single result.
// Returns expected first and only <SampleValue> as a float64
// Returns 0 and err<Empty result from prometheus>, if no values are seen on prometheus (but query did succeed)
func (p *Prometheus) VectorQuery(query string, t time.Time) (float64, error) {
	modelValue, warn, err := p.API.Query(context.Background(), query, t)

	if err != nil {
		return 0, err
	}
	if len(warn) > 0 {
		fmt.Printf("Warnings: %v\n", warn)
	}
	switch {
	case modelValue.Type() == model.ValVector:
		vectorVal := modelValue.(model.Vector)
		if len(vectorVal) == 0 {
			return 0, errEmptyResult
		}
		return float64(vectorVal[0].Value), nil
	default:
		return 0, errors.Errorf("Unknown model value type: %v", modelValue.Type().String())
	}
}

// GetNumEnvoysInMesh Gets the Number of in-mesh pods (or envoys) in the mesh as seen
// by prometheus at a certain point in time.
func (p *Prometheus) GetNumEnvoysInMesh(t time.Time) (int, error) {
	queryString := "count(count by(source_pod_name)(envoy_server_live))"
	val, err := p.VectorQuery(queryString, t)
	if err == errEmptyResult {
		return 0, nil
	}
	return int(val), err
}

// GetMemRSSforContainer returns RSS memory footprint for a given NS/podname/containerName
// at a certain point in time
func (p *Prometheus) GetMemRSSforContainer(ns string, podName string, containerName string, t time.Time) (float64, error) {
	queryString := fmt.Sprintf(
		"container_memory_rss{namespace='%s', pod='%s', container='%s'}",
		ns,
		podName,
		containerName)

	return p.VectorQuery(queryString, t)
}

// GetCPULoadAvgforContainer returns CPU load average for a period <duration> just before time <t>
func (p *Prometheus) GetCPULoadAvgforContainer(ns string, podName string, containerName string,
	period time.Duration, t time.Time) (float64, error) {
	queryString := fmt.Sprintf(
		"rate(container_cpu_usage_seconds_total{namespace='%s', pod='%s', container='%s'}[%ds])",
		ns,
		podName,
		containerName,
		int(period.Seconds()))

	return p.VectorQuery(queryString, t)
}

// GetCPULoadsForContainer convenience wrapper to get 1m, 5m and 15m cpu loads for a resource
func (p *Prometheus) GetCPULoadsForContainer(ns string, podName string, containerName string, t time.Time) (float64, float64, float64, error) {
	timeBuckets := []time.Duration{1 * time.Minute, 5 * time.Minute, 15 * time.Minute}
	loads := []float64{}

	for _, bucketTime := range timeBuckets {
		val, err := p.GetCPULoadAvgforContainer(ns, podName, containerName, bucketTime, t)
		if err != nil {
			return 0, 0, 0, err
		}
		loads = append(loads, val)
	}

	return loads[0], loads[1], loads[2], nil
}

/// --- Grafana Rendering API below ---

// Grafana is a simple handler to represent a target Grafana endpoint to run queries against
type Grafana struct {
	Schema   string
	Hostname string
	Port     uint16
	User     string
	Password string

	pfwd *kubernetes.PortForwarder
}

// Stop gracefully stops the port forwarding to Grafana
func (g *Grafana) Stop() {
	g.pfwd.Stop()
}

// PanelPNGSnapshot takes a snapshot from a Grafana dashboard or panel
// and saves it in local in <filename> in png format, using it's remote rendering HTTP API.
func (g *Grafana) PanelPNGSnapshot(dashboard string, panelID int, fromMinutes int, saveFilepath string) error {
	// Grafana render URL

	renderURL, _ := url.Parse(fmt.Sprintf("%s://%s:%d/render/d-solo/%s",
		g.Schema,
		g.Hostname,
		g.Port,
		dashboard))

	renderURL.User = url.UserPassword(g.User, g.Password)

	// Create queries to assign query values
	query := make(url.Values)

	// Org Id is internal to grafana to address admin organizations
	query.Add("orgId", "1")

	// Graphing from <fromMinutes> to now
	query.Add("from", fmt.Sprintf("now-%dm", fromMinutes))
	query.Add("to", "now")

	// size of the drawing, in pixels
	query.Add("width", "1000")
	query.Add("height", "500")

	// panel ID, which panel are we interested in (cpu, mem, etc.)
	query.Add("panelId", fmt.Sprintf("%d", panelID))

	// Add all query parameters to url
	renderURL.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", renderURL.String(), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("Err closing %v", err)
		}
	}()

	saveFilepath = fmt.Sprintf("%s%s", saveFilepath, ".png")
	out, err := os.Create(saveFilepath)
	if err != nil {
		return err
	}
	defer func() {
		err := out.Close()
		if err != nil {
			fmt.Printf("Err closing %v", err)
		}
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Saved panel snapshot as %s\n", saveFilepath)

	return nil
}
