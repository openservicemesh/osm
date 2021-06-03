package smi

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckSMICrdsExist(t *testing.T) {
	testCases := []struct {
		testName         string
		reqKinds         map[string]string
		candidateVersion []string
		expected         bool
	}{
		{
			testName:         "default",
			reqKinds:         map[string]string{"hi": "bye"},
			candidateVersion: []string{"bye"},
			expected:         true,
		},
		{
			testName:         "unable to get groupVersion",
			reqKinds:         map[string]string{},
			candidateVersion: []string{"abracadabra"},
			expected:         false,
		},
		{
			testName:         "did not find all required CRD versions",
			reqKinds:         map[string]string{"may": "flower"},
			candidateVersion: []string{},
			expected:         false,
		},
	}
	clientset := fake.NewSimpleClientset()
	// Adds resource hi with group version bye
	clientset.Resources = []*metav1.APIResourceList{{GroupVersion: "bye", APIResources: []metav1.APIResource{{Kind: "hi"}}}}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert := tassert.New(t)
			res := checkSMICrdsExist(clientset, tc.reqKinds, tc.candidateVersion)
			assert.Equal(tc.expected, res)
		})
	}
}
