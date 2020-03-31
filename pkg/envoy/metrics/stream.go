package metrics

import (
	"context"
	"encoding/json"
	"io"

	v2 "github.com/envoyproxy/go-control-plane/envoy/service/metrics/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/utils"
)

const (
	serverType = "MetricsService"
)

type server struct{}

func Start(port int, certPem *string, keyPem *string, rootCertPem *string, stop chan struct{}) {
	glog.Infof("[%s] Instantiating new Metrics server...", serverType)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	glog.Infof("[%s] Starting...", serverType)
	grpcServer, listener := utils.NewGrpc(serverType, port, *certPem, *keyPem, *rootCertPem)
	v2.RegisterMetricsServiceServer(grpcServer, &server{})
	go utils.GrpcServe(ctx, grpcServer, listener, cancel, serverType)
	<-stop
}

func (s *server) StreamMetrics(stream v2.MetricsService_StreamMetricsServer) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(stream.Context(), nil, serverType)
	if err != nil {
		glog.Errorf("[%s] Could not start stream for %s: %s", serverType, cn, err)
		return err
	}

	glog.Infof("[%s] Started Metrics Service stream for %s", serverType, cn)
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			glog.Errorf("[%s] Error: %s", serverType, err)
			return err
		}
		if err != nil {
			glog.Errorf("[%s] Error: %s", serverType, err)
			return err
		}
		if msg == nil {
			return errors.New("empty EnvoyMetrics message")
		}
		jsonMetrics, err := json.Marshal(msg.EnvoyMetrics)
		if err != nil {
			glog.Error("Failed marshaling EnvoyMetrics: ", err)
			return nil
		}
		glog.Infof("[%s] CN=%s metrics_count=%d metrics=%s", serverType, cn, len(msg.EnvoyMetrics), string(jsonMetrics))
	}
}
