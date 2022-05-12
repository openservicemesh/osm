package injector

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPortExclusionListForPod(t *testing.T) {
	testCases := []struct {
		name          string
		podAnnotation map[string]string
		forAnnotation string
		expectedError error
		expectedPorts []int
	}{
		{
			name:          "contains outbound port exclusion list annotation",
			podAnnotation: map[string]string{outboundPortExclusionListAnnotation: "6060, 7070"},
			forAnnotation: outboundPortExclusionListAnnotation,
			expectedError: nil,
			expectedPorts: []int{6060, 7070},
		},
		{
			name:          "contains inbound port exclusion list annotation",
			podAnnotation: map[string]string{inboundPortExclusionListAnnotation: "6060, 7070"},
			forAnnotation: inboundPortExclusionListAnnotation,
			expectedError: nil,
			expectedPorts: []int{6060, 7070},
		},
		{
			name:          "does not contains any port exclusion list annontation",
			podAnnotation: nil,
			forAnnotation: "",
			expectedError: nil,
			expectedPorts: nil,
		},
		{
			name:          "contains outbound port exclusion list annontation but invalid port",
			podAnnotation: map[string]string{outboundPortExclusionListAnnotation: "6060, -7070"},
			forAnnotation: outboundPortExclusionListAnnotation,
			expectedError: errors.Errorf("Invalid port value '%s' specified for annotation '%s'", "-7070", outboundPortExclusionListAnnotation),
			expectedPorts: nil,
		},
		{
			name:          "contains inbound port exclusion list annontation but invalid port",
			podAnnotation: map[string]string{inboundPortExclusionListAnnotation: "6060, -7070"},
			forAnnotation: inboundPortExclusionListAnnotation,
			expectedError: errors.Errorf("Invalid port value '%s' specified for annotation '%s'", "-7070", inboundPortExclusionListAnnotation),
			expectedPorts: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod-test",
					Annotations: tc.podAnnotation,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "test-SA",
				},
			}

			ports, err := getPortExclusionListForPod(pod, "test", tc.forAnnotation)
			if tc.expectedError != nil {
				a.EqualError(tc.expectedError, err.Error())
			} else {
				a.Nil(err)
			}
			a.ElementsMatch(tc.expectedPorts, ports)
		})
	}
}

func TestMergePortExclusionLists(t *testing.T) {
	testCases := []struct {
		name                              string
		podOutboundPortExclusionList      []int
		globalOutboundPortExclusionList   []int
		expectedOutboundPortExclusionList []int
	}{
		{
			name:                              "overlap in global and pod outbound exclusion list",
			podOutboundPortExclusionList:      []int{6060, 7070},
			globalOutboundPortExclusionList:   []int{6060, 8080},
			expectedOutboundPortExclusionList: []int{6060, 7070, 8080},
		},
		{
			name:                              "no overlap in global and pod outbound exclusion list",
			podOutboundPortExclusionList:      []int{6060, 7070},
			globalOutboundPortExclusionList:   []int{8080},
			expectedOutboundPortExclusionList: []int{6060, 7070, 8080},
		},
		{
			name:                              "pod outbound exclusion list is nil",
			podOutboundPortExclusionList:      nil,
			globalOutboundPortExclusionList:   []int{8080},
			expectedOutboundPortExclusionList: []int{8080},
		},
		{
			name:                              "global outbound exclusion list is nil",
			podOutboundPortExclusionList:      []int{6060, 7070},
			globalOutboundPortExclusionList:   nil,
			expectedOutboundPortExclusionList: []int{6060, 7070},
		},
		{
			name:                              "no global or pod level outbound exclusion list",
			podOutboundPortExclusionList:      nil,
			globalOutboundPortExclusionList:   nil,
			expectedOutboundPortExclusionList: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := mergePortExclusionLists(tc.podOutboundPortExclusionList, tc.globalOutboundPortExclusionList)
			a.ElementsMatch(tc.expectedOutboundPortExclusionList, actual)
		})
	}
}

func TestGetOutboundIPRangeListForPod(t *testing.T) {
	testCases := []struct {
		name             string
		podAnnotation    map[string]string
		forAnnotation    string
		expectedError    error
		expectedIPRanges []string
	}{
		{
			name:             "valid exclusion annotation",
			podAnnotation:    map[string]string{outboundIPRangeExclusionListAnnotation: "10.0.0.0/8, 2.2.2.2/32"},
			forAnnotation:    outboundIPRangeExclusionListAnnotation,
			expectedError:    nil,
			expectedIPRanges: []string{"10.0.0.0/8", "2.2.2.2/32"},
		},
		{
			name:             "no exclusion annotation",
			podAnnotation:    nil,
			forAnnotation:    outboundIPRangeExclusionListAnnotation,
			expectedError:    nil,
			expectedIPRanges: nil,
		},
		{
			name:             "valid inclusion annotation",
			podAnnotation:    map[string]string{outboundIPRangeInclusionListAnnotation: "10.0.0.0/8, 2.2.2.2/32"},
			forAnnotation:    outboundIPRangeInclusionListAnnotation,
			expectedError:    nil,
			expectedIPRanges: []string{"10.0.0.0/8", "2.2.2.2/32"},
		},
		{
			name:             "no inclusion annotation",
			podAnnotation:    nil,
			forAnnotation:    outboundIPRangeInclusionListAnnotation,
			expectedError:    nil,
			expectedIPRanges: nil,
		},
		{
			name:             "invalid annotation",
			podAnnotation:    map[string]string{outboundIPRangeExclusionListAnnotation: "foobar"},
			forAnnotation:    outboundIPRangeExclusionListAnnotation,
			expectedError:    errors.Errorf("Invalid IP range 'foobar' specified for annotation '%s'", outboundIPRangeExclusionListAnnotation),
			expectedIPRanges: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod-test",
					Annotations: tc.podAnnotation,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "test-SA",
				},
			}

			ipRanges, err := getOutboundIPRangeListForPod(pod, "test", tc.forAnnotation)
			if tc.expectedError != nil {
				a.EqualError(tc.expectedError, err.Error())
			} else {
				a.Nil(err)
			}
			a.ElementsMatch(tc.expectedIPRanges, ipRanges)
		})
	}
}

func TestMergeIPRangeLists(t *testing.T) {
	testCases := []struct {
		name        string
		podSpecific []string
		global      []string
		expected    []string
	}{
		{
			name:        "handles duplicates",
			podSpecific: []string{"1.1.1.1/32", "2.2.2.2/32"},
			global:      []string{"2.2.2.2/32", "3.3.3.3/32"},
			expected:    []string{"1.1.1.1/32", "2.2.2.2/32", "3.3.3.3/32"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			actual := mergeIPRangeLists(tc.podSpecific, tc.global)
			a.ElementsMatch(tc.expected, actual)
		})
	}
}
