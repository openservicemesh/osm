package webhook

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	errorTest = errors.New("error test")
)

func TestGetAdmissionRequestBody(t *testing.T) {
	assert := tassert.New(t)
	w := httptest.NewRecorder()

	testCases := []struct {
		testName string
		req      *http.Request
		expBody  []byte
		expErr   error
	}{
		{
			testName: "Err on nil request body",
			req:      httptest.NewRequest("GET", "/a/b/c", nil),
			expBody:  nil,
			expErr:   errEmptyAdmissionRequestBody,
		},
		{
			testName: "Err on empty request body",
			req:      httptest.NewRequest("GET", "/a/b/c", strings.NewReader("")),
			expBody:  nil,
			expErr:   errEmptyAdmissionRequestBody,
		},
		{
			testName: "Err on reading request body",
			req:      httptest.NewRequest("GET", "/a/b/c", err(5)),
			expBody:  []byte{0},
			expErr:   errorTest,
		},
		{
			testName: "Successfully read request body",
			req:      httptest.NewRequest("GET", "/a/b/c", strings.NewReader("hi123")),
			expBody:  []byte("hi123"),
			expErr:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			b, err := GetAdmissionRequestBody(w, tc.req)
			assert.Equal(tc.expErr, err)
			assert.Equal(tc.expBody, b)
		})
	}
}

func TestAdmissionError(t *testing.T) {
	assert := tassert.New(t)
	message := uuid.New().String()
	err := errors.New(message)
	actual := AdmissionError(err)
	expected := v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: message,
		},
	}
	assert.Equal(&expected, actual)
}

type err int

func (err) Read(_ []byte) (i int, err error) { return 1, errorTest }
