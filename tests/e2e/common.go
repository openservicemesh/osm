package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/openservicemesh/osm/pkg/utils"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

const (
	// contant, default name for the Registry Secret
	registrySecretName = "acr-creds"
	// constant, default name for the mesh
	defaultMeshName = "osm-system"
	// default image tag
	defaultImageTag = "latest"
)

// OsmTestData stores common state, variables and flags for the test at hand
type OsmTestData struct {
	T GinkgoTInterface // for common test logging

	cleanupTest    bool // Cleanup test-related resources once finished
	waitForCleanup bool // Forces test to wait for effective deletion of resources upon cleanup

	// OSM install-time variables
	osmNamespace string
	osmImageTag  string

	// Container registry related vars
	ctrRegistryUser     string // registry login
	ctrRegistryPassword string // registry password, if any
	ctrRegistryServer   string // server name. Has to be network reachable

	// Kind cluster related vars
	kindCluster                    bool   // Create and use a kind cluster
	clusterName                    string // Kind cluster name (used if kindCluster)
	cleanupKindClusterBetweenTests bool   // Clean and re-create kind cluster between tests
	cleanupKindCluster             bool   // Cleanup kind cluster upon test finish

	// Cluster handles and rest config
	restConfig      *rest.Config
	client          *kubernetes.Clientset
	smiClients      *smiClients
	clusterProvider *cluster.Provider // provider, used when kindCluster is used

	// Tracks namespaces to cleanup when test finishes (if cleanup is enabled)
	cleanupNamespaces map[string]bool
}

// Function to run at init before Ginkgo has called parseFlags
// See suite_test.go for details on how Ginko calls parseFlags
func registerFlags(td *OsmTestData) {
	flag.BoolVar(&td.cleanupTest, "cleanupTest", true, "Cleanup test resources when done")
	flag.BoolVar(&td.waitForCleanup, "waitForCleanup", true, "Wait for effective deletion of resources")

	flag.BoolVar(&td.kindCluster, "kindCluster", false, "Creates kind cluster")
	flag.StringVar(&td.clusterName, "kindClusterName", "osm-e2e", "Name of the Kind cluster to be created")
	flag.BoolVar(&td.cleanupKindCluster, "cleanupKindCluster", true, "Cleanup kind cluster upon exit")
	flag.BoolVar(&td.cleanupKindClusterBetweenTests, "cleanupKindClusterBetweenTests", false, "Cleanup kind cluster between tests")

	flag.StringVar(&td.ctrRegistryServer, "ctrRegistry", os.Getenv("CTR_REGISTRY"), "Container registry")
	flag.StringVar(&td.ctrRegistryUser, "ctrRegistryUser", os.Getenv("CTR_REGISTRY_USER"), "Container registry")
	flag.StringVar(&td.ctrRegistryPassword, "ctrRegistrySecret", os.Getenv("CTR_REGISTRY_PASSWORD"), "Container registry secret")

	flag.StringVar(&td.osmImageTag, "osmImageTag", utils.GetEnv("CTR_TAG", defaultImageTag), "OSM image tag")
	flag.StringVar(&td.osmNamespace, "osmNamespace", utils.GetEnv("K8S_NAMESPACE", defaultMeshName), "OSM mesh name")
}

// AreRegistryCredsPresent checks if Registry Credentials are present
// It's usually used to factor if a docker registry secret and ImagePullSecret
// should be installed when creating namespaces and application templates
func (td *OsmTestData) AreRegistryCredsPresent() bool {
	return len(td.ctrRegistryUser) > 0 && len(td.ctrRegistryPassword) > 0
}

