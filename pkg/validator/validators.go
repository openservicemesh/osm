package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	xds_type "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/compute"

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

// validator is a validator that has access to a compute resources
type validator struct {
	computeClient compute.Interface
}

func trafficTargetValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	trafficTarget := &smiAccess.TrafficTarget{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(trafficTarget); err != nil {
		return nil, err
	}

	if trafficTarget.Spec.Destination.Namespace != trafficTarget.Namespace {
		return nil, fmt.Errorf("The traffic target namespace (%s) must match spec.Destination.Namespace (%s)",
			trafficTarget.Namespace, trafficTarget.Spec.Destination.Namespace)
	}

	return nil, nil
}

// ingressBackendValidator validates the IngressBackend custom resource
func (kc *validator) ingressBackendValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
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
			return nil, fmt.Errorf("Duplicate backends detected with service name: %s and port: %d", backend.Name, backend.Port.Number)
		}

		fakeMeshSvc := service.MeshService{
			Name:       backend.Name,
			TargetPort: uint16(backend.Port.Number),
			Protocol:   backend.Port.Protocol,
		}

		if matchingPolicy := kc.computeClient.GetIngressBackendPolicyForService(fakeMeshSvc); matchingPolicy != nil && matchingPolicy.Name != ingressBackend.Name {
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
				return nil, fmt.Errorf("HTTPS ingress with client certificate validation enabled must specify at least one 'AuthenticatedPrincipal` source")
			}

		default:
			return nil, fmt.Errorf("Expected 'port.protocol' to be 'http' or 'https', got: %s", backend.Port.Protocol)
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
				return nil, fmt.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindService)
			}
			if source.Namespace == "" {
				return nil, fmt.Errorf("'source.namespace' not specified for source kind %s", policyv1alpha1.KindService)
			}

		case policyv1alpha1.KindAuthenticatedPrincipal:
			if source.Name == "" {
				return nil, fmt.Errorf("'source.name' not specified for source kind %s", policyv1alpha1.KindAuthenticatedPrincipal)
			}

		case policyv1alpha1.KindIPRange:
			if _, _, err := net.ParseCIDR(source.Name); err != nil {
				return nil, fmt.Errorf("Invalid 'source.name' value specified for IPRange. Expected CIDR notation 'a.b.c.d/x', got '%s'", source.Name)
			}

		default:
			return nil, fmt.Errorf("Invalid 'source.kind' value specified. Must be one of: %s, %s, %s",
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
				return nil, fmt.Errorf("Expected 'matches.kind' for match '%s' to be 'HTTPRouteGroup', got: %s", m.Name, m.Kind)
			}

		case policyv1alpha1.SchemeGroupVersion.String():
			switch m.Kind {
			case "UpstreamTrafficSetting":
				upstreamTrafficSettingMatchCount++

			default:
				return nil, fmt.Errorf("Expected 'matches.kind' for match '%s' to be 'UpstreamTrafficSetting', got: %s", m.Name, m.Kind)
			}

		default:
			return nil, fmt.Errorf("Expected 'matches.apiGroup' to be one of %v, got: %s", allowedAPIGroups, *m.APIGroup)
		}
	}

	// Can't have more than 1 UpstreamTrafficSetting match for an Egress policy
	if upstreamTrafficSettingMatchCount > 1 {
		return nil, fmt.Errorf("Cannot have more than 1 UpstreamTrafficSetting match")
	}

	return nil, nil
}

// upstreamTrafficSettingValidator validates the UpstreamTrafficSetting custom resource
func (kc *validator) upstreamTrafficSettingValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	upstreamTrafficSetting := &policyv1alpha1.UpstreamTrafficSetting{}
	if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(upstreamTrafficSetting); err != nil {
		return nil, err
	}

	ns := upstreamTrafficSetting.Namespace
	hostComponents := strings.Split(upstreamTrafficSetting.Spec.Host, ".")
	if len(hostComponents) < 2 {
		return nil, field.Invalid(field.NewPath("spec").Child("host"), upstreamTrafficSetting.Spec.Host, "invalid FQDN specified as host")
	}

	if matchingUpstreamTrafficSetting := kc.computeClient.GetUpstreamTrafficSettingByHost(upstreamTrafficSetting.Spec.Host); matchingUpstreamTrafficSetting != nil && matchingUpstreamTrafficSetting.Name != upstreamTrafficSetting.Name {
		// duplicate detected
		return nil, fmt.Errorf("UpstreamTrafficSetting %s/%s conflicts with %s/%s since they have the same host %s", ns, upstreamTrafficSetting.ObjectMeta.GetName(), ns, matchingUpstreamTrafficSetting.ObjectMeta.GetName(), matchingUpstreamTrafficSetting.Spec.Host)
	}

	// Validate rate limiting config
	rl := upstreamTrafficSetting.Spec.RateLimit
	if rl != nil && rl.Local != nil && rl.Local.HTTP != nil {
		if _, ok := xds_type.StatusCode_name[int32(rl.Local.HTTP.ResponseStatusCode)]; !ok {
			return nil, fmt.Errorf("Invalid responseStatusCode %d. See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto#enum-type-v3-statuscode for allowed values",
				rl.Local.HTTP.ResponseStatusCode)
		}
	}
	for _, route := range upstreamTrafficSetting.Spec.HTTPRoutes {
		if route.RateLimit != nil && route.RateLimit.Local != nil {
			if _, ok := xds_type.StatusCode_name[int32(route.RateLimit.Local.ResponseStatusCode)]; !ok {
				return nil, fmt.Errorf("Invalid responseStatusCode %d. See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto#enum-type-v3-statuscode for allowed values",
					route.RateLimit.Local.ResponseStatusCode)
			}
		}
	}

	return nil, nil
}

