package witesand

import (
	"encoding/json"
	"fmt"
	"net/http"
)
func (wc *WitesandCatalog) UpdateApigroupMap(w http.ResponseWriter, method string, r *http.Request) {

	var input ApigroupToPodMap
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil  {
		w.WriteHeader(400)
		fmt.Fprintf(w, "Decode error! please check your JSON formating.")
		return
	}

	if method == "DELETE" {
		_, exists := wc.apigroupToPodMap[input.Apigroup]
		if !exists {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s doest not exist.", input.Apigroup)
			return
		}
		log.Info().Msgf("[UpdateApigroupMap] DELETE apigroup:%s", input.Apigroup)
		delete(wc.apigroupToPodMap, input.Apigroup)
		delete(wc.apigroupToPodIPMap, input.Apigroup)
		wc.updateEnvoy()
		return
	}
	if method == "POST" {
		_, exists := wc.apigroupToPodMap[input.Apigroup]
		if exists {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s already exists.", input.Apigroup)
			return
		}
		log.Info().Msgf("[UpdateApigroupMap] POST apigroup:%s pods:%+v", input.Apigroup, input.Pods)
		wc.apigroupToPodMap[input.Apigroup] = input
	}
	if method == "PUT" {
		_, exists := wc.apigroupToPodMap[input.Apigroup]
		if !exists {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s doest not exist.", input.Apigroup)
			return
		}
		/*
		if input.Revision < o.Revision {
			w.WriteHeader(400)
			fmt.Fprintf(w, "apigroup %s, revision(old:%d new:%d) stale.", input.Apigroup, o.Revision, input.Revision)
			return
		}
		*/
		log.Info().Msgf("[UpdateApigroupMap] PUT apigroup:%s pods:%+v", input.Apigroup, input.Pods)
		wc.apigroupToPodMap[input.Apigroup] = input
	}

	// Resolve POD to IP
	atopipmap := ApigroupToPodIPMap{
		Apigroup: input.Apigroup,
		PodIPs:   make([]string, 0),
	}
	for _, pod := range input.Pods {
		podip := ""
		for _, podInfo := range wc.clusterPodMap {
			var exists bool
			if podip, exists = podInfo.PodToIPMap[pod]; exists {
				break
			}
		}
		if podip != "" {
			log.Info().Msgf("[UpdateApigroupMap] RESOLVE pod:%s IP:%s", pod, podip)
			atopipmap.PodIPs = append(atopipmap.PodIPs, podip)
		} else {
			log.Info().Msgf("[UpdateApigroupMap] CANNOT RESOLVE pod:%s !!", pod)
		}
	}
	wc.apigroupToPodIPMap[atopipmap.Apigroup] = atopipmap
	wc.updateEnvoy()
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
