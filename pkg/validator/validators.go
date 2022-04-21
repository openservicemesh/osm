package validator

import (
	"bytes"
	"encoding/json"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	admissionv1 "k8s.io/api/admission/v1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
)

// validateFunc is a function type that accepts an AdmissionRequest and returns an AdmissionResponse.
/*
There are a few ways to utilize the Validator function:

1. return resp, nil

	In this case we simply return the raw resp. This allows for the most customization.

2. return nil, err

	In this case we convert the error to an AdmissionResponse.  If the error type is an AdmissionError, we
	convert accordingly, which allows for some customization of the AdmissionResponse. Otherwise, we set Allow to
	false and the status to the error message.

3. return nil, nil

	In this case we create a simple AdmissionResponse, with Allow set to true.

4. Note that resp, err will ignore the error. It assumes that you are returning nil for resp if there is an error

In all of the above cases we always populate the UID of the response from the request.

An example of a validator:

func FakeValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	o, n := &FakeObj{}, &FakeObj{}
	// If you need to compare against the old object
	if err := json.NewDecoder(bytes.NewBuffer(req.OldObject.Raw)).Decode(o); err != nil {
		return nil, err
	}

	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(n); err != nil {
		returrn nil, err
	}

	// validate the objects, potentially returning an error, or a more detailed AdmissionResponse.

	// This will set allow to true
	return nil, nil
}
*/
type validateFunc func(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error)

func trafficTargetValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	trafficTarget := &smiAccess.TrafficTarget{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(trafficTarget); err != nil {
		return nil, err
	}

	if trafficTarget.Spec.Destination.Namespace != trafficTarget.Namespace {
		return nil, errors.Errorf("The traffic target namespace (%s) must match spec.Destination.Namespace (%s)",
			trafficTarget.Namespace, trafficTarget.Spec.Destination.Namespace)
	}

	return nil, nil
}

// ingressBackendValidator validates the IngressBackend custom resource
func ingressBackendValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	ingressBackend := &policyv1alpha1.IngressBackend{}
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
				if source.Kind == policyv1alpha1.KindAuthenticatedPrincipal {
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

	// Validate sources
	for _, source := range ingressBackend.Spec.Sources {
		switch source.Kind {
		// Add validation for source kinds here
		case policyv1alpha1.KindService:
			if source.Name == "" {
				return nil, errors.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindService)
			}
			if source.Namespace == "" {
				return nil, errors.Errorf("'source.namespace' not specified for source kind %s", policyv1alpha1.KindService)
			}

		case policyv1alpha1.KindAuthenticatedPrincipal:
			if source.Name == "" {
				return nil, errors.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindAuthenticatedPrincipal)
			}

		case policyv1alpha1.KindIPRange:
			if _, _, err := net.ParseCIDR(source.Name); err != nil {
				return nil, errors.Errorf("Invalid 'source.name' value specified for IPRange. Expected CIDR notation 'a.b.c.d/x', got '%s'", source.Name)
			}

		default:
			return nil, errors.Errorf("Invalid 'source.kind' value specified. Must be one of: %s, %s, %s",
				policyv1alpha1.KindService, policyv1alpha1.KindAuthenticatedPrincipal, policyv1alpha1.KindIPRange)
		}
	}

	return nil, nil
}

// egressValidator validates the Egress custom resource
func egressValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	egress := &policyv1alpha1.Egress{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(egress); err != nil {
		return nil, err
	}

	// Validate match references
	allowedAPIGroups := []string{smiSpecs.SchemeGroupVersion.String(), policyv1alpha1.SchemeGroupVersion.String()}
	upstreamTrafficSettingMatchCount := 0
	for _, m := range egress.Spec.Matches {
		switch *m.APIGroup {
		case smiSpecs.SchemeGroupVersion.String():
			switch m.Kind {
			case "HTTPRouteGroup":
				// no additional validation

			default:
				return nil, errors.Errorf("Expected 'matches.kind' for match '%s' to be 'HTTPRouteGroup', got: %s", m.Name, m.Kind)
			}

		case policyv1alpha1.SchemeGroupVersion.String():
			switch m.Kind {
			case "UpstreamTrafficSetting":
				upstreamTrafficSettingMatchCount++

			default:
				return nil, errors.Errorf("Expected 'matches.kind' for match '%s' to be 'UpstreamTrafficSetting', got: %s", m.Name, m.Kind)
			}

		default:
			return nil, errors.Errorf("Expected 'matches.apiGroup' to be one of %v, got: %s", allowedAPIGroups, *m.APIGroup)
		}
	}

	// Can't have more than 1 UpstreamTrafficSetting match for an Egress policy
	if upstreamTrafficSettingMatchCount > 1 {
		return nil, errors.New("Cannot have more than 1 UpstreamTrafficSetting match")
	}

	return nil, nil
}

// MultiClusterServiceValidator validates the MultiClusterService CRD.
func MultiClusterServiceValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	config := &configv1alpha2.MultiClusterService{}
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
