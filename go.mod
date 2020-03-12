module github.com/open-service-mesh/osm

go 1.13

require (
	9fans.net/go v0.0.2 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.0 // indirect
	github.com/Azure/application-gateway-kubernetes-ingress v0.0.0-20190820222238-dfce74a2549f
	github.com/Azure/azure-sdk-for-go v34.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.5.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/Azure/go-autorest/tracing v0.2.0 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/deislabs/smi-sdk-go v0.2.0
	github.com/eapache/channels v1.1.0
	github.com/envoyproxy/go-control-plane v0.9.2
	github.com/fsnotify/fsnotify v1.4.7
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.0.0
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/mitchellh/gox v1.0.1 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/rogpeppe/godef v1.1.1 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50 // indirect
	golang.org/x/tools v0.0.0-20200115222509-97cd989a7672 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	google.golang.org/grpc v1.25.1
	gopkg.in/yaml.v2 v2.2.7 // indirect
	istio.io/gogo-genproto v0.0.0-20190731221249-06e20ada0df2 // indirect
	k8s.io/api v0.0.0-20190614205929-e4e27c96b39a
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190612125919-5c45477a8ae7
