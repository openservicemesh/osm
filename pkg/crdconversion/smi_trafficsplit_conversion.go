package crdconversion

import (
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// serveTrafficSplitConversion servers endpoint for the converter defined as convertTrafficSplit function.
func serveTrafficSplitConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertTrafficSplit)
}

// convertTrafficSplit contains the business logic to convert trafficsplits.access.smi-spec.io CRD
// Example implementation reference : https://github.com/kubernetes/kubernetes/blob/release-1.22/test/images/agnhost/crd-conversion-webhook/converter/example_converter.go
func convertTrafficSplit(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, error) {
	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, errors.Errorf("TrafficSplit: conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	log.Debug().Msg("TrafficSplit: successfully converted object")
	return convertedObject, nil
}
