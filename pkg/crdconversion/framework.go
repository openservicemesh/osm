/*
Copyright 2018 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package crdconversion

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/munnerz/goautoneg"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// convertFunc is the user defined function for any conversion. The code in this file is a
// template that can be use for any CR conversion given this function.
type convertFunc func(Object *unstructured.Unstructured, version string) (*unstructured.Unstructured, error)

// conversionResponseFailureWithMessagef is a helper function to create an AdmissionResponse
// with a formatted embedded error message.
func conversionResponseFailureWithMessagef(msg string, params ...interface{}) *v1beta1.ConversionResponse {
	return &v1beta1.ConversionResponse{
		Result: metav1.Status{
			Message: fmt.Sprintf(msg, params...),
			Status:  metav1.StatusFailure,
		},
	}
}

// doConversion converts the requested object given the conversion function and returns a conversion response.
// failures will be reported as Reason in the conversion response.
func doConversion(convertRequest *v1beta1.ConversionRequest, convert convertFunc) *v1beta1.ConversionResponse {
	var convertedObjects []runtime.RawExtension
	// aggregate errors from all objects in the request vs. only returning the
	// first so errors from all objects are sent in the request and metrics are
	// recorded as accurately as possible.
	var errs []string
	for i, obj := range convertRequest.Objects {
		cr := unstructured.Unstructured{}
		if err := cr.UnmarshalJSON(obj.Raw); err != nil {
			log.Error().Err(err).Msg("error unmarshalling object JSON")
			errs = append(errs, fmt.Sprintf("failed to unmarshal object (%v) with error: %v", string(obj.Raw), err))
			continue
		}
		// Save the parsed object to read from later when metrics are recorded
		convertRequest.Objects[i].Object = &cr
		convertedCR, err := convert(&cr, convertRequest.DesiredAPIVersion)
		if err != nil {
			log.Error().Err(err).Msg("conversion failed")
			errs = append(errs, err.Error())
			continue
		}
		convertedCR.SetAPIVersion(convertRequest.DesiredAPIVersion)
		convertedObjects = append(convertedObjects, runtime.RawExtension{Object: convertedCR})
	}

	resp := &v1beta1.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result: metav1.Status{
			Status: metav1.StatusSuccess,
		},
	}
	if len(errs) > 0 {
		resp = &v1beta1.ConversionResponse{
			Result: metav1.Status{
				Message: strings.Join(errs, "; "),
				Status:  metav1.StatusFailure,
			},
		}
	}
	return resp
}

func serve(w http.ResponseWriter, r *http.Request, convert convertFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	contentType := r.Header.Get("Content-Type")
	serializer := getInputSerializer(contentType)
	if serializer == nil {
		msg := fmt.Sprintf("invalid Content-Type header `%s`", contentType)
		log.Error().Msgf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	convertReview := v1beta1.ConversionReview{}
	if _, _, err := serializer.Decode(body, nil, &convertReview); err != nil {
		log.Error().Err(err).Msgf("failed to deserialize body (%v)", string(body))
		convertReview.Response = conversionResponseFailureWithMessagef("failed to deserialize body (%v) with error %v", string(body), err)
	} else {
		convertReview.Response = doConversion(convertReview.Request, convert)
		convertReview.Response.UID = convertReview.Request.UID
	}

	var success bool
	if convertReview.Response != nil {
		success = convertReview.Response.Result.Status == metav1.StatusSuccess
	}
	if convertReview.Request != nil {
		toVersion := convertReview.Request.DesiredAPIVersion
		for _, reqObj := range convertReview.Request.Objects {
			var fromVersion string
			var kind string
			if reqObj.Object != nil && reqObj.Object.GetObjectKind() != nil {
				fromVersion = reqObj.Object.GetObjectKind().GroupVersionKind().GroupVersion().String()
				kind = reqObj.Object.GetObjectKind().GroupVersionKind().Kind
			}
			metricsstore.DefaultMetricsStore.ConversionWebhookResourceTotal.WithLabelValues(kind, fromVersion, toVersion, strconv.FormatBool(success)).Inc()
		}
	}

	log.Debug().Msgf(fmt.Sprintf("sending response: %v", convertReview.Response))

	// reset the request, it is not needed in a response.
	convertReview.Request = &v1beta1.ConversionRequest{}

	accept := r.Header.Get("Accept")
	outSerializer := getOutputSerializer(accept)
	if outSerializer == nil {
		msg := fmt.Sprintf("invalid accept header `%s`", accept)
		log.Error().Msgf(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err := outSerializer.Encode(&convertReview, w)
	if err != nil {
		log.Error().Err(err).Msg("error encoding ConversionReview")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type mediaType struct {
	Type, SubType string
}

var scheme = runtime.NewScheme()
var serializers = map[mediaType]runtime.Serializer{
	{"application", "json"}: json.NewSerializer(json.DefaultMetaFactory, scheme, scheme, false),
	{"application", "yaml"}: json.NewYAMLSerializer(json.DefaultMetaFactory, scheme, scheme),
}

func getInputSerializer(contentType string) runtime.Serializer {
	parts := strings.SplitN(contentType, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	return serializers[mediaType{parts[0], parts[1]}]
}

func getOutputSerializer(accept string) runtime.Serializer {
	if len(accept) == 0 {
		return serializers[mediaType{"application", "json"}]
	}

	clauses := goautoneg.ParseAccept(accept)
	for _, clause := range clauses {
		for k, v := range serializers {
			switch {
			case clause.Type == k.Type && clause.SubType == k.SubType,
				clause.Type == k.Type && clause.SubType == "*",
				clause.Type == "*" && clause.SubType == "*":
				return v
			}
		}
	}

	return nil
}
