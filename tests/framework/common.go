package framework

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/docker/docker/client"
	"github.com/fatih/color"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certman "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	policyV1alpha1Client "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/utils"
)

// Td the global context for test.
var Td OsmTestData

// Since parseFlags is global, this is the Ginkgo way to do it.
// "init" is usually called by the go test runtime
// https://github.com/onsi/ginkgo/issues/265
func init() {
	registerFlags(&Td)
}

// Cleanup when error
var _ = BeforeEach(func() {
	Expect(Td.InitTestData(GinkgoT())).To(BeNil())
})

// Cleanup when error
var _ = AfterEach(func() {
	Td.Cleanup(Test)
})

var _ = AfterSuite(func() {
	Td.Cleanup(Suite)
})

func (o OSMDescribeInfo) String() string {
	return fmt.Sprintf("[Tier %d][Bucket %d][%s]", o.Tier, o.Bucket, o.OS)
}

// OSMDescribe givens the description of an e2e test
func OSMDescribe(name string, opts OSMDescribeInfo, body func()) bool {
	return Describe(fmt.Sprintf("%s %s", opts, name), body)
}

const (
	// DefaultUpstreamServicePort is the default port on which the server apps listen for connections from client apps.
	// Note: Port 80 should not be used because it does not work on OpenShift.
	DefaultUpstreamServicePort = 14001
)

// HttpbinCmd is the command to be used for httpbin applications in e2es
var HttpbinCmd = []string{"gunicorn", "-b", fmt.Sprintf("0.0.0.0:%d", DefaultUpstreamServicePort), "httpbin:app", "-k", "gevent"}

// Verifies the instType string flag option is a valid enum type
func verifyValidInstallType(t InstallType) error {
	switch t {
	case SelfInstall, KindCluster, NoInstall:
		return nil
	default:
		return errors.Errorf("%s is not a valid InstallType (%s, %s, %s) ",
			t, SelfInstall, KindCluster, NoInstall)
	}
}

// Verifies the instType string flag option is a valid enum type
func verifyValidCollectLogs(t CollectLogsType) error {
	switch t {
	case CollectLogs, CollectLogsIfErrorOnly, NoCollectLogs, ControlPlaneOnly:
		return nil
	default:
		return errors.Errorf("%s is not a valid CollectLogsType (%s, %s, %s)",
			t, CollectLogs, CollectLogsIfErrorOnly, NoCollectLogs)
	}
}

// Function to run at init before Ginkgo has called parseFlags
// See suite_test.go for details on how Ginko calls parseFlags
func registerFlags(td *OsmTestData) {
	flag.BoolVar(&td.CleanupTest, "cleanupTest", true, "Cleanup test resources when done")
	flag.BoolVar(&td.WaitForCleanup, "waitForCleanup", true, "Wait for effective deletion of resources")
	flag.BoolVar(&td.IgnoreRestarts, "ignoreRestarts", false, "When true, will not make tests fail if restarts of control plane processes are observed")

	flag.StringVar(&td.TestDirBase, "testDirBase", testFolderBase, "Test directory base. Test directory name will be created inside.")

	flag.StringVar((*string)(&td.InstType), "installType", string(SelfInstall), "Type of install/deployment for OSM")
	flag.StringVar((*string)(&td.CollectLogs), "collectLogs", string(CollectLogsIfErrorOnly), "Defines if/when to collect logs.")

	flag.StringVar(&td.ClusterName, "kindClusterName", "osm-e2e", "Name of the Kind cluster to be created")

	flag.BoolVar(&td.CleanupKindCluster, "cleanupKindCluster", true, "Cleanup kind cluster upon exit")
	flag.BoolVar(&td.CleanupKindClusterBetweenTests, "cleanupKindClusterBetweenTests", false, "Cleanup kind cluster between tests")
	flag.StringVar(&td.ClusterVersion, "kindClusterVersion", "", "Kind cluster version, ex. v.1.20.2")

	flag.StringVar(&td.CtrRegistryServer, "ctrRegistry", os.Getenv("CTR_REGISTRY"), "Container registry")
	flag.StringVar(&td.CtrRegistryUser, "ctrRegistryUser", os.Getenv("CTR_REGISTRY_USER"), "Container registry")
	flag.StringVar(&td.CtrRegistryPassword, "ctrRegistrySecret", os.Getenv("CTR_REGISTRY_PASSWORD"), "Container registry secret")

	flag.StringVar(&td.OsmImageTag, "osmImageTag", utils.GetEnv("CTR_TAG", defaultImageTag), "OSM image tag")
	flag.StringVar(&td.OsmNamespace, "OsmNamespace", utils.GetEnv("K8S_NAMESPACE", defaultOsmNamespace), "OSM Namespace")
	flag.StringVar(&td.OsmMeshConfigName, "OsmMeshConfig", defaultMeshConfigName, "OSM MeshConfig name")

	flag.BoolVar(&td.EnableNsMetricTag, "EnableMetricsTag", true, "Enable tagging Namespaces for metrics collection")
	flag.BoolVar(&td.DeployOnOpenShift, "deployOnOpenShift", false, "Configure tests to run on OpenShift")
	flag.BoolVar(&td.DeployOnWindowsWorkers, "deployOnWindowsWorkers", false, "Configure tests to run on Windows workers")
	flag.BoolVar(&td.RetryAppPodCreation, "retryAppPodCreation", true, "Retry app pod creation on error")
}

// ValidateStringParams validates input string parameters are valid
func (td *OsmTestData) ValidateStringParams() error {
	err := verifyValidInstallType(td.InstType)
	if err != nil {
		return err
	}

	err = verifyValidCollectLogs(td.CollectLogs)
	if err != nil {
		return err
	}
	return nil
}

// GetTestDirPath Returns absolute TestDirPath
func (td *OsmTestData) GetTestDirPath() string {
	absPath, err := filepath.Abs(strings.Join([]string{td.TestDirBase, td.TestDirName}, "/"))
	if err != nil {
		td.T.Errorf("Error getting TestDirAbsPath: %v", err)
	}
	return absPath
}

// GetTestFilePath Returns absolute filepath for a filename. Will ensure TestFolder already exists.
// Convenience function used to get a proper filepath when creating a file in TestDir
func (td *OsmTestData) GetTestFilePath(filename string) string {
	testDirPath := td.GetTestDirPath()

	err := os.Mkdir(testDirPath, 0750)
	if err != nil && !os.IsExist(err) {
		td.T.Errorf("Error on Mkdir for %s: %v", testDirPath, err)
	}

	absPath, err := filepath.Abs(strings.Join([]string{testDirPath, filename}, "/"))
	if err != nil {
		td.T.Errorf("Error computing TestDirAbsPath: %v", err)
	}
	return absPath
}

