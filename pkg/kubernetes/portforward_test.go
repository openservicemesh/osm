package kubernetes

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/httpstream"
)

type fakeDialer struct {
	conn    httpstream.Connection
	dialErr error
}

func (d *fakeDialer) Dial(protocols ...string) (httpstream.Connection, string, error) {
	return d.conn, "", d.dialErr
}

type noopConnection struct{}

func (*noopConnection) CreateStream(headers http.Header) (httpstream.Stream, error) {
	return nil, nil
}
func (*noopConnection) CloseChan() <-chan bool       { return nil }
func (*noopConnection) Close() error                 { return nil }
func (*noopConnection) SetIdleTimeout(time.Duration) {}

func TestPortForwardSuccess(t *testing.T) {
	dialer := &fakeDialer{
		conn: &noopConnection{},
	}

	pf, err := NewPortForwarder(dialer, ":80")
	if err != nil {
		t.Fatal("error creating PortForwarder:", err)
	}

	err = pf.Start(func(*PortForwarder) error {
		return nil
	})
	if err != nil {
		t.Error("error running port forward:", err)
	}
	pf.Stop()
}

func TestPortForwardInvalidPortSpec(t *testing.T) {
	portSpec := ""
	pf, err := NewPortForwarder(nil, "")
	if err == nil {
		t.Errorf("Expected error for port spec %q, got none", portSpec)
	}
	if pf != nil {
		t.Errorf("Expected PortForwarder to be nil, got %+v", pf)
	}
}

func TestPortForwardDialError(t *testing.T) {
	dialer := &fakeDialer{
		dialErr: errors.New("some error"),
	}
	pf, err := NewPortForwarder(dialer, ":80")
	if err != nil {
		t.Fatal("error creating PortForwarder:", err)
	}

	err = pf.Start(func(*PortForwarder) error {
		t.Error("Expected PortForwarder not to become ready but it did")
		return nil
	})

	if err == nil {
		t.Fatal("Expected running port forward to fail but it succeeded")
	}
	if !strings.Contains(err.Error(), dialer.dialErr.Error()) {
		t.Errorf("Expected error matching %q, got %q", dialer.dialErr, err)
	}
}

func TestPortForwardStoppedBySignal(t *testing.T) {
	dialer := &fakeDialer{
		conn: &noopConnection{},
	}

	pf, err := NewPortForwarder(dialer, ":80")
	if err != nil {
		t.Fatal("error creating PortForwarder:", err)
	}

	err = pf.Start(func(*PortForwarder) error {
		return nil
	})
	if err != nil {
		t.Error("error running port forward:", err)
	}

	// No Stop() needed when responding to SIGINT
	pf.sigChan <- os.Interrupt
	<-pf.Done()
}
