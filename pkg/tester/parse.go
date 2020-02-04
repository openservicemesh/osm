package tester

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func LoadService() []runtime.Object {
	return LoadKubernetesResource("../../pkg/tester/fixtures/service.yaml")
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
			workingDir, _ := os.Getwd()
			panic(fmt.Sprintf("[tester] Could not load %s (cwd=%s): %s", filename, workingDir, err))
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode

		obj, groupVersionKind, err := decode(content, nil, nil)
		if err != nil {
			glog.Fatalf("[tester] Could not decode object %s: %s", filename, err)
		}
		glog.Infof("[tester] Loaded: %s %s", groupVersionKind, obj)

		if err != nil {
			panic(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
		}
		retVal = append(retVal, obj)
	}

	return retVal
}
