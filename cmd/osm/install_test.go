package main

import (
	"reflect"
	"testing"
)

func TestResolveValues(t *testing.T) {
	installCmd := &installCmd{
		containerRegistry:       "test-registry",
		containerRegistrySecret: "test-registry-secret",
	}
	vals, err := installCmd.resolveValues()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]interface{}{
		"image": map[string]interface{}{
			"registry": "test-registry",
		},
		"imagePullSecrets": []interface{}{
			map[string]interface{}{
				"name": "test-registry-secret",
			},
		},
	}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("Expected values to resolve as %#v\nbut got %#v", expected, vals)
	}
}
