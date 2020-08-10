package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/util/validation"
)

type actionConfig struct {
	containerRegistry             string
	containerRegistrySecret       string
	chartPath                     string
	osmImageTag                   string
	certManager                   string
	vaultHost                     string
	vaultProtocol                 string
	vaultToken                    string
	vaultRole                     string
	serviceCertValidityMinutes    int
	prometheusRetentionTime       string
	enableDebugServer             bool
	enablePermissiveTrafficPolicy bool
	enableEgress                  bool
	meshName                      string
	meshCIDRRanges                []string
	// This is an experimental flag, which will eventually
	// become part of SMI Spec.
	enableBackpressureExperimental bool
	// Toggle this to enable/disable the automatic deployment of Zipkin
	deployZipkin bool
	// Toggle to deploy/not deploy metrics (Promethus+Grafana) stack
	enableMetricsStack bool
}

func (a actionConfig) validate() error {
	meshNameErrs := validation.IsValidLabelValue(a.meshName)

	if len(meshNameErrs) != 0 {
		return errors.Errorf("Invalid mesh-name.\nValid mesh-name:\n- must be no longer than 63 characters\n- must consist of alphanumeric characters, '-', '_' or '.'\n- must start and end with an alphanumeric character\nregex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?'")
	}

	if strings.EqualFold(a.certManager, "vault") {
		var missingFields []string
		if a.vaultHost == "" {
			missingFields = append(missingFields, "vault-host")
		}
		if a.vaultToken == "" {
			missingFields = append(missingFields, "vault-token")
		}
		if len(missingFields) != 0 {
			return errors.Errorf("Missing arguments for certificate-manager vault: %v", missingFields)
		}
	}

	// Validate CIDR ranges if egress is enabled
	if a.enableEgress {
		if err := validateCIDRs(a.meshCIDRRanges); err != nil {
			return errors.Errorf("Invalid mesh-cidr-ranges: %q, error: %v. Valid mesh CIDR ranges must be specified with egress enabled.", a.meshCIDRRanges, err)
		}
	}

	return nil
}

func (a actionConfig) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}
	valuesConfig := []string{
		fmt.Sprintf("OpenServiceMesh.image.registry=%s", a.containerRegistry),
		fmt.Sprintf("OpenServiceMesh.image.tag=%s", a.osmImageTag),
		fmt.Sprintf("OpenServiceMesh.imagePullSecrets[0].name=%s", a.containerRegistrySecret),
		fmt.Sprintf("OpenServiceMesh.certManager=%s", a.certManager),
		fmt.Sprintf("OpenServiceMesh.vault.host=%s", a.vaultHost),
		fmt.Sprintf("OpenServiceMesh.vault.protocol=%s", a.vaultProtocol),
		fmt.Sprintf("OpenServiceMesh.vault.token=%s", a.vaultToken),
		fmt.Sprintf("OpenServiceMesh.vault.role=%s", a.vaultRole),
		fmt.Sprintf("OpenServiceMesh.serviceCertValidityMinutes=%d", a.serviceCertValidityMinutes),
		fmt.Sprintf("OpenServiceMesh.prometheus.retention.time=%s", a.prometheusRetentionTime),
		fmt.Sprintf("OpenServiceMesh.enableDebugServer=%t", a.enableDebugServer),
		fmt.Sprintf("OpenServiceMesh.enablePermissiveTrafficPolicy=%t", a.enablePermissiveTrafficPolicy),
		fmt.Sprintf("OpenServiceMesh.enableBackpressureExperimental=%t", a.enableBackpressureExperimental),
		fmt.Sprintf("OpenServiceMesh.enableMetricsStack=%t", a.enableMetricsStack),
		fmt.Sprintf("OpenServiceMesh.meshName=%s", a.meshName),
		fmt.Sprintf("OpenServiceMesh.enableEgress=%t", a.enableEgress),
		fmt.Sprintf("OpenServiceMesh.meshCIDRRanges=%s", strings.Join(a.meshCIDRRanges, " ")),
		fmt.Sprintf("OpenServiceMesh.deployZipkin=%t", a.deployZipkin),
	}

	for _, val := range valuesConfig {
		if err := strvals.ParseInto(val, finalValues); err != nil {
			return finalValues, err
		}
	}
	return finalValues, nil
}

func validateCIDRs(cidrRanges []string) error {
	if len(cidrRanges) == 0 {
		return errors.Errorf("CIDR ranges cannot be empty when `enable-egress` option is true`")
	}
	for _, cidr := range cidrRanges {
		cidrNoSpaces := strings.Replace(cidr, " ", "", -1)
		_, _, err := net.ParseCIDR(cidrNoSpaces)
		if err != nil {
			return errors.Errorf("Error parsing CIDR %s", cidr)
		}
	}
	return nil
}
