package verifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
)

// PodStatusVerifier implements the Verifier interface for control plane health
type PodStatusVerifier struct {
	stdout     io.Writer
	stderr     io.Writer
	kubeClient kubernetes.Interface
	app        types.NamespacedName
}

// PodProbeVerifier implements the Verifier interface for control plane health probes
type PodProbeVerifier struct {
	stdout     io.Writer
	stderr     io.Writer
	kubeClient kubernetes.Interface
	app        types.NamespacedName
	prober     httpProber
}

// httpProber is the interface that wrapes the pod's HTTP Probe method
//
// Probe sends an HTTP(s) probe and returns an error if encountered
type httpProber interface {
	Probe(pod types.NamespacedName) error
}

type podProber struct {
	kubeClient kubernetes.Interface
	restConfig *rest.Config
	port       uint16
	path       string
	protocol   string
}

// NewPodStatusVerifier implements verification for control plane health
func NewPodStatusVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface, app types.NamespacedName) Verifier {
	return &PodStatusVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		app:        app,
	}
}

// Run executes the pod status verifier
func (v *PodStatusVerifier) Run() Result {
	result := Result{
		Context: fmt.Sprintf("Verify readiness status of pod for app %s", v.app),
	}

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: v.app.Name}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	podList, err := v.kubeClient.CoreV1().Pods(v.app.Namespace).List(context.TODO(), listOptions)
	if err != nil || podList == nil {
		result.Status = Unknown
		result.Reason = fmt.Sprintf("Error fetching pods for app %s, err: %s", v.app, err)
		return result
	}

	if len(podList.Items) == 0 {
		result.Status = Failure
		result.Reason = fmt.Sprintf("No pods found for app %s", v.app)
		return result
	}

	var failureReasons []string
	for _, pod := range podList.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s not scheduled", pod.Namespace, pod.Name))
				break
			}
			if cond.Type == corev1.ContainersReady && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s containers not ready", pod.Namespace, pod.Name))
				break
			}
			if cond.Type == corev1.PodInitialized && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s init-containers pending completion", pod.Namespace, pod.Name))
				break
			}
			if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
				failureReasons = append(failureReasons, fmt.Sprintf("%s/%s not ready", pod.Namespace, pod.Name))
			}
		}
	}
	if len(failureReasons) > 0 {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Pod not ready: %s", strings.Join(failureReasons, ", "))
		return result
	}

	result.Status = Success
	return result
}

// NewPodProbeVerifier implements verification for control plane health probes
func NewPodProbeVerifier(stdout io.Writer, stderr io.Writer, kubeClient kubernetes.Interface,
	app types.NamespacedName, httpProber httpProber) Verifier {
	return &PodProbeVerifier{
		stdout:     stdout,
		stderr:     stderr,
		kubeClient: kubeClient,
		app:        app,
		prober:     httpProber,
	}
}

// Run executes the verifier for pod health probe
func (v *PodProbeVerifier) Run() Result {
	result := Result{
		Context: fmt.Sprintf("Verify health probes of pod for app %s", v.app),
	}

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: v.app.Name}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	podList, err := v.kubeClient.CoreV1().Pods(v.app.Namespace).List(context.TODO(), listOptions)
	if err != nil || podList == nil {
		result.Status = Unknown
		result.Reason = fmt.Sprintf("Error fetching pods for app %s, err: %s", v.app, err)
		return result
	}

	if len(podList.Items) == 0 {
		result.Status = Failure
		result.Reason = fmt.Sprintf("No pods found for app %s", v.app)
		return result
	}

	var failureReasons []string
	for _, p := range podList.Items {
		pod := types.NamespacedName{Namespace: p.Namespace, Name: p.Name}
		if err := v.prober.Probe(pod); err != nil {
			failureReasons = append(failureReasons, fmt.Sprintf("HTTP probe failed for pod %s: %s", pod, err))
		}
	}

	if len(failureReasons) > 0 {
		result.Status = Failure
		result.Reason = fmt.Sprintf("Probe failure: %v", strings.Join(failureReasons, ", "))
	} else {
		result.Status = Success
	}

	return result
}

func (p *podProber) Probe(pod types.NamespacedName) error {
	if p.protocol != constants.ProtocolHTTP && p.protocol != constants.ProtocolHTTPS {
		return fmt.Errorf("unsupported probe protocol: %s", p.protocol)
	}

	dialer, err := k8s.DialerToPod(p.restConfig, p.kubeClient, pod.Name, pod.Namespace)
	if err != nil {
		return err
	}

	// TODO(#4634): try a different local port if the given port is unavailable
	localPort := p.port
	remotePort := p.port
	portForwarder, err := k8s.NewPortForwarder(dialer, fmt.Sprintf("%d:%d", localPort, remotePort))
	if err != nil {
		return err
	}

	err = portForwarder.Start(func(pf *k8s.PortForwarder) error {
		defer pf.Stop()

		url := fmt.Sprintf("%s://localhost:%d/%s", p.protocol, localPort, p.path)
		client := &http.Client{}

		if p.protocol == constants.ProtocolHTTPS {
			// Certificate validation is to be skipped for HTTPS probes
			// similar to how k8s api server handles HTTPS probes.
			// #nosec G402
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS12,
				},
			}
			client.Transport = transport
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		//nolint: errcheck
		//#nosec G307
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("unexpected response code: %d", resp.StatusCode)
		}
		return nil
	})

	return err
}
