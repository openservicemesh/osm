package framework

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// AddNsToMesh Adds monitored namespaces to the OSM mesh
func (td *OsmTestData) AddNsToMesh(enableSidecarInjection bool, ns ...string) error {
	td.T.Logf("Adding Namespaces [+%s] to the mesh", ns)
	for _, namespace := range ns {
		args := []string{"namespace", "add", namespace}
		if !enableSidecarInjection {
			args = append(args, "--disable-sidecar-injection")
		}

		stdout, stderr, err := td.RunLocal(filepath.FromSlash("../../bin/osm"), args...)
		if err != nil {
			td.T.Logf("error running osm namespace add")
			td.T.Logf("stdout:\n%s", stdout)
			td.T.Logf("stderr:\n%s", stderr)
			return fmt.Errorf("failed to run osm namespace add: %w", err)
		}

		if Td.EnableNsMetricTag {
			args = []string{"metrics", "enable", "--namespace", namespace}
			stdout, stderr, err = td.RunLocal(filepath.FromSlash("../../bin/osm"), args...)
			if err != nil {
				td.T.Logf("error running osm namespace add")
				td.T.Logf("stdout:\n%s", stdout)
				td.T.Logf("stderr:\n%s", stderr)
				return fmt.Errorf("failed to run osm namespace add: %w", err)
			}
		}
	}
	return nil
}

// WaitForNamespacesDeleted waits for the namespaces to be deleted.
// Reference impl taken from https://github.com/kubernetes/kubernetes/blob/master/test/e2e/framework/util.go#L258
func (td *OsmTestData) WaitForNamespacesDeleted(namespaces []string, timeout time.Duration) error {
	ginkgo.By(fmt.Sprintf("Waiting for namespaces %v to vanish", namespaces))
	nsMap := map[string]bool{}
	for _, ns := range namespaces {
		nsMap[ns] = true
	}
	//Now POLL until all namespaces have been eradicated.
	return wait.Poll(2*time.Second, timeout,
		func() (bool, error) {
			nsList, err := td.Client.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
			if err != nil {
				return false, err
			}
			for _, item := range nsList.Items {
				if _, ok := nsMap[item.Name]; ok {
					return false, nil
				}
			}
			return true, nil
		})
}

// GetTestNamespaceSelectorMap returns a string-based selector used to refer/select all namespace
// resources for this test
func (td *OsmTestData) GetTestNamespaceSelectorMap() map[string]string {
	return map[string]string{
		osmTest: fmt.Sprintf("%d", ginkgo.GinkgoRandomSeed()),
	}
}
