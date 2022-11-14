package lds

import (
	"encoding/json"
	"fmt"
	"strings"

	xds_accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	xds_core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	xds_grpc_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/grpc/v3"
	xds_otel_accesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/open_telemetry/v3"
	xds_accesslog_stream "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	xds_formatter "github.com/envoyproxy/go-control-plane/envoy/extensions/formatter/req_without_query/v3"
	otlpCommon "go.opentelemetry.io/proto/otlp/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/protobuf"
)

const (
	reqWithoutQuery = "%REQ_WITHOUT_QUERY"
)

var (
	defaultAccessLogJSONFields = &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"start_time":            structpb.NewStringValue(`%START_TIME%`),
			"method":                structpb.NewStringValue(`%REQ(:METHOD)%`),
			"path":                  structpb.NewStringValue(`%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%`),
			"protocol":              structpb.NewStringValue(`%PROTOCOL%`),
			"response_code":         structpb.NewStringValue(`%RESPONSE_CODE%`),
			"response_code_details": structpb.NewStringValue(`%RESPONSE_CODE_DETAILS%`),
			"time_to_first_byte":    structpb.NewStringValue(`%RESPONSE_DURATION%`),
			"upstream_cluster":      structpb.NewStringValue(`%UPSTREAM_CLUSTER%`),
			"response_flags":        structpb.NewStringValue(`%RESPONSE_FLAGS%`),
			"bytes_received":        structpb.NewStringValue(`%BYTES_RECEIVED%`),
			"bytes_sent":            structpb.NewStringValue(`%BYTES_SENT%`),
			"duration":              structpb.NewStringValue(`%DURATION%`),
			"upstream_service_time": structpb.NewStringValue(`%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%`),
			"x_forwarded_for":       structpb.NewStringValue(`%REQ(X-FORWARDED-FOR)%`),
			"user_agent":            structpb.NewStringValue(`%REQ(USER-AGENT)%`),
			"request_id":            structpb.NewStringValue(`%REQ(X-REQUEST-ID)%`),
			"requested_server_name": structpb.NewStringValue("%REQUESTED_SERVER_NAME%"),
			"authority":             structpb.NewStringValue(`%REQ(:AUTHORITY)%`),
			"upstream_host":         structpb.NewStringValue(`%UPSTREAM_HOST%`),
		},
	}

	defaultAccessLogJSONStr = protobuf.MustToJSON(defaultAccessLogJSONFields)

	accessLogFormatters = []*xds_core.TypedExtensionConfig{
		{
			Name:        "envoy.formatter.req_without_query",
			TypedConfig: protobuf.MustMarshalAny(&xds_formatter.ReqWithoutQuery{}),
		},
	}
)

type accessLogBuilder struct {
	name                    string
	format                  string
	openTelemetryCluster    string
	openTelemetryAttributes map[string]string
}

// NewAccessLogBuilder returns an accessLogBuilder instance used to build access log configuration
func NewAccessLogBuilder() *accessLogBuilder { //nolint: revive // unexported-return
	return &accessLogBuilder{}
}

// Name sets the access log's name
func (ab *accessLogBuilder) Name(name string) *accessLogBuilder {
	ab.name = name
	return ab
}

// Format sets the access log format
func (ab *accessLogBuilder) Format(format string) *accessLogBuilder {
	ab.format = format
	return ab
}

// OpenTelemetryCluster sets the cluster name of the OpenTelemetry collector
func (ab *accessLogBuilder) OpenTelemetryCluster(cluster string) *accessLogBuilder {
	ab.openTelemetryCluster = cluster
	return ab
}

// OpenTelemetryAttributes sets the attributes for the logs exported to the OpenTelemetry collector
func (ab *accessLogBuilder) OpenTelemetryAttributes(attributes map[string]string) *accessLogBuilder {
	ab.openTelemetryAttributes = make(map[string]string, len(attributes))
	for key, val := range attributes {
		ab.openTelemetryAttributes[key] = val
	}
	return ab
}

