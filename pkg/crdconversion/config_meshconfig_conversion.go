package crdconversion

import (
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// serveMeshConfigConversion servers endpoint for the converter defined as convertMeshConfig function.
func serveMeshConfigConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertMeshConfig)
}

// convertMeshConfig contains the business logic to convert meshconfigs.config.openservicemesh.io CRD
// Example implementation reference : https://github.com/kubernetes/kubernetes/blob/release-1.22/test/images/agnhost/crd-conversion-webhook/converter/example_converter.go
func convertMeshConfig(obj *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, error) {
	convertedObject := obj.DeepCopy()
	fromVersion := obj.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, errors.Errorf("MeshConfig: conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	log.Debug().Msgf("MeshConfig conversion request: from-version=%s, to-version=%s", fromVersion, toVersion)
	switch fromVersion {
	case "config.openservicemesh.io/v1alpha1":
		switch toVersion {
		case "config.openservicemesh.io/v1alpha2":
			log.Debug().Msgf("Converting MeshConfig v1alpha1 -> v1alpha2")
			// v1alpha2 is backward compatible with v1alpha1, so no conversion is
			// necessary at this moment.

		default:
			return nil, errors.Errorf("Unexpected conversion to-version for MeshConfig resource: %s", toVersion)
		}

	case "config.openservicemesh.io/v1alpha2":
		switch toVersion {
		case "config.openservicemesh.io/v1alpha1":
			log.Debug().Msgf("Converting MeshConfig v1alpha2 -> v1alpha1")
			// Remove spec.traffic.outboundIPRangeInclusionList field not supported in v1alpha1
			unsupportedFields := [][]string{
				{"spec", "traffic", "outboundIPRangeInclusionList"},
				{"spec", "sidecar", "tlsMinProtocolVersion"},
				{"spec", "sidecar", "tlsMaxProtocolVersion"},
				{"spec", "sidecar", "cipherSuites"},
				{"spec", "sidecar", "ecdhCurves"},
			}

			for _, unsupportedField := range unsupportedFields {
				_, found, err := unstructured.NestedSlice(convertedObject.Object, unsupportedField...)
				if found && err == nil {
					unstructured.RemoveNestedField(convertedObject.Object, unsupportedField...)
				}
			}
		default:
			return nil, errors.Errorf("Unexpected conversion to-version for MeshConfig resource: %s", toVersion)
		}

	default:
		return nil, errors.Errorf("Unexpected conversion from-version for MeshConfig resource: %s", fromVersion)
	}

	log.Debug().Msg("MeshConfig: successfully converted object")
	return convertedObject, nil
}
