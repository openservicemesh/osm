package framework

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/docker/docker/client"
	"github.com/fatih/color"
	"github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certman "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

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

const (
	// default name for the container registry secret
	registrySecretName = "acr-creds"

	// test tag prefix, for NS labeling
	osmTest = "osmTest"
)

var (
	// default name for the mesh
	defaultOsmNamespace = "osm-system"
	// default image tag
	defaultImageTag = "latest"
	// default cert manager
	defaultCertManager = "tresor"
	// default enable NS metrics tag
	defaultEnableNsMetricTag = true
	// default enable debug server
	defaultEnableDebugServer = true
	// default deploy Prometheus
	defaultDeployPrometheus = false
	// default deploy Grafana
	defaultDeployGrafana = false
	// default deploy Jaeger
	defaultDeployJaeger = false
	// default deploy Fluentbit
	defaultDeployFluentbit = false
	// default envoy loglevel
	defaultEnvoyLogLevel = "debug"
)

// OSMDescribeInfo is a struct to represent the Tier and Bucket of a given e2e test
type OSMDescribeInfo struct {
	// Tier represents the priority of the test. Lower value indicates higher priority.
	Tier int

	// Bucket indicates in which test Bucket the test will run in for CI. Each
	// Bucket is run in parallel while tests in the same Bucket run sequentially.
	Bucket int
}

func (o OSMDescribeInfo) String() string {
	return fmt.Sprintf("[Tier %d][Bucket %d]", o.Tier, o.Bucket)
}

// OSMDescribe givens the description of an e2e test
func OSMDescribe(name string, opts OSMDescribeInfo, body func()) bool {
	return Describe(opts.String()+" "+name, body)
}

// InstallType defines several OSM test deployment scenarios
type InstallType string

const (
	// SelfInstall uses current kube cluster, installs OSM using CLI
	SelfInstall InstallType = "SelfInstall"
	// KindCluster Creates Kind cluster on docker and uses it as cluster, OSM installs through CLI
	KindCluster InstallType = "KindCluster"
	// NoInstall uses current kube cluster, assumes an OSM is present in `OsmNamespace`
	NoInstall InstallType = "NoInstall"
)

// Verifies the instType string flag option is a valid enum type
func verifyValidInstallType(t InstallType) error {
	switch t {
	case SelfInstall, KindCluster, NoInstall:
		return nil
	default:
		return errors.Errorf("%s is not a valid OSM install type", string(t))
	}
}

// OsmTestData stores common state, variables and flags for the test at hand
type OsmTestData struct {
	T              GinkgoTInterface // for common test logging
	TestID         uint64           // uint randomized for every test. GinkgoRandomSeed can't be used as is per-suite.
	TestFolderName string           // Test folder name, when overridden by test flags

	CleanupTest    bool // Cleanup test-related resources once finished
	WaitForCleanup bool // Forces test to wait for effective deletion of resources upon cleanup

	// OSM install-time variables
	InstType          InstallType // Install type.
	OsmNamespace      string
	OsmImageTag       string
	EnableNsMetricTag bool

	// Container registry related vars
	CtrRegistryUser     string // registry login
	CtrRegistryPassword string // registry password, if any
	CtrRegistryServer   string // server name. Has to be network reachable

	// Kind cluster related vars
	ClusterName                    string // Kind cluster name (used if kindCluster)
	CleanupKindClusterBetweenTests bool   // Clean and re-create kind cluster between tests
	CleanupKindCluster             bool   // Cleanup kind cluster upon test finish

	// Cluster handles and rest config
	Env             *cli.EnvSettings
	RestConfig      *rest.Config
	Client          *kubernetes.Clientset
	SmiClients      *smiClients
	ClusterProvider *cluster.Provider // provider, used when kindCluster is used
}

