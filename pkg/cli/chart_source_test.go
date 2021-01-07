package cli

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	tassert "github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestChartSource(t *testing.T) {
	assert := tassert.New(t)

	tmp, err := ioutil.TempDir(os.TempDir(), "osm-test")
	assert.Nil(errors.Wrap(err, "failed to create temp dir"))

	defer func() {
		err := os.RemoveAll(tmp)
		if err != nil {
			t.Log("error cleaning up temp dir:", err)
		}
	}()

	chartName := "test-chart"
	chartPath, err := chartutil.Create(chartName, tmp)
	assert.Nil(errors.Wrap(err, "failed to create temp chart"))

	source, err := GetChartSource(chartPath)
	assert.Nil(errors.Wrap(err, "failed to get chart source"))

	ch, err := LoadChart(source)
	assert.Nil(errors.Wrap(err, "failed to load chart source"))

	assert.Equal(ch.Name(), chartName)
}
