package health

// FakeProbe is a fake health.Probes impl for Liveness/Readiness testing
type FakeProbe struct {
	LivenessRet  Probe
	ReadinessRet Probe
}

// Liveness is interface impl from health.Probes
func (t FakeProbe) Liveness() bool {
	return t.LivenessRet()
}

// Readiness is interface impl from health.Probes
func (t FakeProbe) Readiness() bool {
	return t.ReadinessRet()
}
