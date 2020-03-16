# Generating OSM CRDs

This document outlines the steps necessary to generate the Go code supporting the CRDs in [this](./crd/) directory.

### Assumptions
Code generation scripts assumes:
  - `$GOPATH` is correctly setup on this workstation
  - this repository has been cloned in `$GOPATH/src/github.com/open-service-mesh/osm/`

### Prerequisites
  1. Download (clone) [the code-generation tool](https://github.com/kubernetes/code-generator):
        ```bash
        mkdir -p $GOPATH/src/k8s.io
        pushd $GOPATH/src/k8s.io
        git clone git@github.com:kubernetes/code-generator.git
        popd
        ```

### Generate Informers etc.
  1. Run the code-eneration tool:
        ```
        $GOPATH/src/k8s.io/code-generator/generate-groups.sh \
            all \
            github.com/open-service-mesh/osm/pkg/osm_client \
            github.com/open-service-mesh/osm/pkg/apis \
            "azureresource:v1"
        ```

### Install the Custom Resource Definitions:
  1. Install the actual CRD: `kubectl apply -f ./AzureResource.yaml`
  1. Create a sample object to test the new CRD: `kubectl apply -f ./AzureResource-example.yaml`
