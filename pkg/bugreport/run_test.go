// +build withEnvPathSet

package bugreport

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestRun(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	fakeClient := fake.NewSimpleClientset()

	outFile, err := ioutil.TempFile("", "*_osm-bug-report.zip")
	r.Nil(err)

	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns1",
			Labels: map[string]string{constants.OSMKubeResourceMonitorAnnotation: "osm"},
		},
	}
	_, err = fakeClient.CoreV1().Namespaces().Create(context.TODO(), ns1, metav1.CreateOptions{})
	r.Nil(err)

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
			Labels:    map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
		},
	}
	_, err = fakeClient.CoreV1().Pods(pod1.Namespace).Create(context.TODO(), pod1, metav1.CreateOptions{})
	r.Nil(err)

	c := &Config{
		Stdout:        new(bytes.Buffer),
		Stderr:        new(bytes.Buffer),
		KubeClient:    fakeClient,
		AppNamespaces: []string{ns1.Name},
		AppPods:       []types.NamespacedName{{Name: pod1.Name, Namespace: pod1.Namespace}},
		OutFile:       outFile.Name(),
	}

	err = c.Run()
	a.Nil(err)

	err = os.RemoveAll(outFile.Name())
	r.Nil(err)
}
