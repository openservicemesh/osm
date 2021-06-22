package debugger

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type portForward struct {
	// Pod for which port forwarding is done.
	Pod *v1.Pod

	// LocalPort is port on local host, which will be used.
	LocalPort int

	// PodPort is the port on the target Pod, which will be forwarded.
	PodPort int

	// Stop is a channel managing the port forward lifecycle.
	Stop chan struct{}

	// Ready is a channel informing us when the tunnel is ready.
	Ready chan struct{}
}

func (ds DebugConfig) forwardPort(req portForward) {
	log.Debug().Msgf("Start port forward to pod with UID=%s on PodPort=%d to LocalPort=%d", req.Pod.ObjectMeta.UID, req.PodPort, req.LocalPort)
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", req.Pod.Namespace, req.Pod.Name)
	hostIP := strings.TrimLeft(ds.kubeConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(ds.kubeConfig)
	if err != nil {
		log.Error().Err(err)
	}

	client := &http.Client{Transport: transport}
	u := &url.URL{Scheme: "https", Path: path, Host: hostIP}
	streams := genericclioptions.IOStreams{}
	fw, err := portforward.New(
		spdy.NewDialer(upgrader, client, http.MethodPost, u),
		[]string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)},
		req.Stop,
		req.Ready,
		streams.Out,
		streams.ErrOut,
	)
	if err != nil {
		log.Error().Err(err)
	}

	if err = fw.ForwardPorts(); err != nil {
		log.Error().Err(err)
	}
}