// Function to run at init before Ginkgo has called parseFlags
// See suite_test.go for details on how Ginko calls parseFlags
func registerFlags(td *OsmTestData) {
	flag.BoolVar(&td.CleanupTest, "cleanupTest", true, "Cleanup test resources when done")
	flag.BoolVar(&td.WaitForCleanup, "waitForCleanup", true, "Wait for effective deletion of resources")
	flag.StringVar(&td.TestFolderName, "testFolderName", "", "Test folder name")

	flag.StringVar((*string)(&td.InstType), "installType", string(SelfInstall), "Type of install/deployment for OSM")

	flag.StringVar(&td.ClusterName, "kindClusterName", "osm-e2e", "Name of the Kind cluster to be created")
	flag.BoolVar(&td.CleanupKindCluster, "cleanupKindCluster", true, "Cleanup kind cluster upon exit")
	flag.BoolVar(&td.CleanupKindClusterBetweenTests, "cleanupKindClusterBetweenTests", false, "Cleanup kind cluster between tests")

	flag.StringVar(&td.CtrRegistryServer, "ctrRegistry", os.Getenv("CTR_REGISTRY"), "Container registry")
	flag.StringVar(&td.CtrRegistryUser, "ctrRegistryUser", os.Getenv("CTR_REGISTRY_USER"), "Container registry")
	flag.StringVar(&td.CtrRegistryPassword, "ctrRegistrySecret", os.Getenv("CTR_REGISTRY_PASSWORD"), "Container registry secret")

	flag.StringVar(&td.OsmImageTag, "osmImageTag", utils.GetEnv("CTR_TAG", defaultImageTag), "OSM image tag")
	flag.StringVar(&td.OsmNamespace, "OsmNamespace", utils.GetEnv("K8S_NAMESPACE", defaultOsmNamespace), "OSM Namespace")

	flag.BoolVar(&td.EnableNsMetricTag, "EnableMetricsTag", defaultEnableNsMetricTag, "Enable tagging Namespaces for metrics collection")
}

// GetTestFile prefixes a filename with current test folder
// and creates the test folder for current test if it doesn't exists.
// Only if some part of a test calls this function the test folder will be created,
// otherwise nothing is created to avoid extra clutter.
func (td *OsmTestData) GetTestFile(filename string) string {
	var testDir string
	if len(td.TestFolderName) == 0 {
		testDir = fmt.Sprintf("test-%d", td.TestID)
	} else {
		testDir = td.TestFolderName
	}

	err := os.Mkdir(testDir, 0750)

	exists := false
	if err == nil {
		td.T.Logf("Created test dir %s", testDir)
		exists = true
	}

	if os.IsExist(err) || exists {
		return fmt.Sprintf("./%s/%s", testDir, filename)
	}

	return ""
}

// GetTestNamespaceSelectorMap returns a string-based selector used to refer/select all namespace
// resources for this test
func (td *OsmTestData) GetTestNamespaceSelectorMap() map[string]string {
	return map[string]string{
		osmTest: fmt.Sprintf("%d", GinkgoRandomSeed()),
	}
}

// AreRegistryCredsPresent checks if Registry Credentials are present
// It's usually used to factor if a docker registry secret and ImagePullSecret
// should be installed when creating namespaces and application templates
func (td *OsmTestData) AreRegistryCredsPresent() bool {
	return len(td.CtrRegistryUser) > 0 && len(td.CtrRegistryPassword) > 0
}

