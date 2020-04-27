package LogAggregation

import (
	"context"
	"encoding/json"
	"io"

	envoy_service_accesslog_v2 "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/utils"
)

const (
	serverType = "LogAggregationService"
)

type server struct{}

func Start(port int, cert certificate.Certificater, stop chan struct{}) {
	glog.Infof("[%s] Instantiating new LogAggregation server...", serverType)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	glog.Infof("[%s] Starting...", serverType)
	grpcServer, listener := utils.NewGrpc(serverType, port, cert.GetCertificateChain(), cert.GetPrivateKey(), cert.GetIssuingCA())
	envoy_service_accesslog_v2.RegisterAccessLogServiceServer(grpcServer, &server{})
	go utils.GrpcServe(ctx, grpcServer, listener, cancel, serverType)
	<-stop
}

func (s *server) StreamAccessLogs(stream envoy_service_accesslog_v2.AccessLogService_StreamAccessLogsServer) error {
	// When a new Envoy proxy connects, ValidateClient would ensure that it has a valid certificate,
	// and the Subject CN is in the allowedCommonNames set.
	cn, err := utils.ValidateClient(stream.Context(), nil)
	if err != nil {
		glog.Errorf("[%s] Could not start stream for %s: %s", serverType, cn, err)
		return err
	}

	glog.Infof("[%s] Started LogAggregation Service stream for %s", serverType, cn)
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
			return errors.New("empty EnvoyLogAggregation message")
		}
		jsonLogs, err := json.Marshal(msg.LogEntries)
		if err != nil {
			glog.Errorf("[%s] Failed marshaling EnvoyLogAggregation: %s", serverType, err)
			return nil
		}
		glog.Infof("[%s] CN=%s count=%d logs=%s", serverType, cn, len(msg.GetHttpLogs().LogEntry), string(jsonLogs))
	}
}
