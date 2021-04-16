package main

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestAnnotateErrorMessageWithActionableMessage(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		errorMsg     string
		name         string
		namespace    string
		exceptionMsg string
		annotatedMsg string
	}

	actionableMsg := "Use flags to modify the command to suit your needs"

	testCases := []test{
		{
			"Error message with args such as [name: %s], [namespace: %s], and [err: %s]",
			"osm-name",
			"osm-namespace",
			"osm-exception",
			"Error message with args such as [name: osm-name], [namespace: osm-namespace], and [err: osm-exception]\n\n" + actionableMsg,
		},
	}

	for _, tc := range testCases {
		t.Run("Testing annotated error message", func(t *testing.T) {
			assert.Equal(
				tc.annotatedMsg,
				annotateErrorMessageWithActionableMessage(actionableMsg, tc.errorMsg, tc.name, tc.namespace, tc.exceptionMsg).Error())
		})
	}
}

func TestAnnotateErrorMessageWithOsmNamespace(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		errorMsg     string
		name         string
		namespace    string
		exceptionMsg string
		annotatedMsg string
	}

	osmNamespaceErrorMsg := fmt.Sprintf(
		"Note: The command failed when run in the OSM namespace [%s].\n"+
			"Use the global flag --osm-namespace if [%s] is not the intended OSM namespace.",
		settings.Namespace(), settings.Namespace())

	testCases := []test{
		{
			"Error message with args such as [name: %s], [namespace: %s], and [err: %s]",
			"osm-name",
			"osm-namespace",
			"osm-exception",
			"Error message with args such as [name: osm-name], [namespace: osm-namespace], and [err: osm-exception]\n\n" + osmNamespaceErrorMsg,
		},
	}

	for _, tc := range testCases {
		t.Run("Testing annotated error message", func(t *testing.T) {
			assert.Equal(
				tc.annotatedMsg,
				annotateErrorMessageWithOsmNamespace(tc.errorMsg, tc.name, tc.namespace, tc.exceptionMsg).Error())
		})
	}
}
