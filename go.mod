module github.com/deislabs/smc

go 1.13

require (
	github.com/Azure/application-gateway-kubernetes-ingress v0.0.0-20190820222238-dfce74a2549f
	github.com/Azure/azure-sdk-for-go v34.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.5.0
	github.com/Azure/go-autorest/autorest/azure/auth v0.1.0
	github.com/Azure/go-autorest/tracing v0.2.0 // indirect
	github.com/deislabs/smi-sdk-go v0.2.0
	github.com/eapache/channels v1.1.0
	github.com/envoyproxy/go-control-plane v0.8.4
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0 // indirect
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9 // indirect
	golang.org/x/sys v0.0.0-20190616124812-15dcb6c0061f // indirect
	google.golang.org/grpc v1.22.1
	gopkg.in/yaml.v2 v2.2.4 // indirect
	k8s.io/api v0.0.0-20190614205929-e4e27c96b39a
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0 // indirect
)

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190612125919-5c45477a8ae7