// InitTestData Initializes the test structures
// Called by Gingkgo BeforeEach
func (td *OsmTestData) InitTestData(t GinkgoTInterface) error {
	td.T = t
	r, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
	if err != nil {
		return err
	}
	td.TestID = r.Uint64()
	td.T.Log(color.HiGreenString("ID for test: %d", td.TestID))

	err = verifyValidInstallType(td.InstType)
	if err != nil {
		return err
	}

	if (td.InstType == KindCluster) && td.ClusterProvider == nil {
		td.ClusterProvider = cluster.NewProvider()
		td.T.Logf("Creating local kind cluster")
		if err := td.ClusterProvider.Create(td.ClusterName); err != nil {
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

	td.RestConfig = kubeConfig
	td.Client = clientset
	td.Env = cli.New()

	if err := td.InitSMIClients(); err != nil {
		return errors.Wrap(err, "failed to initialize SMI clients")
	}

	// After client creations, do a wait for kind cluster just in case it's not done yet coming up
	// Ballparking pod number. kind has a large number of containers to run by default
	if (td.InstType == KindCluster) && td.ClusterProvider != nil {
		if err := td.WaitForPodsRunningReady("kube-system", 120*time.Second, 5); err != nil {
			return errors.Wrap(err, "failed to wait for kube-system pods")
		}
	}

	return nil
}

// InstallOSMOpts describes install options for OSM
type InstallOSMOpts struct {
	ControlPlaneNS          string
	CertManager             string
	ContainerRegistryLoc    string
	ContainerRegistrySecret string
	OsmImagetag             string
	DeployGrafana           bool
	DeployPrometheus        bool
	DeployJaeger            bool
	DeployFluentbit         bool

	VaultHost     string
	VaultProtocol string
	VaultToken    string
	VaultRole     string

	CertmanagerIssuerGroup string
	CertmanagerIssuerKind  string
	CertmanagerIssuerName  string

	EgressEnabled        bool
	EnablePermissiveMode bool
	EnvoyLogLevel        string
	EnableDebugServer    bool

	SetOverrides []string
}

// GetOSMInstallOpts initializes install options for OSM
func (td *OsmTestData) GetOSMInstallOpts() InstallOSMOpts {
	return InstallOSMOpts{
		ControlPlaneNS:          td.OsmNamespace,
		CertManager:             defaultCertManager,
		ContainerRegistryLoc:    td.CtrRegistryServer,
		ContainerRegistrySecret: td.CtrRegistryPassword,
		OsmImagetag:             td.OsmImageTag,
		DeployGrafana:           defaultDeployGrafana,
		DeployPrometheus:        defaultDeployPrometheus,
		DeployJaeger:            defaultDeployJaeger,
		DeployFluentbit:         defaultDeployFluentbit,

		VaultHost:     "vault." + td.OsmNamespace + ".svc.cluster.local",
		VaultProtocol: "http",
		VaultRole:     "openservicemesh",
		VaultToken:    "token",

		CertmanagerIssuerGroup: "cert-manager.io",
		CertmanagerIssuerKind:  "Issuer",
		CertmanagerIssuerName:  "osm-ca",
		EnvoyLogLevel:          defaultEnvoyLogLevel,
		EnableDebugServer:      defaultEnableDebugServer,
		SetOverrides:           []string{},
	}
}

// HelmInstallOSM installs an osm control plane using the osm chart which lives in charts/osm
func (td *OsmTestData) HelmInstallOSM(release, namespace string) error {
	if td.InstType == KindCluster {
		if err := td.loadOSMImagesIntoKind(); err != nil {
			return err
		}
	}

	values := fmt.Sprintf("OpenServiceMesh.image.registry=%s,OpenServiceMesh.image.tag=%s,OpenServiceMesh.meshName=%s", td.CtrRegistryServer, td.OsmImageTag, release)
	args := []string{"install", release, "../../charts/osm", "--set", values, "--namespace", namespace, "--create-namespace", "--wait"}
	stdout, stderr, err := td.RunLocal("helm", args)
	if err != nil {
		td.T.Logf("stdout:\n%s", stdout)
		return errors.Errorf("failed to run helm install with osm chart: %s", stderr)
	}

	return nil
}

// DeleteHelmRelease uninstalls a particular helm release
func (td *OsmTestData) DeleteHelmRelease(name, namespace string) error {
	args := []string{"uninstall", name, "--namespace", namespace}
	_, _, err := td.RunLocal("helm", args)
	if err != nil {
		td.T.Fatal(err)
	}
	return nil
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
	defer imageData.Close() //nolint: errcheck,gosec
	nodes, err := td.ClusterProvider.ListNodes(td.ClusterName)
	if err != nil {
		return errors.Wrap(err, "failed to list kind nodes")
	}
	for _, n := range nodes {
		td.T.Log("Loading images onto node", n)
		if err := nodeutils.LoadImageArchive(n, imageData); err != nil {
			return errors.Wrap(err, "failed to load images")
		}
	}

	return nil
}

// InstallOSM installs OSM. The behavior of this function is dependant on
// installType and instOpts
func (td *OsmTestData) InstallOSM(instOpts InstallOSMOpts) error {
	if td.InstType == NoInstall {
		if instOpts.CertManager != defaultCertManager ||
			instOpts.DeployPrometheus != defaultDeployPrometheus ||
			instOpts.DeployGrafana != defaultDeployGrafana ||
			instOpts.DeployJaeger != defaultDeployJaeger ||
			instOpts.DeployFluentbit != defaultDeployFluentbit {
			Skip("Skipping test: NoInstall marked on a test that requires modified install")
		}

		// TODO: Check there is a valid OSM instance running already in OsmNamespace

		// This resets supported dynamic configs expected by the caller
		err := td.UpdateOSMConfig("egress",
			fmt.Sprintf("%t", instOpts.EgressEnabled))
		if err != nil {
			return err
		}
		err = td.UpdateOSMConfig("permissive_traffic_policy_mode",
			fmt.Sprintf("%t", instOpts.EnablePermissiveMode))
		if err != nil {
			return err
		}
		err = td.UpdateOSMConfig("enable_debug_server",
			fmt.Sprintf("%t", instOpts.EnableDebugServer))
		if err != nil {
			return err
		}
		return nil
	}

	if td.InstType == KindCluster {
		if err := td.loadOSMImagesIntoKind(); err != nil {
			return errors.Wrap(err, "failed to load OSM images to nodes for Kind cluster")
		}
	}

	if err := td.CreateNs(instOpts.ControlPlaneNS, nil); err != nil {
		return errors.Wrap(err, "failed to create namespace "+instOpts.ControlPlaneNS)
	}

	var args []string
	args = append(args, "install",
		"--container-registry="+instOpts.ContainerRegistryLoc,
		"--osm-image-tag="+instOpts.OsmImagetag,
		"--osm-namespace="+instOpts.ControlPlaneNS,
		"--certificate-manager="+instOpts.CertManager,
		"--enable-egress="+strconv.FormatBool(instOpts.EgressEnabled),
		"--enable-permissive-traffic-policy="+strconv.FormatBool(instOpts.EnablePermissiveMode),
		"--enable-debug-server="+strconv.FormatBool(instOpts.EnableDebugServer),
		"--envoy-log-level="+instOpts.EnvoyLogLevel,
	)

	switch instOpts.CertManager {
	case "vault":
		if err := td.installVault(instOpts); err != nil {
			return err
		}
		args = append(args,
			"--vault-host="+instOpts.VaultHost,
			"--vault-token="+instOpts.VaultToken,
			"--vault-protocol="+instOpts.VaultProtocol,
			"--vault-role="+instOpts.VaultRole,
		)
	case "cert-manager":
		if err := td.installCertManager(instOpts); err != nil {
			return err
		}
		args = append(args,
			"--cert-manager-issuer-name="+instOpts.CertmanagerIssuerName,
			"--cert-manager-issuer-kind="+instOpts.CertmanagerIssuerKind,
			"--cert-manager-issuer-group="+instOpts.CertmanagerIssuerGroup,
		)
	}

	if !(td.InstType == KindCluster) {
		// Making sure the image is always pulled in registry-based testing
		args = append(args, "--osm-image-pull-policy=Always")
	}

	if len(instOpts.ContainerRegistrySecret) != 0 {
		args = append(args, "--container-registry-secret="+registrySecretName)
	}

	args = append(args, fmt.Sprintf("--deploy-prometheus=%v", instOpts.DeployPrometheus))
	args = append(args, fmt.Sprintf("--deploy-grafana=%v", instOpts.DeployGrafana))
	args = append(args, fmt.Sprintf("--deploy-jaeger=%v", instOpts.DeployJaeger))
	args = append(args, fmt.Sprintf("--enable-fluentbit=%v", instOpts.DeployFluentbit))
	args = append(args, fmt.Sprintf("--timeout=%v", 90*time.Second))

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
	stdout, stderr, err := td.RunLocal(filepath.FromSlash("../../bin/osm"), args)
	if err != nil {
		td.T.Logf("error running osm install")
		td.T.Logf("stdout:\n%s", stdout)
		td.T.Logf("stderr:\n%s", stderr)
		return errors.Wrap(err, "failed to run osm install")
	}

	return nil
}

// RestartOSMController restarts the osm-controller pod in the installed controller's namespace
func (td *OsmTestData) RestartOSMController(instOpts InstallOSMOpts) error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	controllerPods, err := td.Client.CoreV1().Pods(instOpts.ControlPlaneNS).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error fetching controller pod")
	}
	if len(controllerPods.Items) != 1 {
		return errors.Errorf("expected 1 osm-controller pod, got %d", len(controllerPods.Items))
	}

	pod := controllerPods.Items[0]

	// Delete the pod and let k8s spin it up again
	err = td.Client.CoreV1().Pods(instOpts.ControlPlaneNS).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "erorr deleting osm-controller pod")
	}

	return nil
}

