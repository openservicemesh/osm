package ads

import (
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy"
)

func TestGetXDSLog(t *testing.T) {
	assert := tassert.New(t)

	testXDSLog := make(map[string]map[envoy.TypeURI][]time.Time)
	testXDSLog["abra"] = make(map[envoy.TypeURI][]time.Time)
	testXDSLog["abra"]["cadabra"] = []time.Time{time.Now()}

	s := Server{
		xdsLog: testXDSLog,
	}

	res := s.GetXDSLog()
	assert.Equal(res, testXDSLog)
}
