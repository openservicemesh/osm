package reconciler

import (
	"context"
	reflect "reflect"
	"strings"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/errcode"
)

// crdEventHandler creates crd events handlers.
func (c client) crdEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCrd := oldObj.(*apiv1.CustomResourceDefinition)
			newCrd := newObj.(*apiv1.CustomResourceDefinition)
			log.Debug().Msgf("CRD update event for %s", newCrd.Name)
			if !isCRDUpdated(oldCrd, newCrd) {
				return
			}
			c.reconcileCrd(oldCrd, newCrd)
		},

		DeleteFunc: func(obj interface{}) {
			crd := obj.(*apiv1.CustomResourceDefinition)
			c.addCrd(crd)
			log.Debug().Msgf("CRD delete event for %s", crd.Name)
		},
	}
}

func (c client) reconcileCrd(oldCrd, newCrd *apiv1.CustomResourceDefinition) {
	newCrd.Spec = oldCrd.Spec
	newCrd.ObjectMeta.Name = oldCrd.ObjectMeta.Name
	newCrd.ObjectMeta.Labels = oldCrd.ObjectMeta.Labels
	if _, err := c.apiServerClient.ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), newCrd, metav1.UpdateOptions{}); err != nil {
		// There might be conflicts when multiple osm-bootstraps try to update the same resource
		// One of the bootstrap will successfully update the resource, hence conflicts shoud be ignored and not treated as an error
		if !apierrors.IsConflict(err) {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrReconcilingUpdatedCRD)).
				Msgf("Error updating crd: %s", newCrd.Name)
		}
	}
	log.Debug().Msgf("Successfully reconciled CRD %s", newCrd.Name)
}

func (c client) addCrd(oldCrd *apiv1.CustomResourceDefinition) {
	oldCrd.ResourceVersion = ""
	if _, err := c.apiServerClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.Background(), oldCrd, metav1.CreateOptions{}); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrReconcilingDeletedCRD)).
			Msgf("Error adding back deleted crd: %s", oldCrd.Name)
	}
	log.Debug().Msgf("Successfully added back CRD %s", oldCrd.Name)
}

func isCRDUpdated(oldCrd, newCrd *apiv1.CustomResourceDefinition) bool {
	crdSpecEqual := reflect.DeepEqual(oldCrd.Spec, newCrd.Spec)
	crdNameChanged := strings.Compare(oldCrd.ObjectMeta.Name, newCrd.ObjectMeta.Name) != 0
	crdUpdated := !crdSpecEqual || crdNameChanged
	return crdUpdated
}

func isLabelModified(key string, expectedValue string, labelMap map[string]string) bool {
	if value, ok := labelMap[key]; ok {
		if !strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(expectedValue)) {
			return true
		}
	} else {
		return true
	}
	return false
}
