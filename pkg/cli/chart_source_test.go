package cli

import (
	"io/ioutil"
	"os"
	"testing"

	"helm.sh/helm/v3/pkg/chartutil"
)

func TestChartSource(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "osm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := os.RemoveAll(tmp)
		if err != nil {
			t.Log(err)
		}
	}()

	chartName := "test-chart"
	chartPath, err := chartutil.Create(chartName, tmp)
	if err != nil {
		t.Fatal(err)
	}

	source, err := GetChartSource(chartPath)
	if err != nil {
		t.Fatal(err)
	}

	ch, err := LoadChart(source)
	if err != nil {
		t.Fatal(err)
	}

	if ch.Name() != chartName {
		t.Errorf("expected chart name %q, got %q", chartName, ch.Name())
	}
}