// GetConfigMap is a wrapper to get a config map by name in a particular namespace
func (td *OsmTestData) GetConfigMap(name, namespace string) (*corev1.ConfigMap, error) {
	configmap, err := td.Client.CoreV1().ConfigMaps(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configmap, nil
}

func (td *OsmTestData) loadOSMImagesIntoKind() error {
	imageNames := []string{
		"osm-controller",
		"init",
	}

	return td.LoadImagesToKind(imageNames)
}

func (td *OsmTestData) installVault(instOpts InstallOSMOpts) error {
	td.T.Log("Installing Vault")
	replicas := int32(1)
	terminationGracePeriodSeconds := int64(10)
	vaultDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vault",
			Labels: map[string]string{
				"app": "vault",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "vault",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "vault",
					},
				},
				Spec: corev1.PodSpec{
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
	_, err := td.Client.AppsV1().Deployments(instOpts.ControlPlaneNS).Create(context.TODO(), vaultDep, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create vault deployment")
	}

	vaultSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vault",
			Labels: map[string]string{
				"app": "vault",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app": "vault",
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
	install.Version = "v0.16.1"

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

	selfsigned := &v1alpha2.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "selfsigned",
		},
		Spec: v1alpha2.IssuerSpec{
			IssuerConfig: v1alpha2.IssuerConfig{
				SelfSigned: &v1alpha2.SelfSignedIssuer{},
			},
		},
	}

	cert := &v1alpha2.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-ca",
		},
		Spec: v1alpha2.CertificateSpec{
			IsCA:       true,
			Duration:   &metav1.Duration{Duration: 90 * 24 * time.Hour},
			SecretName: "osm-ca-bundle",
			CommonName: "osm-system",
			IssuerRef: cmmeta.ObjectReference{
				Name:  selfsigned.Name,
				Kind:  "Issuer",
				Group: "cert-manager.io",
			},
		},
	}

	ca := &v1alpha2.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "osm-ca",
		},
		Spec: v1alpha2.IssuerSpec{
			IssuerConfig: v1alpha2.IssuerConfig{
				CA: &v1alpha2.CAIssuer{
					SecretName: "osm-ca-bundle",
				},
			},
		},
	}

	if err := td.WaitForPodsRunningReady(install.Namespace, 60*time.Second, 3); err != nil {
		return errors.Wrap(err, "failed to wait for cert-manager pods ready")
	}

	cmClient, err := certman.NewForConfig(td.RestConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create cert-manager config")
	}

	_, err = cmClient.CertmanagerV1alpha2().Certificates(td.OsmNamespace).Create(context.TODO(), cert, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create Certificate "+cert.Name)
	}

	_, err = cmClient.CertmanagerV1alpha2().Issuers(td.OsmNamespace).Create(context.TODO(), selfsigned, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create Issuer "+selfsigned.Name)
	}

	_, err = cmClient.CertmanagerV1alpha2().Issuers(td.OsmNamespace).Create(context.TODO(), ca, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create Issuer "+ca.Name)
	}

	return nil
}

