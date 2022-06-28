package validator

import (
	"context"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MRCInfo struct {
	storedMRC    *configv1alpha2.MeshRootCertificate
	suggestedMRC *configv1alpha2.MeshRootCertificate
	allStoredMRC []configv1alpha2.MeshRootCertificate
}

func newMRCInfo() MRCInfo {
	return MRCInfo{}
}

func (m MRCInfo) countActiveMRCs() int {
	var active int
	for _, mrc := range m.allStoredMRC {
		if mrc.Status.State == constants.MRCStateActive {
			active++
		}
	}
	// we could probably store names of active certs and
	// return them for a better user exp. ?
	return active
}

func (m MRCInfo) getAllStoredMRC(cv configValidator, ns string) error {
	mrcs, err := cv.configClient.ConfigV1alpha2().MeshRootCertificates(ns).
		List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	m.allStoredMRC = mrcs.Items
	return nil
}

func (m MRCInfo) getStoredMRC() bool {
	for _, v := range m.allStoredMRC {
		if m.suggestedMRC.Name == v.Name {
			m.storedMRC = &v
		}
	}
	return false
}

func (m MRCInfo) validateMRCdelete() error {
	//Delete inactive or error only
	switch m.storedMRC.Status.State {
	case constants.MRCStateInactive, constants.MRCStateError:
		return nil
	default:
		return errors.Errorf("cannot delete certificate %v in stage %v", m.storedMRC.Name, m.storedMRC.Status.State)
	}
}

func (m MRCInfo) validateMRCupdate() bool {
	return m.validateMRCTransition()
}

func (m MRCInfo) validateMRCcreate() bool {
	return m.countActiveMRCs() < 2
}

func (m MRCInfo) validateMRCTransition() bool {
	allowedTransitions := map[string][]string{
		constants.MRCStateValidatingRollout:  {constants.MRCStateIssuingRollout, constants.MRCStateError},
		constants.MRCStateIssuingRollout:     {constants.MRCStateActive, constants.MRCStateError},
		constants.MRCStateActive:             {constants.MRCStateValidatingRollback, constants.MRCStateError},
		constants.MRCStateValidatingRollback: {constants.MRCStateIssuingRollback, constants.MRCStateError},
		constants.MRCStateIssuingRollback:    {constants.MRCStateInactive, constants.MRCStateError},
	}
	//look up storedMRC state key
	//applied state must be in the values for that key
	if allowedStates, ok := allowedTransitions[m.storedMRC.Status.State]; ok {
		//if going into active, safety check we have less than two
		if m.suggestedMRC.Status.State == constants.MRCStateActive {
			return m.countActiveMRCs() < 2
		}
		for _, state := range allowedStates {
			return m.suggestedMRC.Status.State == state

		}

	}
	// on false we could probably return []string of allowedStates for better
	// user exp. but that makes the logic less pretty
	return false
}
