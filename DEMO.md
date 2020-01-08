# How to run a Demo of SMC

## System Requirements
- Go version 1.13 or higher
- OpenSSL/LibreSSL 2.8.3 or higher
- Kubectl version 1.16 or higher
- Docker CLI
   - on a Debian based GNU/Linux system: `sudo apt-get install docker`
   - on a macOS use `brew install docker` or alternatively visit [Docker for Mac](https://docs.docker.com/docker-for-mac/install/)
   - on Windows visit [Docker for Windows](https://docs.docker.com/docker-for-windows/install/)
- [Watch](http://www.linfo.org/watch.html)
   - `brew install watch` on macOS

## Prerequisites
1. Clone this repo on your workstation
1. Provision access to a Kubernetes cluster - save the credentials in `~/.kube/config` or set the config path in `$KUBECONFIG` env variable:
   - The Azure Kubernetes Service is a fitting provider of a hosted Kubernetes service
   - [Install Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
   - Login to your Azure account: `az login`
   - Create an AKS cluster via [Azure Portal](https://portal.azure.com/)
   - Using the Azure CLI download AKS credentials into `~/.kube/config`: `az aks get-credentials --resource-group your_Resource_Group --name your_AKS_name`
1. Authenticate with a container registry, which is accessible to both your workstation and your Kubernetes cluster. One such registry is the Azure Container Registry (ACR), which is used by the demo scripts in this repo:
   - [Install Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
   - Login to your Azure account: `az login`
   - Create an ACR via [Azure Portal](https://portal.azure.com/)
   - Create local Docker credentials for your ACR: `az acr login --name name_of_your_Azure_Container_Registry`. This command will create new credentials in `~/.docker/config.json`, which will be used by the demo scripts below.
1. Create [Azure authentication JSON](https://docs.microsoft.com/en-us/dotnet/api/overview/azure/containerinstance?view=azure-dotnet#authentication) file. These credentials will be used by SMC to connect to Azure and fetch IP addresses of virtual machines participating in the service mesh: `az ad sp create-for-rbac --sdk-auth > $HOME/.azure/azureAuth.json`

## Configure Environment Variables
In the root directory of the repo create a `.env` file. It is already listed in `.gitignore` so that anything you put in it would not accidentally leak into a public git repo. The `.env` file should contain the following Bash variables:
   - `K8S_NAMESPACE` - Namespace within your Kubernetes cluster, where SMC will be installed. This cannot be the `default` namespace because it has to be a namespace that can be deleted.
   - `AZURE_SUBSCRIPTION` - the Azure subscription where your Kubernete cluster resides. The demo will use this to configure the Endpoint Discovery Service's cloud observer.
   - `CTR_REGISTRY` - URL of the container registry. For example: `draychev.azurecr.io/smc`
   - `CTR_REGISTRY_CREDS_NAME` - name to be used for the Kubernetes secrets resource to be created from the Docker container registry.

An example is provided in the `.env.example` in the root of this repo.

## Run the Demo
1. From the root of this repository execute `./demo/run-demo.sh`. This script will:
   - compile both the Endpoint Discovery Service and the Secrets Discovery Service, create separate containers and push these to the workstation's default container registry (See `~/.docker/config.json`)
   - create a BookBuyer service, composed of a Bash script running `curl http://bookstore.mesh/` in an infinite loop (see `demo/bookbuyer.sh`); creates a container and uploats it to your contaner registry
   - create a Bookstore service, composed of a single binary, a web server, which increases a counter on every GET request/response and returns that counter in a header; creates a container and uploats it to your contaner registry
   - the demo script assumes you have Azure Container Registry and automatically provisions credentials to your local workstation and pushes a secret to your Kubernetes cluster
   - another script creates certificates to be distributed by SDS and saves these in the Kubernetes cluster
   - bootstrap Envoy configs (ConfigMaps) for the bookstore and bookbuyer services are also uploaded (applid) to the K8s cluster
   - a Kubernetes Deployment and Services are applied for the bookstore, bookbuyer, eds, and sds containers
   - an SMI TrafficSplit policy is applied
   - finally a script runs in an infinite loop querying the Pods within the Kubernetes cluster

1. To see the results of deploying the services and the service mesh - run the tailing script: `./demo/tail-bookbuyer.sh`
   - the script will connect to the bookbuyer Kubernetes Pod and will stream its logs
   - the output will be the cURL command to the `bookstore.mesh` service and counter responses
   - a properly working service mesh will result in HTTP 200 OK responses from the bookstore, along with a monotonically increasing counter appearing in the response headers.