// AddNsToMesh Adds monitored namespaces to the OSM mesh
func (td *OsmTestData) AddNsToMesh(shouldInjectSidecar bool, ns ...string) error {
	td.T.Logf("Adding Namespaces [+%s] to the mesh", ns)
	for _, namespace := range ns {
		args := []string{"namespace", "add", namespace}
		if !shouldInjectSidecar {
			args = append(args, "--disable-sidecar-injection")
		}

		stdout, stderr, err := td.RunLocal(filepath.FromSlash("../../bin/osm"), args)
		if err != nil {
			td.T.Logf("error running osm namespace add")
			td.T.Logf("stdout:\n%s", stdout)
			td.T.Logf("stderr:\n%s", stderr)
			return errors.Wrap(err, "failed to run osm namespace add")
		}

		if Td.EnableNsMetricTag {
			args = []string{"metrics", "enable", "--namespace", namespace}
			stdout, stderr, err = td.RunLocal(filepath.FromSlash("../../bin/osm"), args)
			if err != nil {
				td.T.Logf("error running osm namespace add")
				td.T.Logf("stdout:\n%s", stdout)
				td.T.Logf("stderr:\n%s", stderr)
				return errors.Wrap(err, "failed to run osm namespace add")
			}
		}
	}
	return nil
}