// AreRegistryCredsPresent checks if Registry Credentials are present
// It's usually used to factor if a docker registry secret and ImagePullSecret
// should be installed when creating namespaces and application templates
func (td *OsmTestData) AreRegistryCredsPresent() bool {
	return len(td.CtrRegistryUser) > 0 && len(td.CtrRegistryPassword) > 0
}

// InitTestData Initializes the test structures
// Called by Ginkgo BeforeEach
func (td *OsmTestData) InitTestData(t GinkgoTInterface) error {
	td.T = t

	// Generate test id
	r, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
	if err != nil {
		return err
	}
	td.TestID = r.Uint64()
	td.TestDirName = fmt.Sprintf("test-%d", td.TestID)
	td.T.Log(color.HiGreenString("> ID for test: %d, Test dir (abs): %s", td.TestID, td.GetTestDirPath()))

	if td.DeployOnWindowsWorkers {
		td.ClusterOS = constants.OSWindows
		//TODO(#4027): Make the timeouts equal on all platforms.
		td.ReqSuccessTimeout = 5 * 60 * time.Second
		td.PodDeploymentTimeout = 240 * time.Second
	} else {
		td.ClusterOS = constants.OSLinux
		td.ReqSuccessTimeout = 60 * time.Second
		td.PodDeploymentTimeout = 90 * time.Second
	}

	// String parameter validation
	err = td.ValidateStringParams()
	if err != nil {
		return err
	}

	if (td.InstType == KindCluster) && td.ClusterProvider == nil {
		td.ClusterProvider = cluster.NewProvider()
		td.T.Logf("Creating local kind cluster")
		clusterConfig := &v1alpha4.Cluster{
			Nodes: []v1alpha4.Node{
				{
					Role: v1alpha4.ControlPlaneRole,
				},
				{
					Role: v1alpha4.WorkerRole,
					KubeadmConfigPatches: []string{`kind: JoinConfiguration
nodeRegistration:
  kubeletExtraArgs:
    node-labels: "ingress-ready=true"`},
					ExtraPortMappings: []v1alpha4.PortMapping{
						{
							ContainerPort: 80,
							HostPort:      80,
							Protocol:      v1alpha4.PortMappingProtocolTCP,
						},
					},
				},
				{
					Role: v1alpha4.WorkerRole,
				},
			},
		}
		if Td.ClusterVersion != "" {
			for i := 0; i < len(clusterConfig.Nodes); i++ {
				clusterConfig.Nodes[i].Image = fmt.Sprintf("kindest/node:%s", td.ClusterVersion)
			}
		}
		if err := td.ClusterProvider.Create(td.ClusterName, cluster.CreateWithV1Alpha4Config(clusterConfig)); err != nil {
			return errors.Wrap(err, "failed to create kind cluster")
		}
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	kubeConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get Kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create Kubernetes client")
	}

	configClient, err := configClientset.NewForConfig(kubeConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s client", configv1alpha3.SchemeGroupVersion)
	}

	policyClient, err := policyV1alpha1Client.NewForConfig(kubeConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to create %s client", policyV1alpha1.SchemeGroupVersion)
	}

	apiServerClient, err := apiclientset.NewForConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create api server client")
	}

	td.RestConfig = kubeConfig
	td.Client = clientset
	td.ConfigClient = configClient
	td.PolicyClient = policyClient
	td.APIServerClient = apiServerClient

	td.Env = cli.New()

	if err := td.InitSMIClients(); err != nil {
		return errors.Wrap(err, "failed to initialize SMI clients")
	}

	// After client creations, do a wait for kind cluster just in case it's not done yet coming up
	// Ballparking pod number. kind has a large number of containers to run by default
	if (td.InstType == KindCluster) && td.ClusterProvider != nil {
		if err := td.WaitForPodsRunningReady("kube-system", 120*time.Second, 5, nil); err != nil {
			return errors.Wrap(err, "failed to wait for kube-system pods")
		}
	}

	k8sServerVersion, err := Td.getKubernetesServerVersionNumber()
	if err != nil {
		return errors.Wrap(err, "Error getting k8s server version")
	}

	// Logs v<major>.<minor>.<patch>
	td.T.Logf("> k8s server version: v%s\n", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(k8sServerVersion)), "."), "[]"))
	return nil
}

// GetOSMInstallOpts initializes install options for OSM
func (td *OsmTestData) GetOSMInstallOpts() InstallOSMOpts {
	enablePrivilegedInitContainer := false
	if td.DeployOnOpenShift {
		enablePrivilegedInitContainer = true
	}
	return InstallOSMOpts{
		ControlPlaneNS:          td.OsmNamespace,
		CertManager:             defaultCertManager,
		ContainerRegistryLoc:    td.CtrRegistryServer,
		ContainerRegistrySecret: td.CtrRegistryPassword,
		OsmImagetag:             td.OsmImageTag,
		DeployGrafana:           false,
		DeployPrometheus:        false,
		DeployJaeger:            false,
		DeployFluentbit:         false,
		EnableReconciler:        false,

		VaultHost:     "vault." + td.OsmNamespace + ".svc.cluster.local",
		VaultProtocol: "http",
		VaultRole:     "openservicemesh",
		VaultToken:    "token",

		CertmanagerIssuerGroup: "cert-manager.io",
		CertmanagerIssuerKind:  "Issuer",
		CertmanagerIssuerName:  "osm-ca",
		CertKeyBitSize:         2048,
		CertValidtyDuration:    time.Hour * 24,
		EnvoyLogLevel:          defaultEnvoyLogLevel,
		OSMLogLevel:            defaultOSMLogLevel,
		EnableDebugServer:      true,
		SetOverrides:           []string{},

		EnablePrivilegedInitContainer: enablePrivilegedInitContainer,
		EnableIngressBackendPolicy:    true,
	}
}

// LoadImagesToKind loads the list of images to the node for Kind clusters
func (td *OsmTestData) LoadImagesToKind(imageNames []string) error {
	if td.InstType != KindCluster {
		td.T.Log("Not a Kind cluster, nothing to load")
		return nil
	}

	td.T.Log("Getting image data")
	docker, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}
	var imageIDs []string
	for _, name := range imageNames {
		imageName := fmt.Sprintf("%s/%s:%s", td.CtrRegistryServer, name, td.OsmImageTag)
		imageIDs = append(imageIDs, imageName)
	}
	imageData, err := docker.ImageSave(context.TODO(), imageIDs)
	if err != nil {
		return errors.Wrap(err, "failed to get image data")
	}

	imageReader, err := ioutil.ReadAll(imageData)
	if err != nil {
		return errors.Wrap(err, "failed to read images")
	}

	reader := bytes.NewReader(imageReader)
	//nolint: errcheck
	//#nosec G307
	defer imageData.Close()
	nodes, err := td.ClusterProvider.ListNodes(td.ClusterName)
	if err != nil {
		return errors.Wrap(err, "failed to list kind nodes")
	}

	for _, n := range nodes {
		td.T.Log("Loading images onto node", n)
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			return errors.Wrap(err, "failed to reset images")
		}
		if err = nodeutils.LoadImageArchive(n, reader); err != nil {
			return errors.Wrap(err, "failed to load images")
		}
	}

	return nil
}

