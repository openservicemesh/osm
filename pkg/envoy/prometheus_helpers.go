package envoy

import (
	"time"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/configurator"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/service"
)

// GetPrometheusCertificate will try to grab the Prometheus service certificate and return it,
// creating it if it doesn't exist in the process
func GetPrometheusCertificate(certManager certificate.Manager, cfg configurator.Configurator) (certificate.Certificater, error) {

	ns := GetPrometheusNamespacedService(cfg)

	cert, err := certManager.GetCertificate(ns.GetCommonName())
	if err == nil {
		return cert, nil
	}

	duration := 87600 * time.Hour // FIXME, long-lived, use potential defines

	return certManager.IssueCertificate(ns.GetCommonName(), &duration)
}

// GetPrometheusNamespacedService returns a namespaced service object which identifies a
// prometheus instance
func GetPrometheusNamespacedService(cfg configurator.Configurator) service.NamespacedService {
	var promNs string
	var promSvc string

	// Prometheus namespace will default to OSM NS if none is defined
	if promNs = cfg.GetPrometheusNamespace(); promNs == "" {
		promNs = cfg.GetOSMNamespace()
	}

	// Prometheus service name will assume a default if none is provided
	if promSvc = cfg.GetPrometheusServiceName(); promSvc == "" {
		promSvc = constants.PrometheusDefaultName
	}

	return service.NamespacedService{
		Namespace: promNs,
		Service:   promSvc,
	}
}
