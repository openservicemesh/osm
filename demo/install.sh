  bin/osm install \
      --osm-namespace "$K8S_NAMESPACE" \
      --mesh-name "$MESH_NAME" \
      --set=OpenServiceMesh.certificateManager="$CERT_MANAGER" \
      --set=OpenServiceMesh.image.registry="$CTR_REGISTRY" \
      --set=OpenServiceMesh.imagePullSecrets[0].name="$CTR_REGISTRY_CREDS_NAME" \
      --set=OpenServiceMesh.image.tag="$CTR_TAG" \
      --set=OpenServiceMesh.image.pullPolicy="$IMAGE_PULL_POLICY" \
      --set=OpenServiceMesh.enableDebugServer="$ENABLE_DEBUG_SERVER" \
      --set=OpenServiceMesh.enableEgress="$ENABLE_EGRESS" \
      --set=OpenServiceMesh.deployGrafana="$DEPLOY_GRAFANA" \
      --set=OpenServiceMesh.deployJaeger="$DEPLOY_JAEGER" \
      --set=OpenServiceMesh.enableFluentbit="$ENABLE_FLUENTBIT" \
      --set=OpenServiceMesh.deployPrometheus="$DEPLOY_PROMETHEUS" \
      --set=OpenServiceMesh.envoyLogLevel="$ENVOY_LOG_LEVEL" \
      --set=OpenServiceMesh.controllerLogLevel="trace" \
      --set=OpenServiceMesh.featureFlags.enableMulticlusterMode="$ENABLE_MULTI_CLUSTER_MODE" \
      --set=OpenServiceMesh.featureFlags.enableOSMGateway="$ENABLE_MULTI_CLUSTER_MODE" \
      --timeout=360s \
      $optionalInstallArgs