func setMeshConfigToDefault(instOpts InstallOSMOpts, meshConfig *configv1alpha3.MeshConfig) (defaultConfig *configv1alpha3.MeshConfig) {
	meshConfig.Spec.Traffic.EnableEgress = instOpts.EgressEnabled
	meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode = instOpts.EnablePermissiveMode
	meshConfig.Spec.Traffic.OutboundPortExclusionList = []int{}
	meshConfig.Spec.Traffic.OutboundIPRangeExclusionList = []string{}

	meshConfig.Spec.Observability.EnableDebugServer = instOpts.EnableDebugServer

	meshConfig.Spec.Sidecar.Resources = corev1.ResourceRequirements{}
	meshConfig.Spec.Sidecar.EnablePrivilegedInitContainer = instOpts.EnablePrivilegedInitContainer
	meshConfig.Spec.Sidecar.LogLevel = instOpts.EnvoyLogLevel
	meshConfig.Spec.Sidecar.MaxDataPlaneConnections = 0
	meshConfig.Spec.Sidecar.ConfigResyncInterval = "0s"

	meshConfig.Spec.Certificate.ServiceCertValidityDuration = instOpts.CertValidtyDuration.String()
	meshConfig.Spec.Certificate.CertKeyBitSize = instOpts.CertKeyBitSize

	meshConfig.Spec.FeatureFlags.EnableIngressBackendPolicy = instOpts.EnableIngressBackendPolicy

	return meshConfig
}

// InstallOSM installs OSM. The behavior of this function is dependant on
// installType and instOpts
func (td *OsmTestData) InstallOSM(instOpts InstallOSMOpts) error {
	if td.InstType == NoInstall {
		if instOpts.CertManager != defaultCertManager || instOpts.DeployPrometheus || instOpts.DeployGrafana || instOpts.DeployJaeger || instOpts.DeployFluentbit || instOpts.EnableReconciler {
			Skip("Skipping test: NoInstall marked on a test that requires modified install")
		}

		// Store current restart values for CTL processes
		td.InitialRestartValues = td.GetOsmCtlComponentRestarts()

		meshConfig, _ := Td.GetMeshConfig(Td.OsmNamespace)
		meshConfig = setMeshConfigToDefault(instOpts, meshConfig)
		if _, err := Td.UpdateOSMConfig(meshConfig); err != nil {
			return err
		}

		return nil
	}

	if td.InstType == KindCluster {
		if err := td.LoadOSMImagesIntoKind(); err != nil {
			return errors.Wrap(err, "failed to load OSM images to nodes for Kind cluster")
		}
	}

	if err := td.CreateNs(instOpts.ControlPlaneNS, nil); err != nil {
		return errors.Wrap(err, "failed to create namespace "+instOpts.ControlPlaneNS)
	}

	var args []string
	args = append(args, "install",
		"--osm-namespace="+instOpts.ControlPlaneNS,
		"--verbose",
		fmt.Sprintf("--timeout=%v", 90*time.Second),
	)

	instOpts.SetOverrides = append(instOpts.SetOverrides,
		fmt.Sprintf("osm.image.registry=%s", instOpts.ContainerRegistryLoc),
		fmt.Sprintf("osm.image.tag=%s", instOpts.OsmImagetag),
		fmt.Sprintf("osm.certificateProvider.kind=%s", instOpts.CertManager),
		fmt.Sprintf("osm.enableEgress=%v", instOpts.EgressEnabled),
		fmt.Sprintf("osm.enablePermissiveTrafficPolicy=%v", instOpts.EnablePermissiveMode),
		fmt.Sprintf("osm.enableDebugServer=%v", instOpts.EnableDebugServer),
		fmt.Sprintf("osm.envoyLogLevel=%s", instOpts.EnvoyLogLevel),
		fmt.Sprintf("osm.deployGrafana=%v", instOpts.DeployGrafana),
		fmt.Sprintf("osm.deployPrometheus=%v", instOpts.DeployPrometheus),
		fmt.Sprintf("osm.deployJaeger=%v", instOpts.DeployJaeger),
		fmt.Sprintf("osm.enableFluentbit=%v", instOpts.DeployFluentbit),
		fmt.Sprintf("osm.enablePrivilegedInitContainer=%v", instOpts.EnablePrivilegedInitContainer),
		fmt.Sprintf("osm.featureFlags.enableIngressBackendPolicy=%v", instOpts.EnableIngressBackendPolicy),
		fmt.Sprintf("osm.enableReconciler=%v", instOpts.EnableReconciler),
	)

	switch instOpts.CertManager {
	case "vault":
		if err := td.installVault(instOpts); err != nil {
			return err
		}
		instOpts.SetOverrides = append(instOpts.SetOverrides,
			fmt.Sprintf("osm.vault.host=%s", instOpts.VaultHost),
			fmt.Sprintf("osm.vault.role=%s", instOpts.VaultRole),
			fmt.Sprintf("osm.vault.protocol=%s", instOpts.VaultProtocol),
			fmt.Sprintf("osm.vault.token=%s", instOpts.VaultToken))
		// Wait for the vault pod
		if err := td.WaitForPodsRunningReady(instOpts.ControlPlaneNS, 60*time.Second, 1, nil); err != nil {
			return errors.Wrap(err, "failed waiting for vault pod to become ready")
		}
	case "cert-manager":
		if err := td.installCertManager(instOpts); err != nil {
			return err
		}
		instOpts.SetOverrides = append(instOpts.SetOverrides,
			fmt.Sprintf("osm.certmanager.issuerName=%s", instOpts.CertmanagerIssuerName),
			fmt.Sprintf("osm.certmanager.issuerKind=%s", instOpts.CertmanagerIssuerKind),
			fmt.Sprintf("osm.certmanager.issuerGroup=%s", instOpts.CertmanagerIssuerGroup))
	}

	if !(td.InstType == KindCluster) {
		// Making sure the image is always pulled in registry-based testing
		instOpts.SetOverrides = append(instOpts.SetOverrides,
			"osm.image.pullPolicy=Always")
	}

	if len(instOpts.ContainerRegistrySecret) != 0 {
		instOpts.SetOverrides = append(instOpts.SetOverrides,
			fmt.Sprintf("osm.imagePullSecrets[0].name=%s", RegistrySecretName),
		)
	}

	td.T.Logf("Setting log OSM's log level through overrides to %s", instOpts.OSMLogLevel)
	instOpts.SetOverrides = append(instOpts.SetOverrides,
		fmt.Sprintf("osm.controllerLogLevel=%s", instOpts.OSMLogLevel))

	if len(instOpts.SetOverrides) > 0 {
		separator := "="
		finalLine := "--set"
		for _, override := range instOpts.SetOverrides {
			finalLine = finalLine + separator + override
			separator = ","
		}
		args = append(args, finalLine)
	}

	td.T.Log("Installing OSM")
	stdout, stderr, err := td.RunLocal(filepath.FromSlash("../../bin/osm"), args...)
	if err != nil {
		td.T.Logf("error running osm install")
		td.T.Logf("stdout:\n%s", stdout)
		td.T.Logf("stderr:\n%s", stderr)
		return errors.Wrap(err, "failed to run osm install")
	}

	// Ensure osm-injector, osm-controller and osm-bootstrap are ready
	err = td.waitForOSMControlPlane(30 * time.Second)
	if err != nil {
		return err
	}

	// We need to set some values to the mesh config to disable WASM which is not supported on Windows
	// For testing we have to use the OSM dev image which matches the underlying OS.
	if td.ClusterOS == constants.OSWindows {
		meshConfig, _ := Td.GetMeshConfig(Td.OsmNamespace)
		meshConfig.Spec.FeatureFlags.EnableWASMStats = false
		meshConfig.Spec.Sidecar.EnvoyWindowsImage = EnvoyOSMWindowsImage
		_, err = Td.UpdateOSMConfig(meshConfig)
		if err != nil {
			return err
		}
	}

	// Store current restart values for CTL processes
	td.InitialRestartValues = td.GetOsmCtlComponentRestarts()

	return nil
}

