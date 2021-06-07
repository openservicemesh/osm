package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cv1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	pv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

// MinDuration is the minimum duration for a MeshConfig's certificate validity.
const MinDuration = time.Second

func init() {
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
	RegisterValidator(egressGvk.String(), EgressValidator)
	RegisterValidator(meshConfigGvk.String(), MeshConfigValidator)
}

// EgressValidator validates the Egress CRD.
func EgressValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	egress := &pv1alpha1.Egress{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(egress); err != nil {
		return nil, err
	}

	for _, m := range egress.Spec.Matches {
		if m.Kind != "HTTPRouteGroup" {
			return nil, fmt.Errorf("Expected Matches.Kind to be 'HTTPRouteGroup', got: %s", m.Kind)
		}

		if *m.APIGroup != "specs.smi-spec.io/v1alpha4" {
			return nil, fmt.Errorf("Expected Matches.APIGroup to be 'specs.smi-spec.io/v1alpha4', got: %s", *m.APIGroup)
		}
	}

	return nil, nil
}

// MeshConfigValidator validates the MeshConfig CRD.
func MeshConfigValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	config := &cv1alpha1.MeshConfig{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(config); err != nil {
		return nil, err
	}

	d, err := time.ParseDuration(config.Spec.Certificate.ServiceCertValidityDuration)
	if err != nil {
		return nil, fmt.Errorf("Certificate.ServiceCertValidityDuration %s is not valid", config.Spec.Certificate.ServiceCertValidityDuration)
	}

	if d < MinDuration {
		return nil, fmt.Errorf("Certificate.ServiceCertValidityDuration %d is lower than %d", d, MinDuration)
	}

	return nil, nil
}
