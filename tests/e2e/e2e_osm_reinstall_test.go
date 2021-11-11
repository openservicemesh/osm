package e2e

import (
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test reinstalling OSM in the same namespace with the same mesh name",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 4,
	},
	func() {
		It("Becomes ready after being reinstalled", func() {
			opts := Td.GetOSMInstallOpts()
			Expect(Td.InstallOSM(opts)).To(Succeed())

			By("Uninstalling OSM")
			stdout, stderr, err := Td.RunLocal(filepath.FromSlash("../../bin/osm"), "uninstall", "mesh", "-f", "--osm-namespace", opts.ControlPlaneNS)
			Td.T.Log(stdout)
			if err != nil {
				Td.T.Logf("stderr:\n%s", stderr)
			}
			Expect(err).NotTo(HaveOccurred())

			By("Reinstalling OSM")
			// Invoke the CLI directly because Td.InstallOSM unconditionally
			// creates the namespace which fails when it already exists.
			stdout, stderr, err = Td.RunLocal(filepath.FromSlash("../../bin/osm"), "install", "--verbose", "--timeout=5m", "--osm-namespace", opts.ControlPlaneNS, "--set", "osm.image.registry="+opts.ContainerRegistryLoc+",osm.image.tag="+opts.OsmImagetag)
			Td.T.Log(stdout)
			if err != nil {
				Td.T.Logf("stderr:\n%s", stderr)
			}
			Expect(err).NotTo(HaveOccurred())
		})
	})
