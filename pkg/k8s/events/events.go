package events

import (
	"context"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// EventRecorder is a type used to record Kubernetes events
type EventRecorder struct {
	recorder record.EventRecorder
	object   runtime.Object
	watcher  watch.Interface
}

var (
	once                 sync.Once
	genericEventRecorder *EventRecorder
)

const (
	// eventSource is the name of the event source which generates Kubernetes events
	eventSource = "osm-controller"

	// fatalEventPrefix is the prefix used with fatal event reasons which is used to identify Fatal events.
	fatalEventPrefix = "Fatal"

	// fatalEventWaitTimeout is the timeout period for waiting on a fatal event
	fatalEventWaitTimeout = 10 * time.Second
)

// NewEventRecorder returns a new EventRecorder object and an error in case of errors
func NewEventRecorder(object runtime.Object, kubeClient kubernetes.Interface, namespace string) (*EventRecorder, error) {
	recorder := eventRecorder(kubeClient, namespace)
	watcher, err := eventWatcher(kubeClient, namespace)

	if err != nil {
		log.Error().Err(err).Msg("Error initializing event watcher")
		return nil, err
	}

	return &EventRecorder{
		recorder: recorder,
		watcher:  watcher,
		object:   object,
	}, nil
}

// GenericEventRecorder is a singleton that returns a generic EventRecorder type.
// The EventRecorder returned needs to be explicitly initialized by calling the 'Initialize' method on the object
func GenericEventRecorder() *EventRecorder {
	once.Do(func() {
		genericEventRecorder = &EventRecorder{}
	})

	return genericEventRecorder
}

// eventRecorder returns an EventRecorder that can be used to post Kubernetes events
func eventRecorder(kubeClient kubernetes.Interface, namespace string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: kubeClient.CoreV1().Events(namespace)})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: eventSource})

	return recorder
}

// eventWatcher returns a Kubernetes watch interface to watch events, and an error in case of errors
func eventWatcher(kubeClient kubernetes.Interface, namespace string) (watch.Interface, error) {
	watcher, err := kubeClient.CoreV1().Events(namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msg("Error getting watcher for Events resource")
		return nil, err
	}
	return watcher, nil
}

// Initialize initializes an uninitialized EventRecorder object
func (e *EventRecorder) Initialize(object runtime.Object, kubeClient kubernetes.Interface, namespace string) error {
	var err error
	e.object = object
	e.recorder = eventRecorder(kubeClient, namespace)
	e.watcher, err = eventWatcher(kubeClient, namespace)

	return err
}

// recordEvent records a Kubernetes event. Kubernetes events are non-blocking.
func (e *EventRecorder) recordEvent(eventType string, reason string, messageFmt string, args ...interface{}) {
	if e.recorder == nil || e.object == nil {
		// This is a safety check to prevent bugs from creeping in and should
		// never be seen. Without this, buggy code will panic down the call stack.
		// This is used to catch missing initialization of the singleton 'GenericEventRecorder'.
		log.Warn().Msg("EventRecorder is uninitialized")
		return
	}
	e.recorder.Eventf(e.object, eventType, reason, messageFmt, args...)
}

// NormalEvent records a Normal Kubernetes event
func (e *EventRecorder) NormalEvent(reason string, messageFmt string, args ...interface{}) {
	e.recordEvent(corev1.EventTypeNormal, reason, messageFmt, args...)
	log.Info().Str("reason", reason).Msgf(messageFmt, args...)
}

// WarnEvent records a Warning Kubernetes event
func (e *EventRecorder) WarnEvent(reason string, messageFmt string, args ...interface{}) {
	e.recordEvent(corev1.EventTypeWarning, reason, messageFmt, args...)
	log.Warn().Str("reason", reason).Msgf(messageFmt, args...)
}

// ErrorEvent records a Warning Kubernetes event
func (e *EventRecorder) ErrorEvent(err error, reason string, messageFmt string, args ...interface{}) {
	e.recordEvent(corev1.EventTypeWarning /* most severe type */, reason, messageFmt, args...)
	log.Error().Err(err).Str("reason", reason).Msgf(messageFmt, args...)
}

// FatalEvent records a Warning Kubernetes event
func (e *EventRecorder) FatalEvent(err error, reason string, messageFmt string, args ...interface{}) {
	e.recordEvent(corev1.EventTypeWarning /* most severe type */, reason, messageFmt, args...)

	// Wait for the event to be recorded before exiting via 'log.Fatal()'
	e.waitForFatalEvent()
	log.Fatal().Err(err).Str("reason", reason).Msgf(messageFmt, args...)
}

// waitForFatalEvent waits until a fatal event has been seen before a wait timeout
func (e *EventRecorder) waitForFatalEvent() {
	timeout := time.After(fatalEventWaitTimeout)

WaitOnEvent:
	for {
		select {
		case <-timeout:
			break WaitOnEvent // Timed out

		case watchedEvent, ok := <-e.watcher.ResultChan():
			if !ok {
				break WaitOnEvent // Channel closed
			}
			// If the event is fatal, finish waiting by returning
			event := watchedEvent.Object.(*corev1.Event)
			if strings.HasPrefix(event.Reason, fatalEventPrefix) {
				return
			}
		}
	}
}