// RestartOSMController restarts the osm-controller pod in the installed controller's namespace
func (td *OsmTestData) RestartOSMController(instOpts InstallOSMOpts) error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	controllerPods, err := td.Client.CoreV1().Pods(instOpts.ControlPlaneNS).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error fetching controller pod")
	}

	controllerDeployment, errDeployment := td.Client.AppsV1().Deployments(instOpts.ControlPlaneNS).Get(context.TODO(), constants.OSMControllerName, metav1.GetOptions{})
	if errDeployment != nil {
		return errors.Wrap(err, "error fetching controller deployment")
	}

	expectedReplicaCount := int(*(controllerDeployment.Spec.Replicas))
	if len(controllerPods.Items) != expectedReplicaCount {
		return errors.Errorf("expected %d osm-controller pod(s), got %d", expectedReplicaCount, len(controllerPods.Items))
	}

	pod := controllerPods.Items[0]

	// Delete the pod and let k8s spin it up again
	err = td.Client.CoreV1().Pods(instOpts.ControlPlaneNS).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "error deleting osm-controller pod")
	}

	return nil
}

// GetMeshConfig is a wrapper to get a MeshConfig by name in a particular namespace
func (td *OsmTestData) GetMeshConfig(namespace string) (*configv1alpha3.MeshConfig, error) {
	meshConfig, err := td.ConfigClient.ConfigV1alpha2().MeshConfigs(namespace).Get(context.TODO(), td.OsmMeshConfigName, v1.GetOptions{})

	if err != nil {
		return nil, err
	}
	return meshConfig, nil
}

// LoadOSMImagesIntoKind loads the OSM images to the node for Kind clusters
func (td *OsmTestData) LoadOSMImagesIntoKind() error {
	imageNames := []string{
		constants.OSMControllerName,
		constants.OSMInjectorName,
		"init",
		"osm-crds",
		constants.OSMBootstrapName,
		"osm-preinstall",
		"osm-healthcheck",
	}

	return td.LoadImagesToKind(imageNames)
}

func (td *OsmTestData) installVault(instOpts InstallOSMOpts) error {
	td.T.Log("Installing Vault")

	appName := "vault"
	replicas := int32(1)
	terminationGracePeriodSeconds := int64(10)

	serviceAccountDefinition := Td.SimpleServiceAccount(appName, td.OsmNamespace)
	svcAccount, err := Td.CreateServiceAccount(serviceAccountDefinition.Namespace, &serviceAccountDefinition)
	if err != nil {
		return errors.Wrap(err, "failed to create vault service account")
	}

	vaultDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				constants.AppLabel: appName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.AppLabel: appName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.AppLabel: appName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            svcAccount.Name,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:            "vault",
							Image:           "vault:1.4.0",
							ImagePullPolicy: corev1.PullAlways,
							Command:         []string{"/bin/sh", "-c"},
							Args: []string{
								fmt.Sprintf(`
# The TTL for the expiration of CA certificate must be beyond that of the longest
# TTL for a certificate issued by OSM. The longest TTL for a certificate issued
# within OSM is 87600h.

# Start the Vault Server
vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=%s & sleep 1;

# Make the token available to the following commands
echo %s>~/.vault-token;

# Enable PKI secrets engine
vault secrets enable pki;

# Set the max allowed lease for a certificate to a decade
vault secrets tune -max-lease-ttl=87700h pki;

# Set the URLs (See: https://www.vaultproject.io/docs/secrets/pki#set-url-configuration)
vault write pki/config/urls issuing_certificates='http://127.0.0.1:8200/v1/pki/ca' crl_distribution_points='http://127.0.0.1:8200/v1/pki/crl';

# Configure a role for OSM (See: https://www.vaultproject.io/docs/secrets/pki#configure-a-role)
vault write pki/roles/%s allow_any_name=true allow_subdomains=true max_ttl=87700h;

# Create the root certificate (See: https://www.vaultproject.io/docs/secrets/pki#setup)
vault write pki/root/generate/internal common_name='osm.root' ttl='87700h';
tail /dev/random;
`, instOpts.VaultToken, instOpts.VaultToken, instOpts.VaultRole),
							},
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"IPC_LOCK",
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8200,
									Name:          "vault-port",
									Protocol:      corev1.ProtocolTCP,
								},
								{
									ContainerPort: 8201,
									Name:          "cluster-port",
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "VAULT_ADDR",
									Value: "http://localhost:8200",
								},
								{
									Name: "POD_IP_ADDR",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "VAULT_LOCAL_CONFIG",
									Value: "api_addr = \"http://127.0.0.1:8200\"\ncluster_addr = \"http://${POD_IP_ADDR}:8201\"",
								},
								{
									Name:  "VAULT_DEV_ROOT_TOKEN_ID",
									Value: "root", // THIS IS NOT A PRODUCTION DEPLOYMENT OF VAULT!
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/v1/sys/health",
										Port:   intstr.FromInt(8200),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
				},
			},
		},
	}
	_, err = td.Client.AppsV1().Deployments(instOpts.ControlPlaneNS).Create(context.TODO(), vaultDep, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create vault deployment")
	}

	vaultSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				constants.AppLabel: appName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				constants.AppLabel: appName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "vault-port",
					Port:       8200,
					TargetPort: intstr.FromInt(8200),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
	_, err = td.Client.CoreV1().Services(instOpts.ControlPlaneNS).Create(context.TODO(), vaultSvc, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create vault service")
	}
	return nil
}

