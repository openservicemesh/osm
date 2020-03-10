package injector

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/deislabs/smc/pkg/catalog"
	"github.com/deislabs/smc/pkg/certificate"
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/log/level"
)

const (
	tlsDir      = `/run/secrets/tls`
	tlsCertFile = `tls.crt`
	tlsKeyFile  = `tls.key`
)

var (
	codecs       = serializer.NewCodecFactory(runtime.NewScheme())
	deserializer = codecs.UniversalDeserializer()
)

// NewWebhook returns a new Webhook object
func NewWebhook(config Config, kubeConfig *rest.Config, certManager certificate.Manager, meshCatalog catalog.MeshCataloger, namespaces []string) *Webhook {
	return &Webhook{
		config:      config,
		kubeClient:  kubernetes.NewForConfigOrDie(kubeConfig),
		certManager: certManager,
		meshCatalog: meshCatalog,
		namespaces:  namespaces,
	}
}

// ListenAndServe starts the mutating webhook
func (wh *Webhook) ListenAndServe(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.DefaultServeMux
	// HTTP handlers
	mux.HandleFunc("/health/ready", wh.healthReadyHandler)
	mux.HandleFunc("/mutate", wh.mutateHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", wh.config.ListenPort),
		Handler: mux,
	}

	glog.Infof("Starting sidecar-injection webhook server on :%v", wh.config.ListenPort)
	go func() {
		if wh.config.EnableTLS {
			certPath := filepath.Join(tlsDir, tlsCertFile)
			keyPath := filepath.Join(tlsDir, tlsKeyFile)
			if err := server.ListenAndServeTLS(certPath, keyPath); err != nil {
				glog.Fatalf("Sidecar-injection webhook HTTP server failed to start: %v", err)
			}
		} else {
			if err := server.ListenAndServe(); err != nil {
				glog.Fatalf("Sidecar-injection webhook HTTP server failed to start: %v", err)
			}
		}
	}()

	// Wait on exit signals
	<-stop

	// Stop the server
	if err := server.Shutdown(ctx); err != nil {
		glog.Errorf("Error shutting down sidecar-injection webhook HTTP server: %v", err)
	} else {
		glog.Info("Done shutting down sidecar-injection webhook HTTP server")
	}
}

func (wh *Webhook) healthReadyHandler(w http.ResponseWriter, req *http.Request) {
	// TODO(shashank): If TLS certificate is not present, mark as not ready
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Health OK"))
}

func (wh *Webhook) mutateHandler(w http.ResponseWriter, req *http.Request) {
	glog.Infof("Request received: Method=%v, URL=%v", req.Method, req.URL)

	if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
		errmsg := fmt.Sprintf("Invalid Content-Type: %q", contentType)
		http.Error(w, errmsg, http.StatusUnsupportedMediaType)
		glog.Errorf("Request error: error=%s, code=%v", errmsg, http.StatusUnsupportedMediaType)
		return
	}

	var body []byte
	if req.Body != nil {
		var err error
		if body, err = ioutil.ReadAll(req.Body); err != nil {
			errmsg := fmt.Sprintf("Error reading request body: %s", err)
			http.Error(w, errmsg, http.StatusInternalServerError)
			glog.Errorf("Request error: error=%s, code=%v", errmsg, http.StatusInternalServerError)
			return
		}
	}

	if len(body) == 0 {
		errmsg := "Empty request body"
		http.Error(w, errmsg, http.StatusBadRequest)
		glog.Errorf("Request error: error=%s, code=%v", errmsg, http.StatusBadRequest)
		return
	}

	var admissionReq v1beta1.AdmissionReview
	var admissionResp v1beta1.AdmissionReview
	if _, _, err := deserializer.Decode(body, nil, &admissionReq); err != nil {
		glog.Errorf("Error decoding admission request: %s", err)
		admissionResp.Response = toAdmissionError(err)
	} else {
		admissionResp.Response = wh.mutate(admissionReq.Request)
	}

	resp, err := json.Marshal(&admissionResp)
	if err != nil {
		errmsg := fmt.Sprintf("Error marshalling admission response: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		glog.Errorf("Request error, error=%s, code=%v", errmsg, http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(resp); err != nil {
		glog.Errorf("Error writing response: %s", err)
	}

	glog.V(level.Debug).Info("Done responding to admission request")
}

func (wh *Webhook) mutate(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	// Decode the Pod spec from the request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		glog.Errorf("Error unmarshaling request to Pod: %s", err)
		return toAdmissionError(err)
	}
	glog.Infof("Mutation request:\nobject: %v\nold object: %v", string(req.Object.Raw), string(req.OldObject.Raw))

	// Start building the response
	resp := &v1beta1.AdmissionResponse{
		Allowed: true,
		UID:     req.UID,
	}

	// Check if we must inject the sidecar
	if inject, err := wh.mustInject(&pod, req.Namespace); err != nil {
		glog.Errorf("Error checking if sidecar must be injected: %s", err)
		return toAdmissionError(err)
	} else if !inject {
		glog.Info("Skipping sidecar injection")
		return resp
	}

	// Create the patches for the spec
	patchBytes, err := wh.createPatch(&pod, req.Namespace)
	if err != nil {
		glog.Infof("Failed to create patch: %s", err)
		return toAdmissionError(err)
	}

	patchAdmissionResponse(resp, patchBytes)
	glog.Info("Done patching admission response")
	return resp
}

func (wh *Webhook) isNamespaceAllowed(namespace string) bool {
	for _, ns := range wh.namespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}

func (wh *Webhook) mustInject(pod *corev1.Pod, namespace string) (bool, error) {
	// If the request belongs to a namespace we are not observing, skip it
	if !wh.isNamespaceAllowed(namespace) {
		glog.Infof("Request belongs to namespace=%s, not in the list of observing namespaces: %v", namespace, wh.namespaces)
		return false, nil
	}
	// TODO(shashank): Check system namespace
	namespacedServiceAcc := endpoint.NamespacedServiceAccount{
		Namespace:      namespace,
		ServiceAccount: pod.Spec.ServiceAccountName,
	}

	// Check to see if the service account is referenced in SMI
	services := wh.meshCatalog.GetServicesByServiceAccountName(namespacedServiceAcc, true)
	if len(services) == 0 {
		// No services found for this service account, don't patch
		glog.Infof("No services found for service account %q", pod.Spec.ServiceAccountName)
		return false, nil
	}

	return true, nil
}

func toAdmissionError(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func patchAdmissionResponse(resp *v1beta1.AdmissionResponse, patchBytes []byte) {
	resp.Patch = patchBytes
	resp.PatchType = func() *v1beta1.PatchType {
		pt := v1beta1.PatchTypeJSONPatch
		return &pt
	}()
}
