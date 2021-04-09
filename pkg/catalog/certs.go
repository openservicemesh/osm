package catalog

import (
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/pkg/errors"
)

var errGotUnexpectedCertRequest = errors.New("errGotUnexpectedCertRequest")

func (mc *MeshCatalog) GetServiceAccountsForCert(certType envoy.SDSCertType, certName string, svcAccount service.K8sServiceAccount) ([]service.K8sServiceAccount, error) {
	// Program SAN matching based on SMI TrafficTarget policies
	switch certType {
	case envoy.RootCertTypeForMTLSOutbound:
		// For the outbound certificate validation context, the SANs needs to match the list of service identities
		// corresponding to the upstream service. This means, if the sdscert.Name points to service 'X',
		// the SANs for this certificate should correspond to the service identities of 'X'.
		meshSvc, err := service.UnmarshalMeshService(certName)
		if err != nil {
			log.Error().Err(err).Msgf("Error unmarshalling upstream service for outbound cert %s", certName)
			return nil, err
		}
		svcAccounts, err := mc.kubeController.ListServiceAccountsForService(*meshSvc) // was mc.ListServiceAccountsForService(*meshSvc)
		if err != nil {
			log.Error().Err(err).Msgf("Error listing service accounts for service %s", meshSvc)
			return nil, err
		}
		return svcAccounts, nil

	case envoy.RootCertTypeForMTLSInbound:
		// Verify that the SDS cert request corresponding to the mTLS root validation cert matches the identity
		// of this proxy. If it doesn't, then something is wrong in the system.
		svcAccountInRequest, err := service.UnmarshalK8sServiceAccount(certName)
		if err != nil {
			log.Error().Err(err).Msgf("Error unmarshalling service account for inbound mTLS validation cert %s", certName)
			return nil, err
		}

		if *svcAccountInRequest != svcAccount {
			log.Error().Err(errGotUnexpectedCertRequest).Msgf("Request for SDS cert %s does not belong to proxy with identity %s", certName, svcAccount)
			return nil, errGotUnexpectedCertRequest
		}

		// For the inbound certificate validation context, the SAN needs to match the list of all downstream
		// service identities that are allowed to connect to this upstream identity. This means, if the upstream proxy
		// identity is 'X', the SANs for this certificate should correspond to all the downstream identities
		// allowed to access 'X'.
		svcAccounts, err := mc.ListAllowedInboundServiceAccounts(svcAccount)
		if err != nil {
			log.Error().Err(err).Msgf("Error listing inbound service accounts for proxy with ServiceAccount %s", svcAccount)
			return nil, err
		}
		return svcAccounts, nil

	default:
		log.Debug().Msgf("SAN matching not needed for cert %s", certName)
		return nil, nil
	}
}