func (td *OsmTestData) installCertManager(instOpts InstallOSMOpts) error {
	By("Installing cert-manager")
	helm := &action.Configuration{}
	if err := helm.Init(td.Env.RESTClientGetter(), td.OsmNamespace, "secret", td.T.Logf); err != nil {
		return errors.Wrap(err, "failed to initialize helm config")
	}
	install := action.NewInstall(helm)
	install.RepoURL = "https://charts.jetstack.io"
	install.Namespace = td.OsmNamespace
	install.ReleaseName = "certmanager"
	install.Version = "v1.3.1"

	chartPath, err := install.LocateChart("cert-manager", helmcli.New())
	if err != nil {
		return errors.Wrap(err, "failed to get cert-manager-chart")
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return errors.Wrap(err, "failed to load cert-manager chart")
	}

	_, err = install.Run(chart, map[string]interface{}{
		"installCRDs": true,
	})
	if err != nil {
		return errors.Wrap(err, "failed to install cert-manager chart")
	}

	selfsigned := &cmapi.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "selfsigned",
		},
		Spec: cmapi.IssuerSpec{
			IssuerConfig: cmapi.IssuerConfig{
				SelfSigned: &cmapi.SelfSignedIssuer{},
			},
		},
	}

	cert := &cmapi.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-ca",
		},
		Spec: cmapi.CertificateSpec{
			IsCA:       true,
			Duration:   &metav1.Duration{Duration: 90 * 24 * time.Hour},
			SecretName: osmCABundleName,
			CommonName: "osm-system",
			IssuerRef: cmmeta.ObjectReference{
				Name:  selfsigned.Name,
				Kind:  "Issuer",
				Group: "cert-manager.io",
			},
		},
	}

	ca := &cmapi.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-ca",
		},
		Spec: cmapi.IssuerSpec{
			IssuerConfig: cmapi.IssuerConfig{
				CA: &cmapi.CAIssuer{
					SecretName: osmCABundleName,
				},
			},
		},
	}

	if err := td.WaitForPodsRunningReady(install.Namespace, 60*time.Second, 3, nil); err != nil {
		return errors.Wrap(err, "failed to wait for cert-manager pods ready")
	}

	cmClient, err := certman.NewForConfig(td.RestConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create cert-manager config")
	}

	// cert-manager.io webhook can experience connection problems after installation:
	// https://cert-manager.io/docs/concepts/webhook/#webhook-connection-problems-shortly-after-cert-manager-installation
	// Retry API errors with some delay in case of failures.
	if err = Td.RetryFuncOnError(func() error {
		_, err = cmClient.CertmanagerV1().Certificates(td.OsmNamespace).Create(context.TODO(), cert, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create Certificate "+cert.Name)
		}
		return nil
	}, 5, 5*time.Second); err != nil {
		return err
	}

	if err = Td.RetryFuncOnError(func() error {
		_, err = cmClient.CertmanagerV1().Issuers(td.OsmNamespace).Create(context.TODO(), selfsigned, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create Issuer "+selfsigned.Name)
		}
		return nil
	}, 5, 5*time.Second); err != nil {
		return err
	}

	if err = Td.RetryFuncOnError(func() error {
		_, err = cmClient.CertmanagerV1().Issuers(td.OsmNamespace).Create(context.TODO(), ca, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create Issuer "+ca.Name)
		}
		return nil
	}, 5, 5*time.Second); err != nil {
		return err
	}

	// cert-manager.io creates the OSM CA bundle secret which is required by osm-controller. Wait for it to be ready.
	if err := Td.waitForCABundleSecret(td.OsmNamespace, 90*time.Second); err != nil {
		return errors.Wrap(err, "error waiting for cert-manager.io to create OSM CA bundle secret")
	}

	return nil
}

// UpdateOSMConfig updates OSM MeshConfig
func (td *OsmTestData) UpdateOSMConfig(meshConfig *configv1alpha3.MeshConfig) (*configv1alpha3.MeshConfig, error) {
	updated, err := td.ConfigClient.ConfigV1alpha2().MeshConfigs(td.OsmNamespace).Update(context.TODO(), meshConfig, metav1.UpdateOptions{})

	if err != nil {
		td.T.Logf("UpdateOSMConfig(): %s", err)
		return nil, fmt.Errorf("UpdateOSMConfig(): %s", err)
	}
	return updated, nil
}

// CreateMultipleNs simple CreateNs for multiple NS creation
func (td *OsmTestData) CreateMultipleNs(nsName ...string) error {
	for _, ns := range nsName {
		err := td.CreateNs(ns, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateNs creates a Namespace. Will automatically add Docker registry creds if provided
func (td *OsmTestData) CreateNs(nsName string, labels map[string]string) error {
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range td.GetTestNamespaceSelectorMap() {
		labels[k] = v
	}

	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName,
			Namespace: "",
			Labels:    labels,
		},
		Status: corev1.NamespaceStatus{},
	}

	td.T.Logf("Creating namespace %v", nsName)
	_, err := td.Client.CoreV1().Namespaces().Create(context.Background(), namespaceObj, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create namespace "+nsName)
	}

	// Check if we are using any specific creds
	if td.AreRegistryCredsPresent() {
		td.CreateDockerRegistrySecret(nsName)
	}

	return nil
}

// DeleteNs deletes a test NS
func (td *OsmTestData) DeleteNs(nsName string) error {
	// Delete Helm releases created in the namespace
	helm := &action.Configuration{}
	if err := helm.Init(td.Env.RESTClientGetter(), nsName, "secret", td.T.Logf); err != nil {
		td.T.Logf("WARNING: failed to initialize helm config, skipping helm cleanup: %v", err)
	} else {
		list := action.NewList(helm)
		list.All = true
		if releases, err := list.Run(); err != nil {
			td.T.Logf("WARNING: failed to list helm releases in namespace %s, skipping release cleanup: %v", nsName, err)
		} else {
			del := action.NewUninstall(helm)
			del.Wait = true
			for _, release := range releases {
				if _, err := del.Run(release.Name); err != nil {
					td.T.Logf("WARNING: failed to delete helm release %s in namespace %s: %v", release.Name, nsName, err)
				}
			}
		}
	}

	var backgroundDelete metav1.DeletionPropagation = metav1.DeletePropagationBackground

	td.T.Logf("Deleting namespace %v", nsName)
	err := td.Client.CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{PropagationPolicy: &backgroundDelete})
	if err != nil {
		return errors.Wrap(err, "failed to delete namespace "+nsName)
	}
	return nil
}

