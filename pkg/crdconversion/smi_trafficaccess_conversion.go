package crdconversion

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// serveTrafficAccessConversion servers endpoint for the converter defined as convertTrafficAccess function.
func serveTrafficAccessConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertTrafficAccess)
}

// convertTrafficAccess contains the business logic to convert traffictargets.access.smi-spec.io CRD
// Example implementation reference : https://github.com/kubernetes/kubernetes/blob/release-1.22/test/images/agnhost/crd-conversion-webhook/converter/example_converter.go
func convertTrafficAccess(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, metav1.Status) {
	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, statusErrorWithMessage("TrafficAccess: conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	log.Debug().Msg("TrafficAccess: successfully converted object")
	return convertedObject, statusSucceed()
}
