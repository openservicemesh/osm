package injector

const (
	livenessCluster  = "liveness_cluster"
	readinessCluster = "readiness_cluster"
	startupCluster   = "startup_cluster"

	livenessListener  = "liveness_listener"
	readinessListener = "readiness_listener"
	startupListener   = "startup_listener"
)

func getLivenessCluster(originalProbe *healthProbe) map[string]interface{} {
	if originalProbe == nil {
		return nil
	}
	return getProbeCluster(livenessCluster, originalProbe.port)
}

func getReadinessCluster(originalProbe *healthProbe) map[string]interface{} {
	if originalProbe == nil {
		return nil
	}
	return getProbeCluster(readinessCluster, originalProbe.port)
}

func getStartupCluster(originalProbe *healthProbe) map[string]interface{} {
	if originalProbe == nil {
		return nil
	}
	return getProbeCluster(startupCluster, originalProbe.port)
}

func getProbeCluster(clusterName string, port int32) map[string]interface{} {
	return map[string]interface{}{
		"name":            clusterName,
		"connect_timeout": "1s",
		"type":            "STATIC",
		"lb_policy":       "ROUND_ROBIN",
		"load_assignment": map[string]interface{}{
			"cluster_name": clusterName,
			"endpoints": []map[string]interface{}{
				{
					"lb_endpoints": []map[string]interface{}{
						{
							"endpoint": map[string]interface{}{
								"address": map[string]interface{}{
									"socket_address": map[string]interface{}{
										"address":    "0.0.0.0",
										"port_value": port,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func getLivenessListener(originalProbe *healthProbe) map[string]interface{} {
	if originalProbe == nil {
		return nil
	}
	return getProbeListener(livenessListener, livenessCluster, livenessProbePath, livenessProbePort, originalProbe)
}

func getReadinessListener(originalProbe *healthProbe) map[string]interface{} {
	if originalProbe == nil {
		return nil
	}
	return getProbeListener(readinessListener, readinessCluster, readinessProbePath, readinessProbePort, originalProbe)
}

func getStartupListener(originalProbe *healthProbe) map[string]interface{} {
	if originalProbe == nil {
		return nil
	}
	return getProbeListener(startupListener, startupCluster, startupProbePath, startupProbePort, originalProbe)
}

func getProbeListener(listenerName, clusterName, newPath string, port int32, originalProbe *healthProbe) map[string]interface{} {
	return map[string]interface{}{
		"name": listenerName,
		"address": map[string]interface{}{
			"socket_address": map[string]interface{}{
				"address":    "0.0.0.0",
				"port_value": port,
			},
		},
		"filter_chains": []map[string]interface{}{
			{
				"filters": []map[string]interface{}{
					{
						"name": "envoy.filters.network.http_connection_manager",
						"typed_config": map[string]interface{}{
							"@type":       "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
							"stat_prefix": "health_probes_http",
							"access_log":  getAccessLog(),
							"codec_type":  "AUTO",
							"route_config": map[string]interface{}{
								"name":          "local_route",
								"virtual_hosts": getVirtualHosts(newPath, clusterName, originalProbe.path),
							},
							"http_filters": []map[string]interface{}{
								{
									"name": "envoy.filters.http.router",
								},
							},
						},
					},
				},
			},
		},
	}
}

func getVirtualHosts(newPath, clusterName, originalProbePath string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name": "local_service",
			"domains": []string{
				"*",
			},
			"routes": []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"prefix": newPath,
					},
					"route": map[string]interface{}{
						"cluster":        clusterName,
						"prefix_rewrite": originalProbePath,
					},
				},
			},
		},
	}
}

func getAccessLog() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name": "envoy.access_loggers.file",
			"typed_config": map[string]interface{}{
				"@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
				"path":  "/dev/stdout",
				"log_format": map[string]interface{}{
					"json_format": map[string]interface{}{
						"requested_server_name": "%REQUESTED_SERVER_NAME%",
						"method":                "%REQ(:METHOD)%",
						"upstream_service_time": "%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%",
						"upstream_cluster":      "%UPSTREAM_CLUSTER%",
						"protocol":              "%PROTOCOL%",
						"response_code":         "%RESPONSE_CODE%",
						"time_to_first_byte":    "%RESPONSE_DURATION%",
						"response_flags":        "%RESPONSE_FLAGS%",
						"bytes_received":        "%BYTES_RECEIVED%",
						"duration":              "%DURATION%",
						"request_id":            "%REQ(X-REQUEST-ID)%",
						"upstream_host":         "%UPSTREAM_HOST%",
						"path":                  "%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%",
						"response_code_details": "%RESPONSE_CODE_DETAILS%",
						"x_forwarded_for":       "%REQ(X-FORWARDED-FOR)%",
						"user_agent":            "%REQ(USER-AGENT)%",
						"authority":             "%REQ(:AUTHORITY)%",
						"start_time":            "%START_TIME%",
						"bytes_sent":            "%BYTES_SENT%",
					},
				},
			},
		},
	}
}
