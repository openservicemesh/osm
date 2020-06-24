package configurator

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

const osmConfigFilename = "/etc/config/osm.conf"

// NewConfigurator implements namespace.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(stop chan struct{}, osmNamespace string) Configurator {
	configMapwatcher := ConfigMapWatcher{
		osmNamespace:     osmNamespace,
		announcements:    make(chan interface{}),
		refreshConfigMap: 1 * time.Second,
		configMap:        &configMap{},
	}

	go configMapwatcher.run(stop)

	return &configMapwatcher
}

// This struct must match the shape of the "osm-config" ConfigMap
// which was created in the OSM namespace.
type configMap struct {
	LogVerbosity string `json:"log_verbosity"`
}

func getOSMConfig() *configMap {
	configData, err := ioutil.ReadFile(osmConfigFilename)
	if err != nil {
		log.Error().Err(err).Msgf("Error reading OSM config file %s", osmConfigFilename)
	}

	conf := configMap{}
	err = yaml.Unmarshal(configData, &conf)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling file %s with content %s", osmConfigFilename, string(configData))
	}

	return &conf
}

func (c *ConfigMapWatcher) run(stop <-chan struct{}) {
	tick := time.Tick(c.refreshConfigMap)
	for {
		// TODO(draychev): Implement fsnotify -- watch the file for changes
		select {
		case <-tick:
			c.Lock()
			c.configMap = getOSMConfig()
			c.Unlock()
		case <-stop:
			return
		}
	}
}
