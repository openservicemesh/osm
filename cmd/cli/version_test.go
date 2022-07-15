package main

import (
	"bytes"
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/version"
)

type fakeRemoteVersion struct {
	err         error
	versionInfo *version.Info
}

func (f *fakeRemoteVersion) proxyGetMeshVersion(pod string, namespace string, clientset kubernetes.Interface) (*version.Info, error) {
	return f.versionInfo, f.err
}

func TestGetMeshVersion(t *testing.T) {
	tests := []struct {
		name                   string
		namespace              string
		remoteVersion          *version.Info
		remoteVersionInfo      *remoteVersionInfo
		controllerPods         []*corev1.Pod
		proxyGetMeshVersionErr error
		expectedErr            error
	}{
		{
			name:                   "no mesh in namespace",
			namespace:              "test",
			remoteVersion:          nil,
			remoteVersionInfo:      &remoteVersionInfo{},
			controllerPods:         []*corev1.Pod{},
			proxyGetMeshVersionErr: nil,
			expectedErr:            nil,
		},
		{
			name:              "mesh in namespace and proxyGetMeshVersion fails",
			namespace:         "test",
			remoteVersion:     nil,
			remoteVersionInfo: nil,
			controllerPods: []*corev1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "controllerPod",
						Namespace: "test",
						Labels: map[string]string{
							constants.AppLabel:               constants.OSMControllerName,
							constants.OSMAppInstanceLabelKey: "osm",
						},
					},
				},
			},
			proxyGetMeshVersionErr: fmt.Errorf("Error retrieving mesh version from pod [controllerPod] in namespace [test]"),
			expectedErr:            fmt.Errorf("Error retrieving mesh version from pod [controllerPod] in namespace [test]"),
		},
		{
			name:      "mesh in namespace and remote version found",
			namespace: "test",
			remoteVersion: &version.Info{
				Version:   "v0.0.0",
				GitCommit: "xxxxxxx",
				BuildDate: "Date",
			},
			remoteVersionInfo: &remoteVersionInfo{
				meshName:  "osm",
				namespace: "test",
				version: &version.Info{
					Version:   "v0.0.0",
					GitCommit: "xxxxxxx",
					BuildDate: "Date",
				},
			},
			controllerPods: []*corev1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "controllerPod",
						Namespace: "test",
						Labels: map[string]string{
							constants.AppLabel:               constants.OSMControllerName,
							constants.OSMAppInstanceLabelKey: "osm",
						},
					},
				},
			},
			proxyGetMeshVersionErr: nil,
			expectedErr:            nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)

			buf := bytes.NewBuffer(nil)

			objs := make([]runtime.Object, len(test.controllerPods))
			for i := range test.controllerPods {
				objs[i] = test.controllerPods[i]
			}

			cmd := versionCmd{
				out:       buf,
				namespace: test.namespace,
				clientset: fake.NewSimpleClientset(objs...),
				remoteVersion: &fakeRemoteVersion{
					err:         test.proxyGetMeshVersionErr,
					versionInfo: test.remoteVersion,
				},
				clientOnly: false,
			}

			remoteVersionInfo, err := cmd.getMeshVersion()
			if err != nil && test.expectedErr != nil {
				assert.Equal(test.expectedErr.Error(), err.Error())
			} else {
				assert.Equal(test.expectedErr, err)
			}

			assert.Equal(test.remoteVersionInfo, remoteVersionInfo)
		})
	}
}

func TestOutputPrettyVersionInfo(t *testing.T) {
	tests := []struct {
		name                  string
		remoteVersionInfoList []*remoteVersionInfo
		expected              string
	}{
		{
			name: "mesh versions with one mesh info not found",
			remoteVersionInfoList: []*remoteVersionInfo{
				{
					meshName: "test",
					version: &version.Info{
						Version:   "v0.0.0",
						GitCommit: "xxxxxxx",
						BuildDate: "0000-00-00-00:00",
					},
				},
				{
					meshName: "test2",
					version: &version.Info{
						Version:   "v0.0.0",
						GitCommit: "aaaaaaa",
						BuildDate: "0000-00-00-00:00",
					},
				},
				{
					meshName: "",
					version:  &version.Info{},
				},
			},
			expected: "\nMESH NAME\tMESH NAMESPACE\tVERSION\tGIT COMMIT\tBUILD DATE" +
				"\ntest\t\tv0.0.0\txxxxxxx\t0000-00-00-00:00" +
				"\ntest2\t\tv0.0.0\taaaaaaa\t0000-00-00-00:00\n",
		},
		{
			name: "mesh versions with multiple remote version info",
			remoteVersionInfoList: []*remoteVersionInfo{
				{
					meshName: "test",
					version: &version.Info{
						Version:   "v0.0.0",
						GitCommit: "xxxxxxx",
						BuildDate: "0000-00-00-00:00",
					},
				},
				{
					meshName: "test2",
					version: &version.Info{
						Version:   "v0.0.1",
						GitCommit: "yyyyyyy",
						BuildDate: "0000-00-00-00:00",
					},
				},
				{
					meshName: "test3",
					version: &version.Info{
						Version:   "v0.0.2",
						GitCommit: "xxxxxxy",
						BuildDate: "0000-00-00-00:00",
					},
				},
			},
			expected: "\nMESH NAME\tMESH NAMESPACE\tVERSION\tGIT COMMIT\tBUILD DATE" +
				"\ntest\t\tv0.0.0\txxxxxxx\t0000-00-00-00:00" +
				"\ntest2\t\tv0.0.1\tyyyyyyy\t0000-00-00-00:00" +
				"\ntest3\t\tv0.0.2\txxxxxxy\t0000-00-00-00:00\n",
		},
		{
			name: "mesh versions with control plane installed",
			remoteVersionInfoList: []*remoteVersionInfo{
				{
					meshName: "test",
					version: &version.Info{
						Version:   "v0.0.0",
						GitCommit: "xxxxxxx",
						BuildDate: "0000-00-00-00:00",
					},
				},
			},
			expected: "\nMESH NAME\tMESH NAMESPACE\tVERSION\tGIT COMMIT\tBUILD DATE" +
				"\ntest\t\tv0.0.0\txxxxxxx\t0000-00-00-00:00\n",
		},
		{
			name:                  "mesh versions with no remote version info",
			remoteVersionInfoList: []*remoteVersionInfo{},
			expected:              "Unable to find OSM control plane in the cluster\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)

			cmd := versionCmd{
				namespace: "test",
			}
			tbl := cmd.outputPrettyVersionInfo(test.remoteVersionInfoList)

			assert.Equal(test.expected, tbl)
		})
	}
}
