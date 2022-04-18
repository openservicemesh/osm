package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/policy"
	"github.com/openservicemesh/osm/pkg/service"
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

// policyValidator is a validator that has access to a policy
type policyValidator struct {
	policyClient policy.Controller
}

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
func (kc *policyValidator) ingressBackendValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	ingressBackend := &policyv1alpha1.IngressBackend{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(ingressBackend); err != nil {
		return nil, err
	}
	ns := ingressBackend.Namespace

	type setEntry struct {
		name string
		port int
	}

	backends := mapset.NewSet()
	var conflictString strings.Builder
	conflictingIngressBackends := mapset.NewSet()
	for _, backend := range ingressBackend.Spec.Backends {
		if unique := backends.Add(setEntry{backend.Name, backend.Port.Number}); !unique {
			return nil, errors.Errorf("Duplicate backends detected with service name: %s and port: %d", backend.Name, backend.Port.Number)
		}

		fakeMeshSvc := service.MeshService{
			Name:       backend.Name,
			TargetPort: uint16(backend.Port.Number),
			Protocol:   backend.Port.Protocol,
		}

		if matchingPolicy := kc.policyClient.GetIngressBackendPolicy(fakeMeshSvc); matchingPolicy != nil && matchingPolicy.Name != ingressBackend.Name {
			// we've found a duplicate
			if unique := conflictingIngressBackends.Add(matchingPolicy); !unique {
				// we've already found the conflicts for this resource
				continue
			}
			conflicts := policy.DetectIngressBackendConflicts(*ingressBackend, *matchingPolicy)
			fmt.Fprintf(&conflictString, "[+] IngressBackend %s/%s conflicts with %s/%s:\n", ns, ingressBackend.ObjectMeta.GetName(), ns, matchingPolicy.ObjectMeta.GetName())
			for _, err := range conflicts {
				fmt.Fprintf(&conflictString, "%s\n", err)
			}
			fmt.Fprintf(&conflictString, "\n")
		}

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

	if conflictString.Len() != 0 {
		return nil, fmt.Errorf("duplicate backends detected\n%s", conflictString.String())
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

// upstreamTrafficSettingValidator validates the UpstreamTrafficSetting custom resource
func (kc *policyValidator) upstreamTrafficSettingValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	upstreamTrafficSetting := &policyv1alpha1.UpstreamTrafficSetting{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(upstreamTrafficSetting); err != nil {
		return nil, err
	}

	ns := upstreamTrafficSetting.Namespace
	hostComponents := strings.Split(upstreamTrafficSetting.Spec.Host, ".")
	if len(hostComponents) < 2 {
		return nil, field.Invalid(field.NewPath("spec").Child("host"), upstreamTrafficSetting.Spec.Host, "invalid FQDN specified as host")
	}

	opt := policy.UpstreamTrafficSettingGetOpt{Host: upstreamTrafficSetting.Spec.Host}
	if matchingUpstreamTrafficSetting := kc.policyClient.GetUpstreamTrafficSetting(opt); matchingUpstreamTrafficSetting != nil && matchingUpstreamTrafficSetting.Name != upstreamTrafficSetting.Name {
		// duplicate detected
		return nil, errors.Errorf("UpstreamTrafficSetting %s/%s conflicts with %s/%s since they have the same host %s", ns, upstreamTrafficSetting.ObjectMeta.GetName(), ns, matchingUpstreamTrafficSetting.ObjectMeta.GetName(), matchingUpstreamTrafficSetting.Spec.Host)
	}

	return nil, nil
}

// MultiClusterServiceValidator validates the MultiClusterService CRD.
func MultiClusterServiceValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	config := &configv1alpha3.MultiClusterService{}
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
