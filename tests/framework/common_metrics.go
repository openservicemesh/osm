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
)

// Prometheus is a simple handler to represent a target Prometheus endpoint to run queries against
type Prometheus struct {
	Client api.Client
	API    v1.API
}

// VectorQuery runs a query at time <t>, expects single vector type and single result.
// Returns expected first and only <SampleValue> as a float64
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
			return 0, errors.Errorf("Empty result from prometheus")
		}
		return float64(vectorVal[0].Value), nil
	default:
		return 0, errors.Errorf("Unknown model value type: %v", modelValue.Type().String())
	}
}

// GetNumEnvoysInMesh Gets the Number of in-mesh pods (or envoys) in the mesh as seen
// by prometheus.
func (p *Prometheus) GetNumEnvoysInMesh() (float64, error) {
	queryString := "count(count by(source_pod_name)(envoy_server_live))"
	return p.VectorQuery(queryString, time.Now())
}

// GetMemRSSforContainer returns RSS memory footprint for a given NS/podname/containerName
func (p *Prometheus) GetMemRSSforContainer(ns string, podName string, containerName string) (float64, error) {
	queryString := fmt.Sprintf(
		"container_memory_rss{namespace='%s', pod='%s', container='%s'}",
		ns,
		podName,
		containerName)

	return p.VectorQuery(queryString, time.Now())
}

// GetCPULoadAvgforContainer returns CPU load average value for the time bucket passed as parametres, in minutes
func (p *Prometheus) GetCPULoadAvgforContainer(ns string, podName string, containerName string, minutesBucket int) (float64, error) {
	queryString := fmt.Sprintf(
		"rate(container_cpu_usage_seconds_total{namespace='%s', pod='%s', container='%s'}[%dm])",
		ns,
		podName,
		containerName,
		minutesBucket)

	return p.VectorQuery(queryString, time.Now())
}

// GetCPULoadsForContainer convenience wrapper to get 1m, 5m and 15m cpu loads for a resource
func (p *Prometheus) GetCPULoadsForContainer(ns string, podName string, containerName string) (float64, float64, float64, error) {
	timesMinutes := []int{1, 5, 15}
	loads := []float64{}

	for _, t := range timesMinutes {
		val, err := p.GetCPULoadAvgforContainer(ns, podName, containerName, t)
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

	// Add all query parametres to url
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
