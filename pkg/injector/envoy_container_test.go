package injector

import (
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

func TestGetPlatformSpecificSpecComponents(t *testing.T) {
	const (
		windowsImage = "windowsImage"
		linuxImage   = "linuxImage"
	)
	type args struct {
		podOS string
	}
	tests := []struct {
		name                   string
		args                   args
		wantPodSecurityContext *corev1.SecurityContext
		wantEnvoyContainer     string
	}{
		{
			name: "success: windows",
			args: args{
				podOS: constants.OSWindows,
			},
			wantPodSecurityContext: &corev1.SecurityContext{
				WindowsOptions: &corev1.WindowsSecurityContextOptions{
					RunAsUserName: func() *string {
						userName := constants.EnvoyWindowsUser
						return &userName
					}(),
				},
			},
			wantEnvoyContainer: windowsImage,
		},
		{
			name: "success: linux",
			args: args{
				podOS: constants.OSLinux,
			},
			wantPodSecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: pointer.BoolPtr(false),
				RunAsUser: func() *int64 {
					uid := constants.EnvoyUID
					return &uid
				}(),
			},
			wantEnvoyContainer: linuxImage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			mockCfg := configurator.NewMockConfigurator(mockCtrl)
			if tt.args.podOS == constants.OSWindows {
				mockCfg.EXPECT().GetEnvoyWindowsImage().Return(windowsImage)
			} else {
				mockCfg.EXPECT().GetEnvoyImage().Return(linuxImage)
			}

			defer mockCtrl.Finish()
			gotPodSecurityContext, gotEnvoyContainer := getPlatformSpecificSpecComponents(mockCfg, tt.args.podOS)

			assert.Equal(tt.wantPodSecurityContext, gotPodSecurityContext)
			assert.Equal(tt.wantEnvoyContainer, gotEnvoyContainer)
		})
	}
}
