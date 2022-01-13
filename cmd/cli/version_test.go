package main

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
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
			proxyGetMeshVersionErr: errors.Errorf("Error retrieving mesh version from pod [controllerPod] in namespace [test]"),
			expectedErr:            errors.Errorf("Error retrieving mesh version from pod [controllerPod] in namespace [test]"),
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
				meshName: "osm",
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

func TestOutputVersionInfo(t *testing.T) {
	tests := []struct {
		name        string
		versionInfo versionInfo
		expected    string
		clientOnly  bool
	}{
		{
			name: "cli and mesh versions with no control plane installed",
			versionInfo: versionInfo{
				cliVersionInfo: &version.Info{
					Version:   "v0.0.0",
					GitCommit: "xxxxxxx",
					BuildDate: "0000-00-00-00:00",
				},
				remoteVersionInfo: &remoteVersionInfo{},
			},
			expected:   "CLI Version: version.Info{Version:\"v0.0.0\", GitCommit:\"xxxxxxx\", BuildDate:\"0000-00-00-00:00\"}\nMesh Version: No control plane found in namespace [test]\n",
			clientOnly: false,
		},
		{
			name: "cli version only",
			versionInfo: versionInfo{
				cliVersionInfo: &version.Info{
					Version:   "v0.0.0",
					GitCommit: "xxxxxxx",
					BuildDate: "0000-00-00-00:00",
				},
			},
			expected:   "CLI Version: version.Info{Version:\"v0.0.0\", GitCommit:\"xxxxxxx\", BuildDate:\"0000-00-00-00:00\"}\n",
			clientOnly: true,
		},
		{
			name: "cli and mesh versions with control plane installed",
			versionInfo: versionInfo{
				cliVersionInfo: &version.Info{
					Version:   "v0.0.0",
					GitCommit: "xxxxxxx",
					BuildDate: "0000-00-00-00:00",
				},
				remoteVersionInfo: &remoteVersionInfo{
					meshName: "test",
					version: &version.Info{
						Version:   "v0.0.0",
						GitCommit: "xxxxxxx",
						BuildDate: "0000-00-00-00:00",
					},
				},
			},
			expected:   "CLI Version: version.Info{Version:\"v0.0.0\", GitCommit:\"xxxxxxx\", BuildDate:\"0000-00-00-00:00\"}\nMesh [test] Version: version.Info{Version:\"v0.0.0\", GitCommit:\"xxxxxxx\", BuildDate:\"0000-00-00-00:00\"}\n",
			clientOnly: false,
		},
		{
			name: "cli and mesh versions with no remote version info",
			versionInfo: versionInfo{
				cliVersionInfo: &version.Info{
					Version:   "v0.0.0",
					GitCommit: "xxxxxxx",
					BuildDate: "0000-00-00-00:00",
				},
				remoteVersionInfo: nil,
			},
			expected:   "CLI Version: version.Info{Version:\"v0.0.0\", GitCommit:\"xxxxxxx\", BuildDate:\"0000-00-00-00:00\"}\n",
			clientOnly: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)

			buf := bytes.NewBuffer(nil)
			cmd := versionCmd{
				out:        buf,
				clientOnly: test.clientOnly,
				namespace:  "test",
			}
			cmd.outputVersionInfo(test.versionInfo)

			assert.Equal(test.expected, buf.String())
		})
	}
}
