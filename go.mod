module github.com/openservicemesh/osm

go 1.16

require (
	github.com/AlekSi/gocov-xml v0.0.0-20190121064608-3a14fb1c4737
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/axw/gocov v1.0.0
	github.com/cskr/pubsub v1.0.2
	github.com/deckarep/golang-set v1.7.1
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce
	github.com/dustin/go-humanize v1.0.0
	github.com/envoyproxy/go-control-plane v0.9.9
	github.com/fatih/color v1.10.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.1 // indirect
	github.com/golang/mock v1.4.1
	github.com/golang/protobuf v1.4.3
	github.com/golangci/golangci-lint v1.32.2
	github.com/google/go-cmp v0.5.4
	github.com/google/uuid v1.1.2
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/vault/api v1.0.4
	github.com/jetstack/cert-manager v1.3.1
	github.com/jinzhu/copier v0.2.4
	github.com/jstemmer/go-junit-report v0.9.1
	github.com/matm/gocov-html v0.0.0-20200509184451-71874e2e203b
	github.com/mitchellh/gox v1.0.1
	github.com/mitchellh/hashstructure/v2 v2.0.1
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822
	github.com/norwoodj/helm-docs v1.4.0
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.10.0
	github.com/rs/zerolog v1.18.0
	github.com/servicemeshinterface/smi-sdk-go v0.5.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20210414055047-fe65e336abe0 // indirect
	golang.org/x/tools v0.1.1-0.20210319172145-bda8f5cee399 // indirect
	gomodules.xyz/jsonpatch/v2 v2.0.1
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.4.0
	helm.sh/helm/v3 v3.5.3
	honnef.co/go/tools v0.1.1 // indirect
	k8s.io/api v0.20.5
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.5
	k8s.io/cli-runtime v0.20.5
	k8s.io/client-go v0.20.5
	k8s.io/code-generator v0.20.5
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
	mvdan.cc/gofumpt v0.1.0 // indirect
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/kind v0.11.1
)

replace (
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
)
