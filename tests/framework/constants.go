package framework

const (
	// test tag prefix, for NS labeling
	osmTest = "osmTest"

	// osmCABundleName is the name of the secret used to store the CA bundle
	osmCABundleName = "osm-ca-bundle"
)

const (
	// Test is to mark after-test cleanup
	Test CleanupType = "test"

	//Suite is to mark after-suite cleanup
	Suite CleanupType = "suite"
)

const (
	// default name for the mesh
	defaultOsmNamespace = "osm-system"

	// default MeshConfig name
	defaultMeshConfigName = "osm-mesh-config"

	// default image tag
	defaultImageTag = "latest"

	// default cert manager
	defaultCertManager = "tresor"

	// default envoy loglevel
	defaultEnvoyLogLevel = "debug"

	// default OSM loglevel
	defaultOSMLogLevel = "trace"

	// Test folder base default value
	testFolderBase = "/tmp"
)

const (
	// SelfInstall uses current kube cluster, installs OSM using CLI
	SelfInstall InstallType = "SelfInstall"

	// KindCluster Creates Kind cluster on docker and uses it as cluster, OSM installs through CLI
	KindCluster InstallType = "KindCluster"

	// NoInstall uses current kube cluster, assumes an OSM is present in `OsmNamespace`
	NoInstall InstallType = "NoInstall"

	// RegistrySecretName is the default name for the container registry secret
	RegistrySecretName = "acr-creds"
)

const (
	// CollectLogs is used to force log collection
	CollectLogs CollectLogsType = "yes"

	// CollectLogsIfErrorOnly will collect logs only when test errors out
	CollectLogsIfErrorOnly CollectLogsType = "ifError"

	// NoCollectLogs will not collect logs
	NoCollectLogs CollectLogsType = "no"

	// ControlPlaneOnly will collect logs only for control plane processes
	ControlPlaneOnly CollectLogsType = "controlPlaneOnly"
)
