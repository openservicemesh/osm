package cli

import (
	"io/ioutil"
	"os"

	helm "helm.sh/helm/v3/pkg/action"
)

// GetChartSource is a helper to convert a filepath to a chart to a
// base64-encoded, gzipped tarball.
func GetChartSource(path string) ([]byte, error) {
	pack := helm.NewPackage()
	packagedPath, err := pack.Run(path, nil)
	if err != nil {
		return nil, err
	}
	//nolint: errcheck
	//#nosec G307
	defer os.Remove(packagedPath)
	packaged, err := ioutil.ReadFile(packagedPath) // #nosec G304
	if err != nil {
		return nil, err
	}
	return packaged, nil
}