// WaitForPodsDeleted waits for the pods to be deleted.
func (td *OsmTestData) WaitForPodsDeleted(pods *corev1.PodList, namespace string, timeout time.Duration) error {
	By(fmt.Sprintf("Waiting for pods to vanish from namespace %s", namespace))
	podMap := map[string]bool{}
	for _, pod := range pods.Items {
		podMap[string(pod.GetUID())] = true
	}
	//Now POLL until all pods have been eradicated.
	return wait.Poll(2*time.Second, timeout,
		func() (bool, error) {
			podList, err := td.Client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			for _, item := range podList.Items {
				if _, ok := podMap[string(item.GetUID())]; ok {
					return false, nil
				}
			}
			return true, nil
		})
}

// RunLocal Executes command on local
func (td *OsmTestData) RunLocal(path string, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	cmd := exec.Command(path, args...) // #nosec G204
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	td.T.Logf("Running locally '%s %s'", path, strings.Join(args, " "))
	err := cmd.Run()
	return stdout, stderr, err
}

// RunRemote runs command in remote container
func (td *OsmTestData) RunRemote(
	ns string, podName string, containerName string,
	command []string) (string, string, error) {
	var stdin, stdout, stderr bytes.Buffer

	req := td.Client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(ns).SubResource("exec")

	option := &corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}

	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		return "", "", err
	}

	req.VersionedParams(
		option,
		runtime.NewParameterCodec(scheme),
	)
	exec, err := remotecommand.NewSPDYExecutor(td.RestConfig, "POST", req.URL())
	if err != nil {
		return "", "", err
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  &stdin,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

// WaitForPodsRunningReady waits for a <n> number of pods on an NS to be running and ready
// `labelSelector` can be optionally passed to further select the pods to wait for
func (td *OsmTestData) WaitForPodsRunningReady(ns string, timeout time.Duration, nExpectedRunningPods int, labelSelector *metav1.LabelSelector) error {
	td.T.Logf("Wait up to %v for %d pods ready in ns [%s]...", timeout, nExpectedRunningPods, ns)

	listOpts := metav1.ListOptions{
		FieldSelector: "status.phase=Running",
	}

	if labelSelector != nil {
		labelMap, _ := metav1.LabelSelectorAsMap(labelSelector)
		listOpts.LabelSelector = labels.SelectorFromSet(labelMap).String()
	}

	for start := time.Now(); time.Since(start) < timeout; time.Sleep(2 * time.Second) {
		pods, err := td.Client.CoreV1().Pods(ns).List(context.TODO(), listOpts)

		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}

		if len(pods.Items) < nExpectedRunningPods {
			time.Sleep(time.Second)
			continue
		}

		nReadyPods := 0
		for _, pod := range pods.Items {
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					nReadyPods++
					if nReadyPods == nExpectedRunningPods {
						td.T.Logf("Finished waiting for NS [%s].", ns)
						return nil
					}
				}
			}
		}
		time.Sleep(time.Second)
	}

	pods, err := td.Client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list pods")
	}
	td.T.Log("Pod Statuses in namespace", ns)
	for _, pod := range pods.Items {
		status, _ := json.MarshalIndent(pod.Status, "", "  ")
		td.T.Logf("Pod %s:\n%s", pod.Name, status)
	}

	return fmt.Errorf("not all pods were Running & Ready in NS %s after %v", ns, timeout)
}

// WaitForRepeatedSuccess runs and expects a certain result for a certain operation a set number of consecutive times
// over a set amount of time.
func (td *OsmTestData) WaitForRepeatedSuccess(f SuccessFunction, minItForSuccess int, maxWaitTime time.Duration) bool {
	iterations := 0
	startTime := time.Now()

	By(fmt.Sprintf("[WaitForRepeatedSuccess] waiting %v for %d iterations to succeed", maxWaitTime, minItForSuccess))
	for time.Since(startTime) < maxWaitTime {
		if f() {
			iterations++
			if iterations >= minItForSuccess {
				return true
			}
		} else {
			iterations = 0
		}
		time.Sleep(time.Second)
	}
	return false
}

// WaitForSuccessAfterInitialFailure runs and expects a certain result for a certain operation a set number of consecutive times
// but requires only success after the first success.
func (td *OsmTestData) WaitForSuccessAfterInitialFailure(f SuccessFunction, minItForSuccess int, maxWaitTime time.Duration) bool {
	iterations := 0
	startTime := time.Now()
	successHasStarted := false

	By(fmt.Sprintf("[WaitForSuccessAfterFailureBuffer] waiting %v for %d iterations to succeed", maxWaitTime, minItForSuccess))
	for time.Since(startTime) < maxWaitTime {
		if f() {
			successHasStarted = true
			iterations++
			if iterations >= minItForSuccess {
				return true
			}
		} else if successHasStarted {
			return false
		}
		time.Sleep(time.Second)
	}
	return false
}

