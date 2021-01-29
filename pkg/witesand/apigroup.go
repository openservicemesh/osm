package witesand

import (
	"encoding/json"
	"fmt"
	"net/http"
)
func (wc *WitesandCatalog) UpdateApigroupMap(w http.ResponseWriter, r *http.Request) {
	var input map[string][]string
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil  {
		log.Error().Msgf("[UpdateApigroupMap] JSON decode err:%s", err)
		w.WriteHeader(400)
		fmt.Fprintf(w, "Decode error! please check your JSON formating.")
		return
	}

	for apigroupName, pods := range input {
		atopMap := ApigroupToPodMap{
			Apigroup: apigroupName,
			Pods:     pods,
		}
		if len(atopMap.Pods) == 0 {
			// DELETE pods
			log.Info().Msgf("[UpdateApigroupMap] DELETE apigroup:%s", apigroupName)
			delete(wc.apigroupToPodMap, apigroupName)
			delete(wc.apigroupToPodIPMap, apigroupName)
		} else {
			// UPDATE pods
			log.Info().Msgf("[UpdateApigroupMap] POST apigroup:%s pods:%+v", apigroupName, atopMap.Pods)
			wc.apigroupToPodMap[apigroupName] = atopMap
			wc.resolveApigroup(atopMap)
		}
	}

	wc.updateEnvoy()
}

func (wc *WitesandCatalog) UpdateAllApigroupMaps(apigroupToPodMap *map[string][]string) {
	log.Info().Msgf("[UpdateAllApigroupMaps] updating %d apiggroups", len(*apigroupToPodMap))
	for apigroupName, pods := range *apigroupToPodMap {
		apigroupMap := ApigroupToPodMap{
			Apigroup: apigroupName,
			Pods:     pods,
		}
		wc.apigroupToPodMap[apigroupMap.Apigroup] = apigroupMap
	}
	wc.ResolveAllApigroups()
	wc.updateEnvoy()
}

// Resolve apigroup's pods to their respective IPs
func (wc *WitesandCatalog) resolveApigroup(atopmap ApigroupToPodMap) {
	atopipmap := ApigroupToPodIPMap{
		Apigroup: atopmap.Apigroup,
		PodIPs:   make([]string, 0),
	}
	for _, pod := range atopmap.Pods {
		podip := ""
		for _, podInfo := range wc.clusterPodMap {
			var exists bool
			if podip, exists = podInfo.PodToIPMap[pod]; exists {
				break
			}
		}
		if podip != "" {
			log.Info().Msgf("[resolveApigroup] RESOLVE pod:%s IP:%s", pod, podip)
			atopipmap.PodIPs = append(atopipmap.PodIPs, podip)
		} else {
			log.Info().Msgf("[resolveApigroup] CANNOT RESOLVE pod:%s !!", pod)
		}
	}
	wc.apigroupToPodIPMap[atopipmap.Apigroup] = atopipmap
}

func (wc *WitesandCatalog) ResolveAllApigroups() {
	log.Info().Msgf("[ResolveAllApigroups] Resovling all apigroups")
	for _, atopmap := range wc.apigroupToPodMap {
		wc.resolveApigroup(atopmap)
	}
}

func (wc *WitesandCatalog) ListApigroupClusterNames() ([]string, error) {
	var apigroups []string
	for apigroup, _ := range wc.apigroupToPodMap {
		apigroups = append(apigroups, apigroup)
	}

	return apigroups, nil
}

func (wc *WitesandCatalog) ListApigroupToPodIPs() ([]ApigroupToPodIPMap, error) {
	var atopipMaps []ApigroupToPodIPMap
	for _, atopipMap := range wc.apigroupToPodIPMap {
		atopipMaps = append(atopipMaps, atopipMap)
	}
	return atopipMaps, nil
}