// InitTestData Initializes the test structures
// Called by Gingkgo BeforeEach
func (td *OsmTestData) InitTestData(t GinkgoTInterface) error {
	td.T = t
	td.cleanupNamespaces = make(map[string]bool)

	if len(td.ctrRegistryServer) == 0 {
		td.T.Errorf("Did not read any container registry (is CTR_REGISTRY set?)")
	}

	if td.kindCluster && td.clusterProvider == nil {
		td.clusterProvider = cluster.NewProvider()
		td.T.Logf("Creating local kind cluster")
		if err := td.clusterProvider.Create(td.clusterName); err != nil {
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

	td.restConfig = kubeConfig
	td.client = clientset

	if err := td.InitSMIClients(); err != nil {
		return errors.Wrap(err, "failed to initialize SMI clients")
	}

	// After client creations, do a wait for kind cluster just in case it's not done yet coming up
	// Ballparking pod number. kind has a large number of containers to run by default
	if td.kindCluster && td.clusterProvider != nil {
		if err := td.WaitForPodsRunningReady("kube-system", 120*time.Second, 5); err != nil {
			return errors.Wrap(err, "failed to wait for kube-system pods")
		}
	}

	return nil
}

// InstallOSMOpts describes install options for OSM
type InstallOSMOpts struct {
	controlPlaneNS          string
	certManager             string
	containerRegistryLoc    string
	containerRegistrySecret string
	osmImagetag             string
	deployGrafana           bool
	deployPrometheus        bool
	deployJaeger            bool

	vaultHost     string
	vaultProtocol string
	vaultToken    string
	vaultRole     string

	egressEnabled bool
}

// GetOSMInstallOpts initializes install options for OSM
func (td *OsmTestData) GetOSMInstallOpts() InstallOSMOpts {
	return InstallOSMOpts{
		controlPlaneNS:          td.osmNamespace,
		certManager:             "tresor",
		containerRegistryLoc:    td.ctrRegistryServer,
		containerRegistrySecret: td.ctrRegistryPassword,
		osmImagetag:             td.osmImageTag,
		deployGrafana:           false,
		deployPrometheus:        false,
		deployJaeger:            false,

		vaultHost:     "vault." + td.osmNamespace + ".svc.cluster.local",
		vaultProtocol: "http",
		vaultRole:     "openservicemesh",
		vaultToken:    "token",
	}
}

// HelmInstallOSM installs an osm control plane using the osm chart which lives in charts/osm
func (td *OsmTestData) HelmInstallOSM() error {
	if td.kindCluster {
		if err := td.loadOSMImagesIntoKind(); err != nil {
			return err
		}
	}

	values := fmt.Sprintf("OpenServiceMesh.image.registry=%s,OpenServiceMesh.image.tag=%s", td.ctrRegistryServer, td.osmImageTag)
	namespace := "helm-install-namespace"
	release := "helm-install-osm"
	args := []string{"install", release, "../../charts/osm", "--set", values, "--namespace", namespace, "--create-namespace", "--wait"}
	stdout, stderr, err := td.RunLocal("helm", args)
	if err != nil {
		td.T.Logf("stdout:\n%s", stdout)
		return errors.Errorf("failed to run helm install with osm chart: %s", stderr)
	}
	if err = td.deleteHelmRelease(release, namespace); err != nil {
		td.T.Fatal(err)
	}

	return nil
}

func (td *OsmTestData) deleteHelmRelease(name, namespace string) error {
	args := []string{"uninstall", name, "--namespace", namespace}
	stdout, stderr, err := td.RunLocal("helm", args)
	if err != nil {
		td.T.Logf("stdout:\n%s", stdout)
		return errors.Errorf("Could not delete helm release: %s", stderr)
	}
	return nil
}

// InstallOSM installs OSM. Right now relies on externally calling the binary and a subset of possible opts
// TODO: refactor install to be able to call it directly here vs. exec-ing CLI.
func (td *OsmTestData) InstallOSM(instOpts InstallOSMOpts) error {
	if td.kindCluster {
		if err := td.loadOSMImagesIntoKind(); err != nil {
			return err
		}
	}

	if err := td.CreateNs(instOpts.controlPlaneNS, nil); err != nil {
		return errors.Wrap(err, "failed to create namespace "+instOpts.controlPlaneNS)
	}

	if instOpts.certManager == "vault" {
		if err := td.installVault(instOpts); err != nil {
			return err
		}
	}

	td.T.Log("Installing OSM")
	var args []string

	// Add OSM namespace to cleanup namespaces, in case the test can't init
	td.cleanupNamespaces[instOpts.controlPlaneNS] = true

	args = append(args, "install",
		"--container-registry="+instOpts.containerRegistryLoc,
		"--osm-image-tag="+instOpts.osmImagetag,
		"--namespace="+instOpts.controlPlaneNS,
		"--certificate-manager="+instOpts.certManager,
		"--enable-egress="+strconv.FormatBool(instOpts.egressEnabled),
		"--enable-debug-server",
	)

	switch instOpts.certManager {
	case "vault":
		args = append(args,
			"--vault-host="+instOpts.vaultHost,
			"--vault-token="+instOpts.vaultToken,
			"--vault-protocol="+instOpts.vaultProtocol,
			"--vault-role="+instOpts.vaultRole,
		)
	}

	if !td.kindCluster {
		// Making sure the image is always pulled in registry-based testing
		args = append(args, "--osm-image-pull-policy=Always")
	}

	if len(instOpts.containerRegistrySecret) != 0 {
		args = append(args, "--container-registry-secret="+registrySecretName)
	}

	args = append(args, fmt.Sprintf("--enable-prometheus=%v", instOpts.deployPrometheus))
	args = append(args, fmt.Sprintf("--enable-grafana=%v", instOpts.deployGrafana))
	args = append(args, fmt.Sprintf("--deploy-jaeger=%v", instOpts.deployJaeger))

	stdout, stderr, err := td.RunLocal(filepath.FromSlash("../../bin/osm"), args)
	if err != nil {
		td.T.Logf("error running osm install")
		td.T.Logf("stdout:\n%s", stdout)
		td.T.Logf("stderr:\n%s", stderr)
		return errors.Wrap(err, "failed to run osm install")
	}

	return nil
}

func (td *OsmTestData) loadOSMImagesIntoKind() error {
	td.T.Log("Getting image data")
	imageNames := []string{
		"osm-controller",
		"init",
	}
	docker, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return errors.Wrap(err, "failed to create docker client")
	}
	var imageIDs []string
	for _, name := range imageNames {
		imageName := fmt.Sprintf("%s/%s:%s", td.ctrRegistryServer, name, td.osmImageTag)
		imageIDs = append(imageIDs, imageName)
	}
	imageData, err := docker.ImageSave(context.TODO(), imageIDs)
	if err != nil {
		return errors.Wrap(err, "failed to get image data")
	}
	defer imageData.Close()
	nodes, err := td.clusterProvider.ListNodes(td.clusterName)
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
`, instOpts.vaultToken, instOpts.vaultToken, instOpts.vaultRole),
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
	_, err := td.client.AppsV1().Deployments(instOpts.controlPlaneNS).Create(context.TODO(), vaultDep, metav1.CreateOptions{})
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
	_, err = td.client.CoreV1().Services(instOpts.controlPlaneNS).Create(context.TODO(), vaultSvc, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create vault service")
	}
	return nil
}

// AddNsToMesh Adds monitored namespaces to the OSM mesh
func (td *OsmTestData) AddNsToMesh(sidecardInject bool, ns ...string) error {
	td.T.Logf("Adding Namespaces [+%s] to the mesh", ns)
	for _, namespace := range ns {
		args := []string{"namespace", "add", namespace}
		if sidecardInject {
			args = append(args, "--enable-sidecar-injection")
		}

		args = append(args, "--namespace="+td.osmNamespace)
		stdout, stderr, err := td.RunLocal(filepath.FromSlash("../../bin/osm"), args)
		if err != nil {
			td.T.Logf("error running osm namespace add")
			td.T.Logf("stdout:\n%s", stdout)
			td.T.Logf("stderr:\n%s", stderr)
			return errors.Wrap(err, "failed to run osm namespace add")
		}
	}
	return nil
}

func (td *OsmTestData) UpdateOSMConfig(key, value string) error {
	patch := []byte(fmt.Sprintf(`{"data": {%q: %q}}`, key, value))
	_, err := td.client.CoreV1().ConfigMaps(td.osmNamespace).Patch(context.TODO(), "osm-config", types.StrategicMergePatchType, patch, metav1.PatchOptions{})
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
		labels = map[string]string{}
	}

	// For cleanup purposes, we mark this as present at this time.
	// If the test can't run because there's the same namespace running, it's most
	// likely that the user will want it gone anyway
	td.cleanupNamespaces[nsName] = true

	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsName,
			Namespace: "",
			Labels:    labels,
		},
		Status: corev1.NamespaceStatus{},
	}

	td.T.Logf("Creating namespace %v", nsName)
	_, err := td.client.CoreV1().Namespaces().Create(context.Background(), namespaceObj, metav1.CreateOptions{})
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
	var backgroundDelete metav1.DeletionPropagation = metav1.DeletePropagationBackground

	td.T.Logf("Deleting namespace %v", nsName)
	err := td.client.CoreV1().Namespaces().Delete(context.Background(), nsName, metav1.DeleteOptions{PropagationPolicy: &backgroundDelete})
	delete(td.cleanupNamespaces, nsName)
	if err != nil {
		return errors.Wrap(err, "failed to delete namespace "+nsName)
	}
	return nil
}

// WaitForNamespacesDeleted waits for the namespaces to be deleted.
// Reference impl taken from https://github.com/kubernetes/kubernetes/blob/master/test/e2e/framework/util.go#L258
func (td *OsmTestData) WaitForNamespacesDeleted(namespaces []string, timeout time.Duration) error {
	By(fmt.Sprintf("Waiting for namespaces %+v to vanish", namespaces))
	nsMap := map[string]bool{}
	for _, ns := range namespaces {
		nsMap[ns] = true
	}
	//Now POLL until all namespaces have been eradicated.
	return wait.Poll(2*time.Second, timeout,
		func() (bool, error) {
			nsList, err := td.client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
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
	command string) (string, string, error) {
	var stdin, stdout, stderr bytes.Buffer

	req := td.client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(ns).SubResource("exec")

	option := &corev1.PodExecOptions{
		Command:   strings.Fields(command),
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
	exec, err := remotecommand.NewSPDYExecutor(td.restConfig, "POST", req.URL())
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
	td.T.Logf("Wait for pods ready in ns [%s]...", ns)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(2 * time.Second) {
		pods, err := td.client.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
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

// Cleanup is Used to cleanup resorces once the test is done
func (td *OsmTestData) Cleanup(ct CleanupType) {
	// In-cluster Test resources cleanup(namespace, crds, specs and whatnot) here
	if td.cleanupTest {
		var nsList []string
		for ns := range td.cleanupNamespaces {
			err := td.DeleteNs(ns)
			if err != nil {
				td.T.Logf("(warn) delete ns %s err: %v", ns, err)
			}
			nsList = append(nsList, ns)
		}

		if len(nsList) > 0 && td.waitForCleanup {
			// on kind this can take a while apparently
			err := td.WaitForNamespacesDeleted(nsList, 240*time.Second)
			if err != nil {
				td.T.Logf("Could not confirm all namespace deletion in time: %v", err)
			}
		}
		td.cleanupNamespaces = map[string]bool{}
	}

	// Kind cluster deletion, if needed
	if td.kindCluster && td.clusterProvider != nil {
		if ct == Test && td.cleanupKindClusterBetweenTests || ct == Suite && td.cleanupKindCluster {
			td.T.Logf("Deleting kind cluster: %s", td.clusterName)
			if err := td.clusterProvider.Delete(td.clusterName, clientcmd.RecommendedHomeFile); err != nil {
				td.T.Logf("error deleting cluster: %v", err)
			}
			td.clusterProvider = nil
		}
	}
}

//DockerConfig and other configs are docker-specific container registry secret structures.
// Most of it is taken or referenced from kubectl source itself
type DockerConfig map[string]DockerConfigEntry
type DockerConfigJSON struct {
	Auths       DockerConfig      `json:"auths"`
	HTTPHeaders map[string]string `json:"HttpHeaders,omitempty"`
}
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
		Username: td.ctrRegistryUser,
		Password: td.ctrRegistryPassword,
		Email:    "osm@osm.com",
		Auth:     base64.StdEncoding.EncodeToString([]byte(td.ctrRegistryUser + ":" + td.ctrRegistryPassword)),
	}

	dockerCfgJSON := DockerConfigJSON{
		Auths: map[string]DockerConfigEntry{td.ctrRegistryServer: dockercfgAuth},
	}

	json, _ := json.Marshal(dockerCfgJSON)
	secret.Data[corev1.DockerConfigJsonKey] = json

	td.T.Logf("Pushing Registry secret '%s' for namespace %s... ", registrySecretName, ns)
	_, err := td.client.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		td.T.Fatalf("Could not add registry secret")
	}
}