// UpdateOSMConfig updates OSM configmap
func (td *OsmTestData) UpdateOSMConfig(key, value string) error {
	patch := []byte(fmt.Sprintf(`{"data": {%q: %q}}`, key, value))
	_, err := td.Client.CoreV1().ConfigMaps(td.OsmNamespace).Patch(context.TODO(), "osm-config", types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
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

// WaitForNamespacesDeleted waits for the namespaces to be deleted.
// Reference impl taken from https://github.com/kubernetes/kubernetes/blob/master/test/e2e/framework/util.go#L258
func (td *OsmTestData) WaitForNamespacesDeleted(namespaces []string, timeout time.Duration) error {
	By(fmt.Sprintf("Waiting for namespaces %v to vanish", namespaces))
	nsMap := map[string]bool{}
	for _, ns := range namespaces {
		nsMap[ns] = true
	}
	//Now POLL until all namespaces have been eradicated.
	return wait.Poll(2*time.Second, timeout,
		func() (bool, error) {
			nsList, err := td.Client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
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

// RunLocal Executes command on local
func (td *OsmTestData) RunLocal(path string, args []string) (*bytes.Buffer, *bytes.Buffer, error) {
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
	if err != nil {
		return "", "", err
	}

	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), nil
}

// WaitForPodsRunningReady waits for a <n> number of pods on an NS to be running and ready
func (td *OsmTestData) WaitForPodsRunningReady(ns string, timeout time.Duration, nExpectedRunningPods int) error {
	td.T.Logf("Wait up to %v for %d pods ready in ns [%s]...", timeout, nExpectedRunningPods, ns)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(2 * time.Second) {
		pods, err := td.Client.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
			FieldSelector: "status.phase=Running",
		})

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

	return fmt.Errorf("Not all pods were Running & Ready in NS %s after %v", ns, timeout)
}

// SuccessFunction is a simple definition for a success function.
// True as success, false otherwise
type SuccessFunction func() bool

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

// CleanupType identifies what triggered the cleanup
type CleanupType string

const (
	// Test is to mark after-test cleanup
	Test CleanupType = "test"
	//Suite is to mark after-suite cleanup
	Suite CleanupType = "suite"
)

// Cleanup is Used to cleanup resources once the test is done
func (td *OsmTestData) Cleanup(ct CleanupType) {
	if td.Client == nil {
		// Avoid any cleanup (crash) if no test is run;
		// init doesn't happen and clientsets are nil
		return
	}

	// The condition enters to cleanup K8s resources if
	// - cleanup is enabled and it's not a kind cluster
	// - cleanup is enabled and it is a kind cluster, but the kind cluster will NOT be
	//   destroyed after this test.
	//   The latter is a condition to speed up and not wait for k8s resources to vanish
	//   if the current kind cluster has to be destroyed anyway.
	if td.CleanupTest &&
		(!(td.InstType == KindCluster) ||
			(td.InstType == KindCluster &&
				(ct == Test && !td.CleanupKindClusterBetweenTests) ||
				(ct == Suite && !td.CleanupKindCluster))) {
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
				td.T.Logf("Poll err: %v", err)
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
}

//DockerConfig and other configs are docker-specific container registry secret structures.
// Most of it is taken or referenced from kubectl source itself
type DockerConfig map[string]DockerConfigEntry

// DockerConfigJSON  is a struct for docker-specific config
type DockerConfigJSON struct {
	Auths       DockerConfig      `json:"auths"`
	HTTPHeaders map[string]string `json:"HttpHeaders,omitempty"`
}

// DockerConfigEntry is a struct for docker-specific container registry secret structures
type DockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// CreateDockerRegistrySecret creates a secret named `registrySecretName` in namespace <ns>,
// based on ctrRegistry variables
func (td *OsmTestData) CreateDockerRegistrySecret(ns string) {
	secret := &corev1.Secret{}
	secret.Name = registrySecretName
	secret.Type = corev1.SecretTypeDockerConfigJson
	secret.Data = map[string][]byte{}

	dockercfgAuth := DockerConfigEntry{
		Username: td.CtrRegistryUser,
		Password: td.CtrRegistryPassword,
		Email:    "osm@osm.com",
		Auth:     base64.StdEncoding.EncodeToString([]byte(td.CtrRegistryUser + ":" + td.CtrRegistryPassword)),
	}

	dockerCfgJSON := DockerConfigJSON{
		Auths: map[string]DockerConfigEntry{td.CtrRegistryServer: dockercfgAuth},
	}

	json, _ := json.Marshal(dockerCfgJSON)
	secret.Data[corev1.DockerConfigJsonKey] = json

	td.T.Logf("Pushing Registry secret '%s' for namespace %s... ", registrySecretName, ns)
	_, err := td.Client.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		td.T.Fatalf("Could not add registry secret")
	}
}