// Cleanup is Used to cleanup resources once the test is done
// Cyclomatic complexity is disabled in cleanup, as it check a large number of conditions
// nolint:gocyclo
func (td *OsmTestData) Cleanup(ct CleanupType) {
	if td.Client == nil {
		// Avoid any cleanup (crash) if no test is run;
		// init doesn't happen and clientsets are nil
		return
	}

	// Verify no crashes/restarts of OSM and control plane components were observed during the test
	// We will not immediately call Fail() here to not disturb the cleanup process, and instead
	// call it at the end of cleanup
	restartSeen := td.VerifyRestarts()

	// If collect logs or
	// (test failed, by either restarts were seen or because spec failed) and (collect logs on error)
	if td.CollectLogs == CollectLogs || td.CollectLogs == ControlPlaneOnly ||
		((restartSeen && !td.IgnoreRestarts) || CurrentGinkgoTestDescription().Failed) && td.CollectLogs == CollectLogsIfErrorOnly {
		// Grab logs. We will move this to use CLI when able.

		if err := td.GrabLogs(); err != nil {
			td.T.Logf("Error getting logs: %v", err)
		}

		if err := td.GetBugReport(); err != nil {
			td.T.Logf("Error getting bug report: %v", err)
		}
	}

	cleanupTrigger := td.CleanupTest
	// If we are on kind env
	if td.InstType == KindCluster {
		// Check if we can/want to avoid K8s cleanup
		cleanupTrigger = cleanupTrigger && td.shouldCleanupK8sOnKind(ct)
	}

	if cleanupTrigger {
		// Use selector to refer to all namespaces used in this test
		nsSelector := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(td.GetTestNamespaceSelectorMap()).String(),
		}

		testNs, err := td.Client.CoreV1().Namespaces().List(context.Background(), nsSelector)
		if err != nil {
			td.T.Fatalf("Failed to get list of test NS: %v", err)
		}

		for _, ns := range testNs.Items {
			err := td.DeleteNs(ns.Name)
			if err != nil {
				td.T.Logf("Err deleting ns %s: %v", ns.Name, err)
				continue
			}
		}
		By(fmt.Sprintf("[Cleanup] waiting for %s:%d test NS cleanup", osmTest, GinkgoRandomSeed()))
		if td.WaitForCleanup {
			err := wait.Poll(2*time.Second, 240*time.Second,
				func() (bool, error) {
					nsList, err := td.Client.CoreV1().Namespaces().List(context.TODO(), nsSelector)
					if err != nil {
						td.T.Logf("Err waiting for ns list to disappear: %v", err)
						return false, err
					}
					return len(nsList.Items) == 0, nil
				},
			)
			if err != nil {
				td.T.Logf("Error polling namespaces for deletion: %s", err)
				testNsInfo, _ := json.MarshalIndent(testNs, "", "  ")
				td.T.Logf("Namespaces info:\n%s", string(testNsInfo))
			}
		}
	}

	// Kind cluster deletion, if needed
	if (td.InstType == KindCluster) && td.ClusterProvider != nil {
		if ct == Test && td.CleanupKindClusterBetweenTests || ct == Suite && td.CleanupKindCluster {
			td.T.Logf("Deleting kind cluster: %s", td.ClusterName)
			if err := td.ClusterProvider.Delete(td.ClusterName, clientcmd.RecommendedHomeFile); err != nil {
				td.T.Logf("error deleting cluster: %v", err)
			}
			td.ClusterProvider = nil
		}
	}

	// Check restarts
	if restartSeen && !td.IgnoreRestarts {
		Fail("Unexpected restarts for control plane processes were observed")
	}
}

// CreateDockerRegistrySecret creates a secret named `RegistrySecretName` in namespace <ns>,
// based on ctrRegistry variables
func (td *OsmTestData) CreateDockerRegistrySecret(ns string) {
	secret := &corev1.Secret{}
	secret.Name = RegistrySecretName
	secret.Type = corev1.SecretTypeDockerConfigJson
	secret.Data = map[string][]byte{}

	dockercfgAuth := DockerConfigEntry{
		Username: td.CtrRegistryUser,
		Password: td.CtrRegistryPassword,
		Email:    "osm@osm.com",
		Auth:     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", td.CtrRegistryUser, td.CtrRegistryPassword))),
	}

	dockerCfgJSON := DockerConfigJSON{
		Auths: map[string]DockerConfigEntry{td.CtrRegistryServer: dockercfgAuth},
	}

	if jsonConfig, err := json.Marshal(dockerCfgJSON); err != nil {
		td.T.Fatalf("Error marshaling Docker config", err)
	} else {
		secret.Data[corev1.DockerConfigJsonKey] = jsonConfig
	}

	td.T.Logf("Pushing Registry secret '%s' for namespace %s... ", RegistrySecretName, ns)
	_, err := td.Client.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		td.T.Fatalf("Could not add registry secret")
	}
}

// shouldCleanupK8sOnKind returns whether k8s cleanup can be avoided on kind when the
// kind cluster is not going to be kept anyway, when the test/suite is running on kind.
// This optimization considerably speeds up the test suite when running on kind as we don't wait
// for k8s cleanup after every test.
func (td *OsmTestData) shouldCleanupK8sOnKind(ct CleanupType) bool {
	cleanupK8sOnKind := false

	if ct == Test {
		// If Kind cluster is going away, no need to trigger and wait for k8s cleanup
		cleanupK8sOnKind = !td.CleanupKindClusterBetweenTests
	} else if ct == Suite {
		// If Kind cluster is going away, no need to trigger and wait for k8s cleanup
		cleanupK8sOnKind = !td.CleanupKindCluster
	}

	return cleanupK8sOnKind
}

// RetryFuncOnError runs the given function and retries for the given number of times if an error is encountered
func (td *OsmTestData) RetryFuncOnError(f RetryOnErrorFunc, retryTimes int, sleepBetweenRetries time.Duration) error {
	var err error

	for i := 0; i <= retryTimes; i++ {
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(sleepBetweenRetries)
	}
	return errors.Wrapf(err, "Error after retrying %d times", retryTimes)
}

// waitForCABundleSecret waits for the CA bundle secret to be created
func (td *OsmTestData) waitForCABundleSecret(ns string, timeout time.Duration) error {
	td.T.Logf("Wait up to %s for OSM CA bundle to be ready in NS [%s]...", timeout, ns)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(5 * time.Second) {
		_, err := td.Client.CoreV1().Secrets(ns).Get(context.TODO(), osmCABundleName, metav1.GetOptions{})
		if err == nil {
			return nil
		}
		td.T.Logf("OSM CA bundle secret not ready in NS [%s]", ns)
		continue // retry
	}

	return fmt.Errorf("CA bundle secret not ready in NS %s after %s", ns, timeout)
}

// VerifyRestarts ensure no crashes on osm-namespace instances for OSM CTL processes
func (td *OsmTestData) VerifyRestarts() bool {
	restartsAtTestEnd := td.GetOsmCtlComponentRestarts()
	restartOccurred := false

	for podContKey, endRestarts := range restartsAtTestEnd {
		initialRestarts, found := td.InitialRestartValues[podContKey]
		if !found {
			td.T.Logf("Pod/cont %s not found in initial map. Skipping.", podContKey)
			continue
		}

		if initialRestarts != endRestarts {
			td.T.Logf("!! Restarts detected for pod/cont %s: Initial %d End %d", podContKey, initialRestarts, endRestarts)
			restartOccurred = true
		}
	}
	return restartOccurred
}

// GetOsmCtlComponentRestarts gets the number of restarts of OSM CTL processes back in a map
func (td *OsmTestData) GetOsmCtlComponentRestarts() map[string]int {
	restartMap := make(map[string]int)

	// Walks only OSM CTL control plane processes
	for _, ctlAppLabel := range OsmCtlLabels {
		pods, err := Td.GetPodsForLabel(Td.OsmNamespace, metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.AppLabel: ctlAppLabel,
			},
		})
		if err != nil {
			td.T.Logf("Failed to get pods for applabel %s : %v", ctlAppLabel, err)
			continue
		}

		for _, pod := range pods {
			for _, contStatus := range pod.Status.ContainerStatuses {
				td.T.Logf("Restarts for %s/%s : %d", pod.Name, contStatus.Name, contStatus.RestartCount)
				restartMap[fmt.Sprintf("%s/%s", pod.Name, contStatus.Name)] = int(contStatus.RestartCount)
			}
		}
	}

	return restartMap
}

