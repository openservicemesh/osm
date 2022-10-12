package certificate

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/errcode"
)

func (m *Manager) handleMRCEvent(event MRCEvent) error {
	log.Debug().Msgf("handling MRC event for MRC %s", event.MRCName)
	mrcList, err := m.mrcClient.ListMeshRootCertificates()
	if err != nil {
		return err
	}

	filteredMRCList := filterOutInactiveMRCs(mrcList)

	desiredSigningMRC, desiredValidatingMRC, err := getSigningAndValidatingMRCs(filteredMRCList)
	if err != nil {
		return err
	}

	m.mu.Lock()
	signingIssuer := m.signingIssuer
	validatingIssuer := m.validatingIssuer
	m.mu.Unlock()

	// if the issuers are already set to the desired value, return
	if signingIssuer != nil && signingIssuer.ID == desiredSigningMRC.Name && validatingIssuer != nil && validatingIssuer.ID == desiredValidatingMRC.Name {
		return nil
	}

	desiredSigningIssuer, desiredValidatingIssuer, err := m.getCertIssuers(desiredSigningMRC, desiredValidatingMRC)
	if err != nil {
		return err
	}

	return m.updateIssuers(desiredSigningIssuer, desiredValidatingIssuer)
}

var validMRCIntentCombinations = map[v1alpha2.MeshRootCertificateIntent][]v1alpha2.MeshRootCertificateIntent{
	v1alpha2.ActiveIntent: {
		v1alpha2.PassiveIntent,
		v1alpha2.ActiveIntent,
	},
	v1alpha2.PassiveIntent: {
		v1alpha2.ActiveIntent,
	},
}

// ValidateMRCIntents validates the intent combination of MRCs
func ValidateMRCIntents(mrc1, mrc2 *v1alpha2.MeshRootCertificate) error {
	if mrc1 == nil || mrc2 == nil {
		log.Error().Err(ErrUnexpectedNilMRC).Msg("unexpected nil MRC provided when validating MRC intents")
		return ErrUnexpectedNilMRC
	}
	if mrc1 == mrc2 {
		if mrc1.Spec.Intent != v1alpha2.ActiveIntent {
			log.Error().Err(ErrExpectedActiveMRC).Msgf("expected single MRC with %s intent, found %s", v1alpha2.ActiveIntent, mrc1.Spec.Intent)
			return ErrExpectedActiveMRC
		}

		return nil
	}

	intent1 := mrc1.Spec.Intent
	intent2 := mrc2.Spec.Intent

	validIntents, ok := validMRCIntentCombinations[intent1]
	if !ok {
		log.Error().Err(ErrUnknownMRCIntent).Msgf("unable to find %s intent in set of valid intents. Invalid combination of %s intent and %s intent", intent1, intent1, intent2)
		return ErrUnknownMRCIntent
	}

	for _, intent := range validIntents {
		if intent2 == intent {
			log.Debug().Msgf("verified valid intent combination of %s intent and %s intent", intent1, intent2)
			return nil
		}
	}

	log.Error().Err(ErrInvalidMRCIntentCombination).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidMRCIntentCombination)).
		Msgf("invalid combination of %s intent and %s intent", intent1, intent2)
	return ErrInvalidMRCIntentCombination
}

