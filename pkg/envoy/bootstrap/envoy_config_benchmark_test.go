package bootstrap

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/models"
)

func BenchmarkBuildEnvoyBootstrapConfig(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	assert := tassert.New(b)

	cert := tresorFake.NewFakeCertificate()
	builder := &Builder{
		NodeID:                cert.GetCommonName().String(),
		XDSHost:               "osm-controller.osm-system.svc.cluster.local",
		TLSMinProtocolVersion: "TLSv1_0",
		TLSMaxProtocolVersion: "TLSv1_2",
		CipherSuites:          []string{"abc", "xyz"},
		ECDHCurves:            []string{"ABC", "XYZ"},
		OriginalHealthProbes: models.HealthProbes{
			Liveness:  &models.HealthProbe{Path: "/liveness", Port: 81, IsHTTP: true},
			Readiness: &models.HealthProbe{Path: "/readiness", Port: 82, IsHTTP: true},
			Startup:   &models.HealthProbe{Path: "/startup", Port: 83, IsHTTP: true},
		},
	}

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		bootstrapConfig, err := builder.Build()
		assert.Nil(err)
		assert.NotNil(bootstrapConfig)
	}

	b.StopTimer()
}
