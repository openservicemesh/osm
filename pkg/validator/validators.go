package validator

import (
	"bytes"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cv1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	pv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
)

// minCertValidityDuration is the minimum duration for a MeshConfig's certificate validity.
const minCertValidityDuration = 2 * time.Minute

func init() {
	ingressBackendGvk := metav1.GroupVersionKind{
		Kind:    "IngressBackend",
		Group:   "policy.openservicemesh.io",
		Version: "v1alpha1",
	}
	egressGvk := metav1.GroupVersionKind{
		Kind:    "Egress",
		Group:   "policy.openservicemesh.io",
		Version: "v1alpha1",
	}
	meshConfigGvk := metav1.GroupVersionKind{
		Kind:    "MeshConfig",
		Group:   "config.openservicemesh.io",
		Version: "v1alpha1",
	}

	RegisterValidator(ingressBackendGvk.String(), IngressBackendValidator)
	RegisterValidator(egressGvk.String(), EgressValidator)
	RegisterValidator(meshConfigGvk.String(), MeshConfigValidator)
}

// IngressBackendValidator validates the IngressBackend custom resource
func IngressBackendValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	ingressBackend := &pv1alpha1.IngressBackend{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(ingressBackend); err != nil {
		return nil, err
	}

	for _, backend := range ingressBackend.Spec.Backends {
		// Validate port
		switch strings.ToLower(backend.Port.Protocol) {
		case constants.ProtocolHTTP:
			// Valid

		case constants.ProtocolHTTPS:
			// Valid
			// If mTLS is enabled, verify there is an AuthenticatedPrincipal specified
			authenticatedSourceFound := false
			for _, source := range ingressBackend.Spec.Sources {
				if source.Kind == pv1alpha1.KindAuthenticatedPrincipal {
					authenticatedSourceFound = true
					break
				}
			}

			if backend.TLS.SkipClientCertValidation && !authenticatedSourceFound {
				return nil, errors.Errorf("HTTPS ingress with client certificate validation enabled must specify at least one 'AuthenticatedPrincipal` source")
			}

		default:
			return nil, errors.Errorf("Expected 'port.protocol' to be 'http' or 'https', got: %s", backend.Port.Protocol)
		}
	}

	return nil, nil
}

// EgressValidator validates the Egress custom resource
func EgressValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	egress := &pv1alpha1.Egress{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(egress); err != nil {
		return nil, err
	}

	for _, m := range egress.Spec.Matches {
		if m.Kind != "HTTPRouteGroup" {
			return nil, errors.Errorf("Expected 'Matches.Kind' to be 'HTTPRouteGroup', got: %s", m.Kind)
		}

		if *m.APIGroup != "specs.smi-spec.io/v1alpha4" {
			return nil, errors.Errorf("Expected 'Matches.APIGroup' to be 'specs.smi-spec.io/v1alpha4', got: %s", *m.APIGroup)
		}
	}

	return nil, nil
}

// MeshConfigValidator validates the MeshConfig custom resource
func MeshConfigValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	config := &cv1alpha1.MeshConfig{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(config); err != nil {
		return nil, err
	}

	d, err := time.ParseDuration(config.Spec.Certificate.ServiceCertValidityDuration)
	if err != nil {
		return nil, errors.Errorf("'Certificate.ServiceCertValidityDuration' %s is not valid", config.Spec.Certificate.ServiceCertValidityDuration)
	}

	if d < minCertValidityDuration {
		return nil, errors.Errorf("'Certificate.ServiceCertValidityDuration' %d is lower than %d", d, minCertValidityDuration)
	}

	return nil, nil
}

// MultiClusterServiceValidator validates the MultiClusterService CRD.
func MultiClusterServiceValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	config := &cv1alpha1.MultiClusterService{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(config); err != nil {
		return nil, err
	}

	clusterNames := make(map[string]bool)

	for _, cluster := range config.Spec.Clusters {
		if len(strings.TrimSpace(cluster.Name)) == 0 {
			return nil, errors.New("Cluster name is not valid")
		}
		if _, ok := clusterNames[cluster.Name]; ok {
			return nil, errors.Errorf("Cluster named %s already exists", cluster.Name)
		}
		if len(strings.TrimSpace(cluster.Address)) == 0 {
			return nil, errors.Errorf("Cluster address %s is not valid", cluster.Address)
		}
		clusterAddress := strings.Split(cluster.Address, ":")
		if net.ParseIP(clusterAddress[0]) == nil {
			return nil, errors.Errorf("Error parsing IP address %s", cluster.Address)
		}
		_, err := strconv.ParseUint(clusterAddress[1], 10, 32)
		if err != nil {
			return nil, errors.Errorf("Error parsing port value %s", cluster.Address)
		}
		clusterNames[cluster.Name] = true
	}

	return nil, nil
}
