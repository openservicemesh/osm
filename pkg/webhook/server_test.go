package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	w := httptest.NewRecorder()
	healthHandler(w, nil)

	res := w.Result()
	tassert.Equal(t, http.StatusOK, res.StatusCode)
}
