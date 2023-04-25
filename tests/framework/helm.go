package framework

import (
	"fmt"
)

// HelmInstallOSM installs an osm control plane using the osm chart which lives in charts/osm
func (td *OsmTestData) HelmInstallOSM(release, namespace string) error {
	if td.InstType == KindCluster {
		if err := td.LoadOSMImagesIntoKind(); err != nil {
			return err
		}
	}

	values := fmt.Sprintf("osm.image.registry=%s,osm.image.tag=%s,osm.meshName=%s", td.CtrRegistryServer, td.OsmImageTag, release)
	args := []string{"install", release, "../../charts/osm", "--set", values, "--namespace", namespace, "--create-namespace", "--wait"}
	stdout, stderr, err := td.RunLocal("helm", args...)
	if err != nil {
		td.T.Logf("stdout:\n%s", stdout)
		return fmt.Errorf("failed to run helm install with osm chart: %s", stderr)
	}

	return nil
}

// HelmInstallOSMContour installs an osm control plane using the osm chart which lives in charts/osm with contour
func (td *OsmTestData) HelmInstallOSMContour(release, namespace string) error {
	if td.InstType == KindCluster {
		if err := td.LoadOSMImagesIntoKind(); err != nil {
			return err
		}
	}

	values := fmt.Sprintf("osm.image.registry=%s,osm.image.tag=%s,osm.meshName=%s,contour.enabled=%s,contour.configInline.tls.envoy-client-certificate.name=%s,contour.configInline.tls.envoy-client-certificate.namespace=%s", td.CtrRegistryServer, td.OsmImageTag, release, "true", "osm-contour-envoy-client-cert", namespace)
	args := []string{"install", release, "../../charts/osm", "--set", values, "--namespace", namespace, "--create-namespace", "--debug", "--wait"}
	stdout, stderr, err := td.RunLocal("helm", args...)
	if err != nil {
		td.T.Logf("stdout:\n%s", stdout)
		return fmt.Errorf("failed to run helm install with osm chart: %s", stderr)
	}

	return nil
}

// DeleteHelmRelease uninstalls a particular helm release
func (td *OsmTestData) DeleteHelmRelease(name, namespace string) error {
	args := []string{"uninstall", name, "--namespace", namespace}
	_, _, err := td.RunLocal("helm", args...)
	if err != nil {
		td.T.Fatal(err)
	}
	return nil
}