func (kc *validator) meshRootCertificateValidator(req *admissionv1.AdmissionRequest) (*admissionv1.AdmissionResponse, error) {
	switch req.Operation {
	case admissionv1.Create:
		newMRC := &configv1alpha2.MeshRootCertificate{}
		if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(newMRC); err != nil {
			return nil, err
		}

		err := kc.validateMRCOnCreate(newMRC)
		return nil, err
	case admissionv1.Update:
		newMRC, oldMRC := &configv1alpha2.MeshRootCertificate{}, &configv1alpha2.MeshRootCertificate{}
		if err := json.NewDecoder(bytes.NewBuffer(req.Object.Raw)).Decode(newMRC); err != nil {
			return nil, err
		}
		if err := json.NewDecoder(bytes.NewBuffer(req.OldObject.Raw)).Decode(oldMRC); err != nil {
			return nil, err
		}

		err := kc.validateMRCOnUpdate(oldMRC, newMRC)
		return nil, err
	}
	return nil, nil
}

func (kc *validator) validateMRCOnCreate(mrc *configv1alpha2.MeshRootCertificate) error {
	if mrc.Spec.TrustDomain == "" {
		return fmt.Errorf("trustDomain must be non empty for MRC %s", getNamespacedMRC(mrc))
	}

	if err := validateMRCProvider(mrc); err != nil {
		return err
	}

	if mrc.Spec.Intent == v1alpha2.ActiveIntent {
		foundActive, err := kc.checkForExistingActiveMRC(mrc)
		if err != nil {
			return err
		}
		if foundActive {
			return fmt.Errorf("cannot create MRC %s with intent active. An MRC with active intent already exists in the control plane namespace", getNamespacedMRC(mrc))
		}
	}

	return nil
}

func (kc *validator) validateMRCOnUpdate(oldMRC *configv1alpha2.MeshRootCertificate, newMRC *configv1alpha2.MeshRootCertificate) error {
	if !reflect.DeepEqual(oldMRC.Spec.Provider, newMRC.Spec.Provider) {
		return fmt.Errorf("cannot update certificate provider settings for MRC %s. Create a new MRC and initiate root certificate rotation to update the provider", getNamespacedMRC(newMRC))
	}

	if oldMRC.Spec.TrustDomain != newMRC.Spec.TrustDomain {
		return fmt.Errorf("cannot update trust domain for MRC %s. Create a new MRC and initiate root certificate rotation to update the trust domain", getNamespacedMRC(oldMRC))
	}

	if oldMRC.Spec.SpiffeEnabled != newMRC.Spec.SpiffeEnabled {
		return fmt.Errorf("cannot update SpiffeEnabled for MRC %s. Create a new MRC and initiate root certificate rotation to enable SPIFFE certificates", getNamespacedMRC(oldMRC))
	}

	// TODO(#5205): add validation for simplified root certificate rotation process
	return nil
}

func validateMRCProvider(mrc *configv1alpha2.MeshRootCertificate) error {
	p := mrc.Spec.Provider
	switch {
	case p.Tresor != nil:
		secretRef := p.Tresor.CA.SecretRef
		if secretRef.Name == "" || secretRef.Namespace == "" {
			return fmt.Errorf("name and namespace in CA secret reference cannot be set to empty strings for MRC %s", getNamespacedMRC(mrc))
		}
	case p.Vault != nil:
		if p.Vault.Host == "" || p.Vault.Protocol == "" || p.Vault.Role == "" {
			return fmt.Errorf("host, protocol, and role cannot be set to empty strings for MRC %s", getNamespacedMRC(mrc))
		}

		tokenSecret := p.Vault.Token.SecretKeyRef
		if tokenSecret.Key == "" || tokenSecret.Name == "" || tokenSecret.Namespace == "" {
			return fmt.Errorf("key, name, and namespace for the Vault token secret reference cannot be set to empty strings for MRC %s", getNamespacedMRC(mrc))
		}
	case p.CertManager != nil:
		if p.CertManager.IssuerGroup == "" || p.CertManager.IssuerKind == "" || p.CertManager.IssuerName == "" {
			return fmt.Errorf("issuerGroup, issuerKind, and issuerName cannot be set to empty strings for MRC %s", getNamespacedMRC(mrc))
		}
	}

	return nil
}

// checkForExistingActiveMRC returns true if there are active MRCs in addition to the specified MRC
// Returns false if there are no active MRCs or if the specified mrc will be the only active MRC
func (kc *validator) checkForExistingActiveMRC(mrc *configv1alpha2.MeshRootCertificate) (bool, error) {
	mrcs, err := kc.computeClient.ListMeshRootCertificates()
	if err != nil {
		return false, err
	}

	for _, m := range mrcs {
		if m.Spec.Intent == v1alpha2.ActiveIntent && m.Name != mrc.Name {
			log.Error().Msgf("cannot create MRC %s with intent active. An MRC with active intent already exists in the control plane namespace", getNamespacedMRC(mrc))
			return true, nil
		}
	}

	return false, nil
}

func getNamespacedMRC(mrc *configv1alpha2.MeshRootCertificate) string {
	return fmt.Sprintf("%s/%s", mrc.Namespace, mrc.Name)
}
