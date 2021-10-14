package crdconversion

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// serveMeshConfigConversion servers endpoint for the converter defined as convertMeshConfig function.
func serveMeshConfigConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertMeshConfig)
}

// convertMeshConfig contains the business logic to convert meshconfigs.config.openservicemesh.io CRD
// Example implementation reference : https://github.com/kubernetes/kubernetes/blob/release-1.22/test/images/agnhost/crd-conversion-webhook/converter/example_converter.go
func convertMeshConfig(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, statusErrorWithMessage("MeshConfig: conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	log.Debug().Msg("MeshConfig: successfully converted object")
	return convertedObject, statusSucceed()
}
