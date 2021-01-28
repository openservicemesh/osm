module github.com/openservicemesh/osm

go 1.15

require (
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/go-autorest/autorest v0.10.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/axw/gocov v1.0.0
	github.com/cskr/pubsub v1.0.2
	github.com/deckarep/golang-set v1.7.1
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/dustin/go-humanize v1.0.0
	github.com/envoyproxy/go-control-plane v0.9.8
	github.com/fatih/color v1.10.0
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.4.3
	github.com/golangci/golangci-lint v1.32.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-version v1.1.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/jetstack/cert-manager v0.16.1
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/jstemmer/go-junit-report v0.0.0-20190106144839-af01ea7f8024
	github.com/matm/gocov-html v0.0.0-20200509184451-71874e2e203b
	github.com/mitchellh/gox v1.0.1
	github.com/norwoodj/helm-docs v1.4.0
	github.com/nxadm/tail v1.4.5 // indirect
	github.com/olekukonko/tablewriter v0.0.2
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/common v0.4.1
	github.com/rs/zerolog v1.18.0
	github.com/servicemeshinterface/smi-sdk-go v0.5.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20201112073958-5cba982894dd // indirect
	golang.org/x/tools v0.0.0-20201021214918-23787c007979 // indirect
	gomodules.xyz/jsonpatch/v2 v2.0.1
	google.golang.org/grpc v1.27.0
	google.golang.org/protobuf v1.23.0
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.2.0
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/cli-runtime v0.18.5
	k8s.io/client-go v0.18.8
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/kind v0.9.0
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible
