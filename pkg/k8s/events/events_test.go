package events

import (
	"fmt"
	"os"
	"testing"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	"github.com/stretchr/testify/assert"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func setup() {
	kubeClient := fake.NewSimpleClientset()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "foo",
			UID:       "bar",
		},
	}

	eventRecorder := GenericEventRecorder()
	if err := eventRecorder.Initialize(pod, kubeClient, "test"); err != nil {
		log.Fatal().Err(err).Msg("Error initializing event recorder")
	}
}

func TestGenericEventRecording(t *testing.T) {
	assert := tassert.New(t)

	assert.NotNil(GenericEventRecorder().object)
	assert.NotNil(GenericEventRecorder().recorder)
	assert.NotNil(GenericEventRecorder().watcher)

	events := GenericEventRecorder().watcher.ResultChan()

	GenericEventRecorder().NormalEvent("TestReason", "Test message")
	<-events

	GenericEventRecorder().WarnEvent("TestReason", "Test message")
	<-events

	GenericEventRecorder().ErrorEvent(fmt.Errorf("test"), "TestReason", "Test message")
	<-events
}

func TestSpecificEventRecording(t *testing.T) {
	assert := tassert.New(t)

	kubeClient := fake.NewSimpleClientset()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "foo",
			UID:       "bar",
		},
	}

	eventRecorder, err := NewEventRecorder(pod, kubeClient, "test")

	assert.Nil(err)
	assert.NotNil(eventRecorder.object)
	assert.NotNil(eventRecorder.recorder)
	assert.NotNil(eventRecorder.watcher)

	events := eventRecorder.watcher.ResultChan()

	eventRecorder.NormalEvent("TestReason", "Test message")
	<-events

	eventRecorder.WarnEvent("TestReason", "Test message")
	<-events

	eventRecorder.ErrorEvent(fmt.Errorf("test"), "TestReason", "Test message")
	<-events
}

func TestEventKinds(t *testing.T) {
	testCases := []struct {
		obj          interface{}
		expectedKind Kind
	}{
		{
			obj:          &corev1.Namespace{},
			expectedKind: Namespace,
		},
		{
			obj:          &configv1alpha2.MeshConfig{},
			expectedKind: MeshConfig,
		},
		{
			obj:          &configv1alpha2.MeshRootCertificate{},
			expectedKind: MeshRootCertificate,
		},
		{
			obj:          &policyv1alpha1.Egress{},
			expectedKind: Egress,
		},
		{
			obj:          &policyv1alpha1.IngressBackend{},
			expectedKind: IngressBackend,
		},
		{
			obj:          &policyv1alpha1.UpstreamTrafficSetting{},
			expectedKind: UpstreamTrafficSetting,
		},
		{
			obj:          &policyv1alpha1.Retry{},
			expectedKind: RetryPolicy,
		},
		{
			obj:          &corev1.Pod{},
			expectedKind: Pod,
		},
		{
			obj:          &corev1.ServiceAccount{},
			expectedKind: ServiceAccount,
		},
		{
			obj:          &corev1.Service{},
			expectedKind: Service,
		},
		{
			obj:          &corev1.Endpoints{},
			expectedKind: Endpoint,
		},
		{
			obj:          &smiAccess.TrafficTarget{},
			expectedKind: TrafficTarget,
		},
		{
			obj:          &smiSplit.TrafficSplit{},
			expectedKind: TrafficSplit,
		},
		{
			obj:          &smiSpecs.HTTPRouteGroup{},
			expectedKind: RouteGroup,
		},
		{
			obj:          &smiSpecs.TCPRoute{},
			expectedKind: TCPRoute,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedKind.String(), func(t *testing.T) {
			a := assert.New(t)
			a.Equal(tc.expectedKind, GetKind(tc.obj))
		})
	}
}
