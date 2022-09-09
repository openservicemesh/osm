// Package smi implements the Service Mesh Interface (SMI) kubernetes client to observe and retrieve information
// regarding SMI traffic resources.
package smi

import (
	"encoding/json"
	"fmt"
	"net/http"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("smi-mesh-spec")
)

const (
	// ServiceAccountKind is the kind specified for the destination and sources in an SMI TrafficTarget policy
	ServiceAccountKind = "ServiceAccount"

	// TCPRouteKind is the kind specified for the TCP route rules in an SMI Traffictarget policy
	TCPRouteKind = "TCPRoute"

	// HTTPRouteGroupKind is the kind specified for the HTTP route rules in an SMI Traffictarget policy
	HTTPRouteGroupKind = "HTTPRouteGroup"
)

// TrafficTargetListOpt specifies the options used to filter TrafficTarget objects as a part of its lister
type TrafficTargetListOpt struct {
	Destination identity.K8sServiceAccount
}

// TrafficTargetListOption is a function type that implements filters on TrafficTarget lister
type TrafficTargetListOption func(o *TrafficTargetListOpt)

// WithTrafficTargetDestination applies a filter based on the destination service account to the TrafficTarget lister
func WithTrafficTargetDestination(d identity.K8sServiceAccount) TrafficTargetListOption {
	return func(o *TrafficTargetListOpt) {
		o.Destination = d
	}
}

// TrafficSplitListOpt specifies the options used to filter TrafficSplit objects as a part of its lister
type TrafficSplitListOpt struct {
	ApexService    service.MeshService
	BackendService service.MeshService
	KubeController k8s.Controller
}

// TrafficSplitListOption is a function type that implements filters on the TrafficSplit lister
type TrafficSplitListOption func(o *TrafficSplitListOpt)

// WithTrafficSplitApexService applies a filter based on the apex service to the TrafficSplit lister
func WithTrafficSplitApexService(s service.MeshService) TrafficSplitListOption {
	return func(o *TrafficSplitListOpt) {
		o.ApexService = s
	}
}

// WithTrafficSplitBackendService applies a filter based on the backend service to the TrafficSplit lister
func WithTrafficSplitBackendService(s service.MeshService) TrafficSplitListOption {
	return func(o *TrafficSplitListOpt) {
		o.BackendService = s
	}
}

// WithKubeController adds a KubeController to the TrafficSplit lister
func WithKubeController(c k8s.Controller) TrafficSplitListOption {
	return func(o *TrafficSplitListOpt) {
		o.KubeController = c
	}
}

// GetSmiClientVersionHTTPHandler returns an http handler that returns supported smi version information
func GetSmiClientVersionHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		versionInfo := map[string]string{
			"TrafficTarget":  smiAccess.SchemeGroupVersion.String(),
			"HTTPRouteGroup": smiSpecs.SchemeGroupVersion.String(),
			"TCPRoute":       smiSpecs.SchemeGroupVersion.String(),
			"TrafficSplit":   smiSplit.SchemeGroupVersion.String(),
		}

		if jsonVersionInfo, err := json.Marshal(versionInfo); err != nil {
			log.Error().Err(err).Msgf("Error marshaling version info struct: %+v", versionInfo)
		} else {
			_, _ = fmt.Fprint(w, string(jsonVersionInfo))
		}
	})
}
