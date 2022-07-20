package crdconversion

import (
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// serveEgressPolicyConversion servers endpoint for the converter defined as convertEgressPolicy function.
func serveEgressPolicyConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertEgressPolicy)
}

// convertEgressPolicy contains the business logic to convert egresses.policy.openservicemesh.io CRD
// Example implementation reference : https://github.com/kubernetes/kubernetes/blob/release-1.22/test/images/agnhost/crd-conversion-webhook/converter/example_converter.go
func convertEgressPolicy(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, error) {
	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, fmt.Errorf("EgressPolicy: conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	log.Debug().Msg("EgressPolicy: successfully converted object")
	return convertedObject, nil
}
