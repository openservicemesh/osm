module github.com/open-service-mesh/osm

go 1.13

require (
	contrib.go.opencensus.io/exporter/ocagent v0.5.0 // indirect
	git.apache.org/thrift.git v0.0.0-20180902110319-2566ecd5d999 // indirect
	github.com/Azure/azure-sdk-for-go v34.0.0+incompatible
	github.com/Azure/go-autorest v12.1.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.10.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0
	github.com/Azure/go-autorest/autorest/to v0.2.0
	github.com/Azure/go-autorest/autorest/validation v0.1.0 // indirect
	github.com/axw/gocov v1.0.0 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/eapache/channels v1.1.0 // indirect
	github.com/envoyproxy/go-control-plane v0.9.2
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/getlantern/deepcopy v0.0.0-20160317154340-7f45deb8130a // indirect
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48 // indirect
	github.com/golang/lint v0.0.0-20180702182130-06c8688daad7 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/grpc-ecosystem/grpc-gateway v1.9.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/knative/pkg v0.0.0-20190619032946-d90a9bc97dde // indirect
	github.com/matm/gocov-html v0.0.0-20160206185555-f6dd0fd0ebc7 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829
	github.com/rs/zerolog v1.18.0
	github.com/servicemeshinterface/smi-sdk-go v0.3.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	go.opencensus.io v0.22.1 // indirect
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	google.golang.org/api v0.10.0 // indirect
	google.golang.org/appengine v1.6.2 // indirect
	google.golang.org/genproto v0.0.0-20190911173649-1774047e7e51 // indirect
	google.golang.org/grpc v1.25.1
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.18.0
	k8s.io/utils v0.0.0-20190607212802-c55fbcfc754a // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190612125919-5c45477a8ae7
