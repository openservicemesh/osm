module github.com/openservicemesh/osm

go 1.15

require (
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/azure-sdk-for-go v34.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.10.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/axw/gocov v1.0.0
	github.com/cncf/udpa/go v0.0.0-20200629203442-efcf912fb354 // indirect
	github.com/deckarep/golang-set v1.7.1
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/envoyproxy/go-control-plane v0.9.6
	github.com/fatih/color v1.9.0
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/golangci/golangci-lint v1.30.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/vault/api v1.0.4
	github.com/jetstack/cert-manager v0.16.1
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/jstemmer/go-junit-report v0.0.0-20190106144839-af01ea7f8024
	github.com/matm/gocov-html v0.0.0-20200509184451-71874e2e203b
	github.com/mitchellh/gox v1.0.1
	github.com/onsi/ginkgo v1.13.0
	github.com/onsi/gomega v1.10.1
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/rs/zerolog v1.18.0
	github.com/servicemeshinterface/smi-sdk-go v0.4.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/tools v0.0.0-20200911183043-b43031a33b24 // indirect
	google.golang.org/grpc v1.27.0
	google.golang.org/protobuf v1.23.0
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.2.0
	k8s.io/api v0.18.5
	k8s.io/apimachinery v0.18.8
	k8s.io/cli-runtime v0.18.5
	k8s.io/client-go v0.18.5
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/kind v0.9.0
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
