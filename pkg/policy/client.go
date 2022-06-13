package policy

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/k8s/informers"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// kindSvcAccount is the ServiceAccount kind
	kindSvcAccount = "ServiceAccount"
)

// NewPolicyController returns a policy.Controller interface related to functionality provided by the resources in the policy.openservicemesh.io API group
func NewPolicyController(informerCollection *informers.InformerCollection, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := &Client{
		informers:      informerCollection,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}

	egressEventTypes := k8s.EventTypes{
		Add:    announcements.EgressAdded,
		Update: announcements.EgressUpdated,
		Delete: announcements.EgressDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyEgress, k8s.GetEventHandlerFuncs(shouldObserve, egressEventTypes, msgBroker))
	ingressBackendEventTypes := k8s.EventTypes{
		Add:    announcements.IngressBackendAdded,
		Update: announcements.IngressBackendUpdated,
		Delete: announcements.IngressBackendDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyIngressBackend, k8s.GetEventHandlerFuncs(shouldObserve, ingressBackendEventTypes, msgBroker))

	retryEventTypes := k8s.EventTypes{
		Add:    announcements.RetryPolicyAdded,
		Update: announcements.RetryPolicyUpdated,
		Delete: announcements.RetryPolicyDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyRetry, k8s.GetEventHandlerFuncs(shouldObserve, retryEventTypes, msgBroker))

	upstreamTrafficSettingEventTypes := k8s.EventTypes{
		Add:    announcements.UpstreamTrafficSettingAdded,
		Update: announcements.UpstreamTrafficSettingUpdated,
		Delete: announcements.UpstreamTrafficSettingDeleted,
	}
	client.informers.AddEventHandler(informers.InformerKeyUpstreamTrafficSetting, k8s.GetEventHandlerFuncs(shouldObserve, upstreamTrafficSettingEventTypes, msgBroker))

	return client
}

// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity based on service accounts
func (c *Client) ListEgressPoliciesForSourceIdentity(source identity.K8sServiceAccount) []*policyV1alpha1.Egress {
	var policies []*policyV1alpha1.Egress

	for _, egressIface := range c.informers.List(informers.InformerKeyEgress) {
		egressPolicy := egressIface.(*policyV1alpha1.Egress)

		if !c.kubeController.IsMonitoredNamespace(egressPolicy.Namespace) {
			continue
		}

		for _, sourceSpec := range egressPolicy.Spec.Sources {
			if sourceSpec.Kind == kindSvcAccount && sourceSpec.Name == source.Name && sourceSpec.Namespace == source.Namespace {
				policies = append(policies, egressPolicy)
			}
		}
	}

	return policies
}

// GetIngressBackendPolicy returns the IngressBackend policy for the given backend MeshService
func (c *Client) GetIngressBackendPolicy(svc service.MeshService) *policyV1alpha1.IngressBackend {
	for _, ingressBackendIface := range c.informers.List(informers.InformerKeyIngressBackend) {
		ingressBackend := ingressBackendIface.(*policyV1alpha1.IngressBackend)

		if ingressBackend.Namespace != svc.Namespace {
			continue
		}

		// Return the first IngressBackend corresponding to the given MeshService.
		// Multiple IngressBackend policies for the same backend will be prevented
		// using a validating webhook.
		for _, backend := range ingressBackend.Spec.Backends {
			// we need to check ports to allow ingress to multiple ports on the same svc
			if backend.Name == svc.Name && backend.Port.Number == int(svc.TargetPort) {
				return ingressBackend
			}
		}
	}

	return nil
}

// ListRetryPolicies returns the retry policies for the given source identity based on service accounts.
func (c *Client) ListRetryPolicies(source identity.K8sServiceAccount) []*policyV1alpha1.Retry {
	var retries []*policyV1alpha1.Retry

	for _, retryInterface := range c.informers.List(informers.InformerKeyRetry) {
		retry := retryInterface.(*policyV1alpha1.Retry)
		if retry.Spec.Source.Kind == kindSvcAccount && retry.Spec.Source.Name == source.Name && retry.Spec.Source.Namespace == source.Namespace {
			retries = append(retries, retry)
		}
	}

	return retries
}

// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resource that matches the given options
func (c *Client) GetUpstreamTrafficSetting(options UpstreamTrafficSettingGetOpt) *policyV1alpha1.UpstreamTrafficSetting {
	if options.MeshService == nil && options.NamespacedName == nil && options.Host == "" {
		log.Error().Msgf("No option specified to get UpstreamTrafficSetting resource")
		return nil
	}

	if options.NamespacedName != nil {
		// Filter by namespaced name
		resource, exists, err := c.informers.GetByKey(informers.InformerKeyUpstreamTrafficSetting, options.NamespacedName.String())
		if exists && err == nil {
			return resource.(*policyV1alpha1.UpstreamTrafficSetting)
		}
		return nil
	}

	// Filter by MeshService
	for _, resource := range c.informers.List(informers.InformerKeyUpstreamTrafficSetting) {
		upstreamTrafficSetting := resource.(*policyV1alpha1.UpstreamTrafficSetting)

		if upstreamTrafficSetting.Spec.Host == options.Host {
			return upstreamTrafficSetting
		}

		if upstreamTrafficSetting.Namespace == options.MeshService.Namespace &&
			upstreamTrafficSetting.Spec.Host == options.MeshService.FQDN() {
			return upstreamTrafficSetting
		}
	}

	return nil
}
