package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestChartSource(t *testing.T) {
	assert := tassert.New(t)

	tmp, err := ioutil.TempDir(os.TempDir(), "osm-test")
	assert.Nil(err, fmt.Errorf("failed to create temp dir: %w", err))

	defer func() {
		err := os.RemoveAll(tmp)
		if err != nil {
			t.Log("error cleaning up temp dir:", err)
		}
	}()

	chartName := "test-chart"
	chartPath, err := chartutil.Create(chartName, tmp)
	assert.Nil(err, fmt.Errorf("failed to create temp chart: %w", err))

	source, err := GetChartSource(chartPath)
	assert.Nil(err, fmt.Errorf("failed to get chart source: %w", err))

	ch, err := loader.LoadArchive(bytes.NewReader(source))
	assert.Nil(err, fmt.Errorf("failed to load chart source: %w", err))

	assert.Equal(ch.Name(), chartName)
}
