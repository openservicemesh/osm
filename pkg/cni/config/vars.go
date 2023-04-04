package config

var (
	// KernelTracing indicates debug feature of/off
	KernelTracing = false
	// EnableCNI indicates CNI feature enable/disable
	EnableCNI = false
	// IsKind indicates Kubernetes running in Docker
	IsKind = false
	// HostProc defines HostProc volume
	HostProc string
	// CNIBinDir defines CNIBIN volume
	CNIBinDir string
	// CNIConfigDir defines CNIConfig volume
	CNIConfigDir string
	// HostVarRun defines HostVar volume
	HostVarRun string
)
