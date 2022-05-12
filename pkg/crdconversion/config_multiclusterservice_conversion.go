package crdconversion

import (
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/openservicemesh/osm/pkg/constants"
)

// serveMultiClusterServiceConversion servers endpoint for the converter defined as convertMultiClusterService function.
func serveMultiClusterServiceConversion(w http.ResponseWriter, r *http.Request) {
	serve(w, r, convertMultiClusterService)
}

// convertMultiClusterService contains the business logic to convert multiclusterservices.config.openservicemesh.io CRD
// Example implementation reference : https://github.com/kubernetes/kubernetes/blob/release-1.22/test/images/agnhost/crd-conversion-webhook/converter/example_converter.go
func convertMultiClusterService(Object *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, error) {
	convertedObject := Object.DeepCopy()
	fromVersion := Object.GetAPIVersion()

	if toVersion == fromVersion {
		return nil, errors.Errorf("MultiClusterService: conversion from a version to itself should not call the webhook: %s", toVersion)
	}

	log.Debug().Str(constants.LogFieldContext, constants.LogContextMulticluster).Msg("MultiClusterService: successfully converted object")
	return convertedObject, nil
}
