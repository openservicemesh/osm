package crdconversion

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openservicemesh/osm/pkg/metricsstore"
)

func TestServe(t *testing.T) {
	runServe := func(req *v1beta1.ConversionRequest, convert convertFunc) (*httptest.ResponseRecorder, *v1beta1.ConversionReview) {
		j, err := json.Marshal(&v1beta1.ConversionReview{
			Request: req,
		})
		require.NoError(t, err)
		body := bytes.NewBuffer(j)
		r := httptest.NewRequest(http.MethodPost, "http://this.doesnt/matter", body)
		r.Header.Add("Content-Type", "application/json")
		r.Header.Add("Accept", "application/json")
		w := httptest.NewRecorder()

		serve(w, r, convert)

		res := new(v1beta1.ConversionReview)
		err = json.Unmarshal(w.Body.Bytes(), res)
		require.NoError(t, err)

		return w, res
	}

	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.ConversionWebhookResourceTotal)

	req := &v1beta1.ConversionRequest{
		DesiredAPIVersion: "any.group/v1",
		Objects: []runtime.RawExtension{
			{
				Raw: []byte(`{
						"apiVersion": "any.group/v2",
						"kind": "SomeKind"
					}`),
			},
			{
				Raw: []byte(`{
						"apiVersion": "any.group/v3",
						"kind": "SomeKind"
					}`),
			},
		},
	}
	failConvert := func(*unstructured.Unstructured, string) (*unstructured.Unstructured, error) {
		return nil, errors.New("fail")
	}
	okConvert := func(in *unstructured.Unstructured, _ string) (*unstructured.Unstructured, error) {
		return in.DeepCopy(), nil
	}
	v2FailV3OkConvert := func(in *unstructured.Unstructured, toVersion string) (*unstructured.Unstructured, error) {
		switch in.GetAPIVersion() {
		case "any.group/v2":
			return failConvert(in, toVersion)
		case "any.group/v3":
			return okConvert(in, toVersion)
		}
		panic("unexpected API version")
	}

	a := assert.New(t)

	// ok conversion
	w, res := runServe(req, okConvert)

	a.Equal(http.StatusOK, w.Result().StatusCode)
	a.Equal(metav1.StatusSuccess, res.Response.Result.Status)
	a.Len(res.Response.ConvertedObjects, 2)

	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_conversion_webhook_resource_total{from_version="any.group/v2",kind="SomeKind",success="true",to_version="any.group/v1"} 1` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_conversion_webhook_resource_total{from_version="any.group/v3",kind="SomeKind",success="true",to_version="any.group/v1"} 1` + "\n"))

	// failing conversion
	w, res = runServe(req, failConvert)

	a.Equal(http.StatusOK, w.Result().StatusCode)
	a.Equal(metav1.StatusFailure, res.Response.Result.Status)
	a.Len(res.Response.ConvertedObjects, 0)
	a.Equal("fail; fail", res.Response.Result.Message)

	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_conversion_webhook_resource_total{from_version="any.group/v2",kind="SomeKind",success="false",to_version="any.group/v1"} 1` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_conversion_webhook_resource_total{from_version="any.group/v3",kind="SomeKind",success="false",to_version="any.group/v1"} 1` + "\n"))

	// partially successful conversion
	w, res = runServe(req, v2FailV3OkConvert)

	a.Equal(http.StatusOK, w.Result().StatusCode)
	a.Equal(metav1.StatusFailure, res.Response.Result.Status)
	a.Len(res.Response.ConvertedObjects, 0)
	a.Equal("fail", res.Response.Result.Message)

	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_conversion_webhook_resource_total{from_version="any.group/v2",kind="SomeKind",success="false",to_version="any.group/v1"} 2` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_conversion_webhook_resource_total{from_version="any.group/v3",kind="SomeKind",success="false",to_version="any.group/v1"} 2` + "\n"))
}

func TestDoConversionBadObjectJSON(t *testing.T) {
	req := &v1beta1.ConversionRequest{
		DesiredAPIVersion: "any.group/v1",
		Objects: []runtime.RawExtension{
			{
				Raw: []byte(`{`),
			},
			{
				Raw: []byte(`{
						"apiVersion": "any.group/v3",
						"kind": "SomeKind"
					}`),
			},
		},
	}
	okConvert := func(in *unstructured.Unstructured, _ string) (*unstructured.Unstructured, error) {
		return in.DeepCopy(), nil
	}

	res := doConversion(req, okConvert)

	a := assert.New(t)

	a.Equal(metav1.StatusFailure, res.Result.Status)
	a.Contains(res.Result.Message, "unexpected end of JSON input")
	// Remaining objects still get parsed
	a.NotNil(req.Objects[1].Object)
}
