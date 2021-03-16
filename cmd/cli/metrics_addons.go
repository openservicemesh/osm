package main

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/naoina/toml"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Specific names used by AzMon Backend
	azmonConfigMapName = "container-azm-ms-osmconfig"
	azmonCfmapOsmKey   = "osm_metric_collection_configuration"

	// Schema supported in config map
	cfgMapSchemaVersionKey = "schema-version"
	supportedSchema        = "v1"

	// Config version tracking
	cfgMapConfigVersionKey = "config-version"
)

// azMonConfStruct maps the expected collection configuration of AZ monitor
type azMonConfStruct struct {
	AzMonCollectionConf struct {
		Settings struct {
			MonitorNs []string `toml:"monitor_namespaces"`
		} `toml:"settings"`
	} `toml:"osm_metric_collection_configuration"`
}

// runAzmonEnable will create or update Azure Monitor config map (located in `osmNamespace`)
// with the new namespaces to be added to the list of metric-enabled namespaces
func (cmd *metricsEnableCmd) runAzmonEnable(osmNamespace string) error {
	create := false

	var azmonSettings azMonConfStruct
	azCfgMap, err := cmd.clientSet.CoreV1().ConfigMaps(osmNamespace).Get(context.Background(), azmonConfigMapName, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// ConfigMap doesn't exist. Bootstrap one in memory
			azCfgMap = getNewAzmonConfigMap(osmNamespace)
			azmonSettings = getNewAzmonCollectionConfig()
			create = true
		} else {
			// Other error
			return err
		}
	} else {
		// ConfigMap exists
		// Check and parse inner data
		tomlData, ok := azCfgMap.Data[azmonCfmapOsmKey]
		if !ok {
			azmonSettings = getNewAzmonCollectionConfig()
		} else {
			// it exists, parse it
			err = toml.Unmarshal([]byte(tomlData), &azmonSettings)
			if err != nil {
				return err
			}
		}
	}

	// Transform slices to maps for easy intersection
	onConfigMap := getMapFromSlice(azmonSettings.AzMonCollectionConf.Settings.MonitorNs)
	toAdd := getMapFromSlice(cmd.namespaces)
	for toAddKey := range toAdd {
		onConfigMap[toAddKey] = true
	}

	// Transform result back to slice and encode to TOML
	azmonSettings.AzMonCollectionConf.Settings.MonitorNs = getSliceFromMap(onConfigMap)
	var buf bytes.Buffer
	if err = toml.NewEncoder(&buf).Encode(azmonSettings); err != nil {
		return err
	}

	// Update monitor settings in config map
	azCfgMap.Data[azmonCfmapOsmKey] = buf.String()
	if err = increaseConfigVersion(azCfgMap); err != nil {
		return err
	}

	// Create or update to k8s
	if create {
		_, err = cmd.clientSet.CoreV1().ConfigMaps(osmNamespace).Create(context.Background(), azCfgMap, v1.CreateOptions{})
	} else {
		_, err = cmd.clientSet.CoreV1().ConfigMaps(osmNamespace).Update(context.Background(), azCfgMap, v1.UpdateOptions{})
	}
	return err
}

// runAzmonDisable will update Azure Monitor config map (located in `osmNamespace`)
// removing the namespace from the list, if present
func (cmd *metricsDisableCmd) runAzmonDisable(osmNamespace string) error {
	azCfgMap, err := cmd.clientSet.CoreV1().ConfigMaps(osmNamespace).Get(context.Background(), azmonConfigMapName, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Nothing to do
			return nil
		}
		// Other error
		return err
	}

	azmonSettings, ok := azCfgMap.Data[azmonCfmapOsmKey]
	if !ok {
		// No config for settings was found
		return nil
	}

	// Decode TOML
	tempData := azMonConfStruct{}
	err = toml.Unmarshal([]byte(azmonSettings), &tempData)
	if err != nil {
		return err
	}

	// Transform slices into maps for easy intersection
	onConfigMap := getMapFromSlice(tempData.AzMonCollectionConf.Settings.MonitorNs)
	toDelete := getMapFromSlice(cmd.namespaces)
	for toDeleteKey := range toDelete {
		delete(onConfigMap, toDeleteKey)
	}

	// Result back to slice
	tempData.AzMonCollectionConf.Settings.MonitorNs = getSliceFromMap(onConfigMap)
	var buf bytes.Buffer
	err = toml.NewEncoder(&buf).Encode(tempData)
	if err != nil {
		return err
	}

	// Replace settings in config map and push to K8s
	azCfgMap.Data[azmonCfmapOsmKey] = buf.String()
	if err = increaseConfigVersion(azCfgMap); err != nil {
		return err
	}
	_, err = cmd.clientSet.CoreV1().ConfigMaps(osmNamespace).Update(context.Background(), azCfgMap, v1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func getNewAzmonConfigMap(ns string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      azmonConfigMapName,
			Namespace: ns,
		},
		Data: map[string]string{
			cfgMapSchemaVersionKey: supportedSchema,
			cfgMapConfigVersionKey: "0",
		},
	}
}

func increaseConfigVersion(cfgMap *corev1.ConfigMap) error {
	val, ok := cfgMap.Data[cfgMapConfigVersionKey]
	if !ok {
		return errors.Errorf("%s not found in config map", cfgMapConfigVersionKey)
	}
	ver, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return err
	}

	cfgMap.Data[cfgMapConfigVersionKey] = fmt.Sprintf("%d", ver+1)
	return nil
}

func getNewAzmonCollectionConfig() azMonConfStruct {
	return azMonConfStruct{}
}

func getMapFromSlice(s []string) map[string]bool {
	m := map[string]bool{}
	for _, value := range s {
		m[value] = true
	}
	return m
}

func getSliceFromMap(m map[string]bool) []string {
	s := []string{}
	for mapKey := range m {
		s = append(s, mapKey)
	}
	return s
}
