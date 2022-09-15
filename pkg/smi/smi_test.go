package smi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestGetSmiClientVersionHTTPHandler(t *testing.T) {
	a := assert.New(t)

	url := "http://localhost"
	testHTTPServerPort := 8888
	smiVerionPath := constants.OSMControllerSMIVersionPath
	recordCall := func(ts *httptest.Server, path string) *http.Response {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()

		ts.Config.Handler.ServeHTTP(w, req)

		return w.Result()
	}
	handlers := map[string]http.Handler{
		smiVerionPath: GetSmiClientVersionHTTPHandler(),
	}

	router := http.NewServeMux()
	for path, handler := range handlers {
		router.Handle(path, handler)
	}

	testServer := &httptest.Server{
		Config: &http.Server{
			Addr:              fmt.Sprintf(":%d", testHTTPServerPort),
			Handler:           router,
			ReadHeaderTimeout: time.Second * 10,
		},
	}

	// Verify
	resp := recordCall(testServer, fmt.Sprintf("%s%s", url, smiVerionPath))
	a.Equal(http.StatusOK, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	a.Nil(err)
	a.Equal(`{"HTTPRouteGroup":"specs.smi-spec.io/v1alpha4","TCPRoute":"specs.smi-spec.io/v1alpha4","TrafficSplit":"split.smi-spec.io/v1alpha2","TrafficTarget":"access.smi-spec.io/v1alpha3"}`, string(bodyBytes))
}

func TestHasValidRules(t *testing.T) {
	testCases := []struct {
		name           string
		expectedResult bool
		rules          []smiAccess.TrafficTargetRule
	}{
		{
			name:           "has no rules",
			expectedResult: false,
			rules:          []smiAccess.TrafficTargetRule{},
		},
		{
			name:           "has rule with invalid kind",
			expectedResult: false,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test",
					Kind:    "Invalid",
					Matches: []string{},
				},
			},
		},
		{
			name:           "has rule with valid HTTPRouteGroup kind",
			expectedResult: true,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test",
					Kind:    HTTPRouteGroupKind,
					Matches: []string{},
				},
			},
		},
		{
			name:           "has rule with valid TCPRouteGroup kind",
			expectedResult: true,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test",
					Kind:    TCPRouteKind,
					Matches: []string{},
				},
			},
		},
		{
			name:           "has multiple rules with valid and invalid kind",
			expectedResult: false,
			rules: []smiAccess.TrafficTargetRule{
				{
					Name:    "test1",
					Kind:    TCPRouteKind,
					Matches: []string{},
				},
				{
					Name:    "test2",
					Kind:    HTTPRouteGroupKind,
					Matches: []string{},
				},
				{
					Name:    "test2",
					Kind:    "invalid",
					Matches: []string{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			result := HasValidRules(tc.rules)
			a.Equal(tc.expectedResult, result)
		})
	}
}

func TestIsValidTrafficTarget(t *testing.T) {
	testCases := []struct {
		name           string
		expectedResult bool
		trafficTarget  *smiAccess.TrafficTarget
	}{
		{
			name:           "traffic target namespace does not match destination namespace",
			expectedResult: false,
			trafficTarget: &smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "traffic-target-namespace",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "destination-namespace",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
					Rules: []smiAccess.TrafficTargetRule{
						{
							Kind: "TCPRoute",
							Name: "route-1",
						},
					},
				},
			},
		},
		{
			name:           "traffic target is valid",
			expectedResult: true,
			trafficTarget: &smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "namespace",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "namespace",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
					Rules: []smiAccess.TrafficTargetRule{
						{
							Kind: "TCPRoute",
							Name: "route-1",
						},
					},
				},
			},
		},
		{
			name:           "traffic target has invalid rules",
			expectedResult: false,
			trafficTarget: &smiAccess.TrafficTarget{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "access.smi-spec.io/v1alpha3",
					Kind:       "TrafficTarget",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-1",
					Namespace: "traffic-target-namespace",
				},
				Spec: smiAccess.TrafficTargetSpec{
					Destination: smiAccess.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "sa-2",
						Namespace: "destination-namespace",
					},
					Sources: []smiAccess.IdentityBindingSubject{{
						Kind:      "ServiceAccount",
						Name:      "sa-1",
						Namespace: "ns-1",
					}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			result := IsValidTrafficTarget(tc.trafficTarget)
			a.Equal(tc.expectedResult, result)
		})
	}
}