// GetBugReport runs the "osm support bug-report" command
func (td *OsmTestData) GetBugReport() error {
	absTestDirPath, err := filepath.Abs(td.GetTestDirPath())
	if err != nil {
		return err
	}

	args := []string{"support", "bug-report", "--all", fmt.Sprintf("-o=%s/osm_bug_report.tar.gz", absTestDirPath)}

	stdout, stderr, err := td.RunLocal(filepath.FromSlash("../../bin/osm"), args...)

	td.T.Logf("stdout:\n%s", stdout)

	if err != nil {
		td.T.Logf("error running osm support bug-report")
		td.T.Logf("stdout:\n%s", stdout)
		td.T.Logf("stderr:\n%s", stderr)
		return errors.Wrap(err, "failed to run osm support bug-report")
	}

	return nil
}

// GrabLogs Collects logs on test folder for td.OsmNamespace
func (td *OsmTestData) GrabLogs() error {
	logCollector := "../../scripts/get-osm-namespace-logs.sh"

	// Using absolute paths (though inferred from relative) for clarity
	absLogCollectorPath, err := filepath.Abs(logCollector)
	if err != nil {
		return err
	}

	absTestDirPath, err := filepath.Abs(td.GetTestDirPath())
	if err != nil {
		return err
	}

	td.T.Logf("Collecting logs, using \"%s %s\"", absLogCollectorPath, absTestDirPath)
	// Assumes testing has been launched from repo's root
	stdout, stderr, err := td.RunLocal(absLogCollectorPath, absTestDirPath)
	if err != nil {
		td.T.Logf("error running get-osm-namespace-logs script")
		td.T.Logf("stdout:\n%s", stdout)
		td.T.Logf("stderr:\n%s", stderr)
	}

	stdout, stderr, err = td.RunLocal("kubectl", "get", "events", "-A")
	if err != nil {
		td.T.Logf("error running kubectl get events")
		td.T.Logf("stdout:\n%s", stdout)
		td.T.Logf("stderr:\n%s", stderr)
	} else {
		if err := ioutil.WriteFile(fmt.Sprintf("%s/%s", absTestDirPath, "events"), stdout.Bytes(), 0600); err != nil {
			td.T.Logf("Failed to write file for events: %s", err)
		}
	}

	if td.CollectLogs == ControlPlaneOnly {
		return nil
	}

	if td.InstType == KindCluster {
		kindExportPath := td.GetTestFilePath("kindExport")
		td.T.Logf("Collecting kind cluster")

		stdout, stderr, err := td.RunLocal("kind", "export", "logs", "--name", td.ClusterName, kindExportPath)
		if err != nil {
			td.T.Logf("error running get-osm-namespace-logs script")
			td.T.Logf("stdout:\n%s", stdout)
			td.T.Logf("stderr:\n%s", stderr)
		}
	}

	// TODO: Eventually a CLI command should implement collection of configurations necessary for debugging
	pods, err := td.Client.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		// Reliable way to select injected pods
		LabelSelector: "osm-proxy-uuid",
	})

	if err != nil {
		return err
	}

	envoyConfDir := td.GetTestFilePath("envoy_configs")
	err = os.Mkdir(envoyConfDir, 0750)
	if err != nil && !os.IsExist(err) {
		td.T.Logf("Error on creating dir for %s: %v", envoyConfDir, err)
		return err
	}

	for _, pod := range pods.Items {
		podEnvoyConfigFilepath := strings.Join([]string{envoyConfDir, fmt.Sprintf("%s_%s", pod.Namespace, pod.Name)}, "/")
		err := os.Mkdir(podEnvoyConfigFilepath, 0750)
		if err != nil && !os.IsExist(err) {
			td.T.Logf("Error on creating dir for %s: %v, skipping", podEnvoyConfigFilepath, err)
			continue
		}

		envoyDebugPaths := []string{
			"config_dump",
			"clusters",
			"certs",
			"listeners",
			"ready",
			"stats",
		}

		for _, dbgEnvoyPath := range envoyDebugPaths {
			cmd := "../../bin/osm"
			filePath := fmt.Sprintf("%s/%s.txt", podEnvoyConfigFilepath, dbgEnvoyPath)
			args := []string{
				"proxy",
				"get",
				dbgEnvoyPath,
				pod.Name,
				"--namespace",
				pod.Namespace,
				"-f",
				filePath,
			}

			stdout, stderr, err := td.RunLocal(cmd, args...)
			if err != nil {
				td.T.Logf("error running cmd: %s args: %v", cmd, args)
				td.T.Logf("stdout:\n%s", stdout)
				td.T.Logf("stderr:\n%s", stderr)
			}
		}
	}

	return nil
}

// addOpenShiftSCC adds the specified SecurityContextConstraint to the given service account
func (td *OsmTestData) addOpenShiftSCC(scc, serviceAccount, namespace string) error {
	if !td.DeployOnOpenShift {
		return errors.Errorf("Tests are not configured for OpenShift. Try again with -deployOnOpenShift=true")
	}

	roleName := serviceAccount + "-scc"
	roleDefinition := td.simpleRole(roleName, namespace)
	policyRule := rbacv1.PolicyRule{
		APIGroups:     []string{"security.openshift.io"},
		ResourceNames: []string{scc},
		Resources:     []string{"securitycontextconstraints"},
		Verbs:         []string{"use"},
	}
	roleDefinition.Rules = []rbacv1.PolicyRule{policyRule}

	_, err := td.createRole(namespace, &roleDefinition)
	if err != nil {
		return errors.Errorf("Failed to create Role %s: %s", roleName, err)
	}

	roleBindingName := serviceAccount + "-scc"
	roleBindingDefinition := td.simpleRoleBinding(roleBindingName, namespace)
	subject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      serviceAccount,
		Namespace: namespace,
	}
	roleBindingDefinition.Subjects = []rbacv1.Subject{subject}
	roleRef := rbacv1.RoleRef{
		Kind:     "Role",
		Name:     roleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	roleBindingDefinition.RoleRef = roleRef

	_, err = td.createRoleBinding(namespace, &roleBindingDefinition)
	if err != nil {
		return errors.Errorf("Failed to create RoleBinding %s: %s", roleBindingName, err)
	}

	return nil
}
