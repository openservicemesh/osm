package tester

import (
	"fmt"
	"io/ioutil"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func LoadService() []runtime.Object {
	return LoadKubernetesResource("service.yaml")
}

// LoadKubernetesResource loads a Kubernetes YAML file.
func LoadKubernetesResource(filenames ...string) []runtime.Object {
	retVal := make([]runtime.Object, 0, len(filenames))

	for _, filename := range filenames {
		if filename == "\n" || filename == "" {
			continue
		}
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(fmt.Sprintf("Could not load %s: %s", filename, err))
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, groupVersionKind, err := decode([]byte(content), nil, nil)
		glog.Info("[tester] Loaded: %s", groupVersionKind)

		if err != nil {
			panic(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
		}
		retVal = append(retVal, obj)
	}

	return retVal
}
