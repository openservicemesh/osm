package profile

import "github.com/pyroscope-io/client/pyroscope"

const (
	// PyroscopeServerAddress is the address of the Pyroscope server that OSM control plane components will connect to
	PyroscopeServerAddress = "http://osm-pyroscope:4040"
)

// ConnectPyroscope connects to the Pyroscope server with the given app name.
func ConnectPyroscope(appName string) error {
	_, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: appName,
		ServerAddress:   PyroscopeServerAddress,
		Logger:          pyroscope.StandardLogger,

		// by default all profilers are enabled,
		// but you can select the ones you want to use:
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
	return err
}