// getSigningAndValidatingMRCs returns the signing and validating MRCs from a list of MRCs
func getSigningAndValidatingMRCs(mrcList []*v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, *v1alpha2.MeshRootCertificate, error) {
	if len(mrcList) == 0 {
		log.Error().Err(ErrNoMRCsFound).Msg("when handling MRC event, found no MRCs in OSM control plane namespace")
		return nil, nil, ErrNoMRCsFound
	}
	if len(mrcList) > 2 {
		log.Error().Err(ErrNumMRCExceedsMaxSupported).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrNumMRCExceedsMaxSupported)).
			Msgf("expected 2 or less MRCs in the OSM control plane namespace, found %d", len(mrcList))
		return nil, nil, ErrNumMRCExceedsMaxSupported
	}

	var mrc1, mrc2 *v1alpha2.MeshRootCertificate
	mrc1 = mrcList[0]
	if len(mrcList) == 2 {
		mrc2 = mrcList[1]
	} else {
		log.Trace().Msgf("found single MRC in the mesh when handling MRC event for MRC %s", mrc1.Name)
		// if there is only one MRC, set mrc2 equal to mrc1
		mrc2 = mrc1
	}

	log.Debug().Msg("validating MRC intent combination")
	if err := ValidateMRCIntents(mrc1, mrc2); err != nil {
		return nil, nil, err
	}

	if mrc1 == nil || mrc2 == nil {
		log.Error().Err(ErrUnexpectedNilMRC).Msg("unexpected nil MRC provided when validating MRC intents")
		return nil, nil, ErrUnexpectedNilMRC
	}

	intent1 := mrc1.Spec.Intent
	intent2 := mrc2.Spec.Intent

	switch intent1 {
	case v1alpha2.ActiveIntent:
		switch intent2 {
		case v1alpha2.PassiveIntent:
			return mrc1, mrc2, nil
		case v1alpha2.ActiveIntent:
			// If mrc1 != mrc2 and both MRCs have active intents, their state is non
			// deterministic. To avoid continuously resetting the issuers, only update
			// if the issuers have not already been set.
			return mrc1, mrc2, nil
		default:
			log.Error().Err(ErrInvalidMRCIntentCombination).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidMRCIntentCombination)).
				Msgf("invalid combination of %s intent and %s intent", intent1, intent2)
			return nil, nil, ErrInvalidMRCIntentCombination
		}
	case v1alpha2.PassiveIntent:
		switch intent2 {
		case v1alpha2.ActiveIntent:
			return mrc2, mrc1, nil
		default:
			log.Error().Err(ErrInvalidMRCIntentCombination).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidMRCIntentCombination)).
				Msgf("invalid combination of %s intent and %s intent", intent1, intent2)
			return nil, nil, ErrInvalidMRCIntentCombination
		}
	default:
		log.Error().Err(ErrUnknownMRCIntent).Msgf("invalid combination of %s intent and %s intent", intent1, intent2)
		return nil, nil, ErrUnknownMRCIntent
	}
}

func (m *Manager) updateIssuers(signingIssuer, validatingIssuer *issuer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.signingIssuer = signingIssuer
	m.validatingIssuer = validatingIssuer
	log.Trace().Msgf("setting issuers for validating[%s] and signing[%s]", validatingIssuer.ID, signingIssuer.ID)
	return nil
}

func (m *Manager) getCertIssuers(signingMRC, validatingMRC *v1alpha2.MeshRootCertificate) (*issuer, *issuer, error) {
	if signingMRC == nil || validatingMRC == nil {
		log.Error().Err(ErrUnexpectedNilMRC).Msg("unexpected nil MRC provided when getting cert issuers from MRCs")
		return nil, nil, ErrUnexpectedNilMRC
	}

	var desiredSigningIssuer, desiredValidatingIssuer *issuer
	desiredSigningIssuer, err := m.getCertIssuer(signingMRC)
	if err != nil {
		return nil, nil, err
	}
	// don't get the issuer again if there is a single MRC in the control plane
	if signingMRC == validatingMRC {
		desiredValidatingIssuer = desiredSigningIssuer
	} else {
		desiredValidatingIssuer, err = m.getCertIssuer(validatingMRC)
		if err != nil {
			return nil, nil, err
		}
	}

	return desiredSigningIssuer, desiredValidatingIssuer, nil
}

func (m *Manager) getCertIssuer(mrc *v1alpha2.MeshRootCertificate) (*issuer, error) {
	m.mu.Lock()
	signingIssuer := m.signingIssuer
	validatingIssuer := m.validatingIssuer
	m.mu.Unlock()

	// if the issuer has already been created for the specified mrc,
	// return the existing issuer
	if signingIssuer != nil && mrc.Name == signingIssuer.ID {
		return signingIssuer, nil
	}
	if validatingIssuer != nil && mrc.Name == validatingIssuer.ID {
		return validatingIssuer, nil
	}

	client, ca, err := m.mrcClient.GetCertIssuerForMRC(mrc)
	if err != nil {
		return nil, err
	}

	c := &issuer{Issuer: client, ID: mrc.Name, CertificateAuthority: ca, TrustDomain: mrc.Spec.TrustDomain, SpiffeEnabled: mrc.Spec.SpiffeEnabled}
	return c, nil
}

func filterOutInactiveMRCs(mrcList []*v1alpha2.MeshRootCertificate) []*v1alpha2.MeshRootCertificate {
	n := 0
	for _, mrc := range mrcList {
		if mrc.Spec.Intent != v1alpha2.InactiveIntent {
			mrcList[n] = mrc
			n++
		}
	}
	return mrcList[:n]
}