// Build builds the access log configuration
func (ab *accessLogBuilder) Build() ([]*xds_accesslog.AccessLog, error) {
	var accessLogs []*xds_accesslog.AccessLog
	var stdoutAccessLogConfig *xds_accesslog_stream.StdoutAccessLog
	var accessLogFormat string

	if ab.format == "" {
		stdoutAccessLogConfig = buildStdoutJSONAccessLog(defaultAccessLogJSONFields)
		accessLogFormat = defaultAccessLogJSONStr
		log.Trace().Msg("built default JSON based access log")
	} else {
		accessLogFormat = ab.format
		if isJSONStr(ab.format) {
			accessLogJSONFields := &structpb.Struct{}
			err := protojson.Unmarshal([]byte(ab.format), accessLogJSONFields)
			if err != nil {
				return nil, err
			}
			stdoutAccessLogConfig = buildStdoutJSONAccessLog(accessLogJSONFields)
			log.Trace().Msg("built custom JSON based access log")
		} else {
			stdoutAccessLogConfig = buildStdoutTextAccessLog(ab.format)
			log.Trace().Msg("built custom text based access log")
		}
	}

	log.Trace().Msgf("access log: %v", stdoutAccessLogConfig)
	accessLog, err := anypb.New(stdoutAccessLogConfig)
	if err != nil {
		return nil, err
	}
	accessLogs = append(accessLogs, &xds_accesslog.AccessLog{
		Name: ab.name,
		ConfigType: &xds_accesslog.AccessLog_TypedConfig{
			TypedConfig: accessLog,
		},
	})

	if ab.openTelemetryCluster != "" {
		otelAccessLog := buildOpenTelemetryAccessLogConfig(ab.openTelemetryCluster, accessLogFormat, ab.openTelemetryAttributes)
		accessLogs = append(accessLogs, &xds_accesslog.AccessLog{
			Name: fmt.Sprintf("%s_open_telemetry", ab.name),
			ConfigType: &xds_accesslog.AccessLog_TypedConfig{
				TypedConfig: protobuf.MustMarshalAny(otelAccessLog),
			},
		})
	}

	return accessLogs, nil
}

func buildStdoutJSONAccessLog(format *structpb.Struct) *xds_accesslog_stream.StdoutAccessLog {
	config := &xds_accesslog_stream.StdoutAccessLog{
		AccessLogFormat: &xds_accesslog_stream.StdoutAccessLog_LogFormat{
			LogFormat: &xds_core.SubstitutionFormatString{
				Format: &xds_core.SubstitutionFormatString_JsonFormat{
					JsonFormat: format,
				},
			},
		},
	}

	stripReqQuery := false
	for _, value := range format.Fields {
		if strings.Contains(value.GetStringValue(), reqWithoutQuery) {
			stripReqQuery = true
			break
		}
	}

	if stripReqQuery {
		config.GetLogFormat().Formatters = accessLogFormatters
	}

	return config
}

func buildStdoutTextAccessLog(format string) *xds_accesslog_stream.StdoutAccessLog {
	config := &xds_accesslog_stream.StdoutAccessLog{
		AccessLogFormat: &xds_accesslog_stream.StdoutAccessLog_LogFormat{
			LogFormat: &xds_core.SubstitutionFormatString{
				Format: &xds_core.SubstitutionFormatString_TextFormatSource{
					TextFormatSource: &xds_core.DataSource{
						Specifier: &xds_core.DataSource_InlineString{
							InlineString: format,
						},
					},
				},
			},
		},
	}

	stripReqQuery := strings.Contains(format, reqWithoutQuery)
	if stripReqQuery {
		config.GetLogFormat().Formatters = accessLogFormatters
	}

	return config
}

func buildOpenTelemetryAccessLogConfig(cluster string, body string, attributes map[string]string) *xds_otel_accesslog.OpenTelemetryAccessLogConfig {
	config := &xds_otel_accesslog.OpenTelemetryAccessLogConfig{
		CommonConfig: &xds_grpc_accesslog.CommonGrpcAccessLogConfig{
			LogName: cluster,
			GrpcService: &xds_core.GrpcService{
				TargetSpecifier: &xds_core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &xds_core.GrpcService_EnvoyGrpc{
						ClusterName: cluster,
					},
				},
			},
			TransportApiVersion: xds_core.ApiVersion_V3,
		},
		Body: &otlpCommon.AnyValue{
			Value: &otlpCommon.AnyValue_StringValue{
				StringValue: body,
			},
		},
	}

	if attributes != nil {
		config.Attributes = &otlpCommon.KeyValueList{
			Values: getOpenTelemetryAttributes(attributes),
		}
	}

	return config
}

func getOpenTelemetryAttributes(attributes map[string]string) []*otlpCommon.KeyValue {
	if len(attributes) == 0 {
		return nil
	}

	attrs := make([]*otlpCommon.KeyValue, 0, len(attributes))
	for key, value := range attributes {
		kv := &otlpCommon.KeyValue{
			Key:   key,
			Value: &otlpCommon.AnyValue{Value: &otlpCommon.AnyValue_StringValue{StringValue: value}},
		}
		attrs = append(attrs, kv)
	}
	return attrs
}

func isJSONStr(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// BuildAccessLogs builds the access log config from the given telemetry config
func BuildAccessLogs(name string, telemetryConfig models.TelemetryConfig) ([]*xds_accesslog.AccessLog, error) {
	ab := NewAccessLogBuilder().Name(name)

	if telemetryConfig.Policy != nil {
		ab.Format(telemetryConfig.Policy.Spec.AccessLog.Format)

		if telemetryConfig.OpenTelemetryService != nil {
			ab.OpenTelemetryCluster(fmt.Sprintf("%s.%d", telemetryConfig.OpenTelemetryService.Spec.Host, telemetryConfig.OpenTelemetryService.Spec.Port)).
				OpenTelemetryAttributes(telemetryConfig.Policy.Spec.AccessLog.OpenTelemetry.Attributes)
		}
	}

	return ab.Build()
}
