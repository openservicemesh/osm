# How to run a Demo of OSM

## System Requirements
- Go version 1.14 or higher
- Kubectl version 1.16 or higher
- Docker CLI
   - on a Debian based GNU/Linux system: `sudo apt-get install docker`
   - on a macOS use `brew install docker` or alternatively visit [Docker for Mac](https://docs.docker.com/docker-for-mac/install/)
   - on Windows visit [Docker for Windows](https://docs.docker.com/docker-for-windows/install/)
- [Watch](http://www.linfo.org/watch.html)
   - `brew install watch` on macOS

## Prerequisites
1. Clone this repo on your workstation
2. Setup `.env` environment variable file
   - From the root of the repository run `make .env`
   - It is already listed in `.gitignore` so that anything you put in it would not accidentally leak into a public git repo. Refer to `.env.example` in the root of this repo for the mandatory and optional environment variables.
2. Provision access to a Kubernetes cluster **(choose 1 option)**:
	- **Option 1:** Local [kind](https://kind.sigs.k8s.io/) cluster
	    - [Install kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
	       - `brew install kind` on macOS
	    - Provision a local cluster in Docker: `make kind-up`
	- **Option 2:** Azure Kubernetes Service managed cluster - save the credentials in `~/.kube/config` or set the config path in `$KUBECONFIG` env variable:
		- [Install Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
		- Login to your Azure account: `az login`
		- Create an AKS cluster via [Azure Portal](https://portal.azure.com/)
		- Using the Azure CLI download AKS credentials into `~/.kube/config`: `az aks get-credentials --resource-group your_Resource_Group --name your_AKS_name`
		- Set `KUBECONFIG` environment variable: `export KUBECONFIG=~/.kube/config` (also add this command to your .bashrc)
        - Authenticate with a container registry, which is accessible to both your workstation and your Kubernetes cluster. One such registry is the Azure Container Registry (ACR), which is used by the demo scripts in this repo:
           - [Install Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
           - Login to your Azure account: `az login`
           - Create an ACR via [Azure Portal](https://portal.azure.com/)
           - Create local Docker credentials for your ACR: `az acr login --name name_of_your_Azure_Container_Registry`. This command will create new credentials in `~/.docker/config.json`, which will be used by the demo scripts below.
        - Create [Azure authentication JSON](https://docs.microsoft.com/en-us/dotnet/api/overview/azure/containerinstance?view=azure-dotnet#authentication) file. These credentials will be used by OSM to connect to Azure and fetch IP addresses of virtual machines participating in the service mesh: `az ad sp create-for-rbac --sdk-auth > $HOME/.azure/azureAuth.json`
        - Configure Environment Variables
           - In the `.env` file, update the two values `CTR_REGISTRY` and `CTR_REGISTRY_PASSWORD` with appropriate values. The optional environment variables only need to be set if the default values used in the demo need to be overridden.

## Run the Demo
From the root of this repository execute:
```shell
./demo/run-osm-demo.sh
```

### This script will:
  - compile OSM's control plane (ADS), create a separate container image and push it to the workstation's default container registry (See `~/.docker/config.json`)
  - create a `bookstore` service that provides the `bookstore` domain for the `bookstore` service backends
  - create a `bookbuyer` service that curls `bookstore` domain for books (see `demo/cmd/bookbuyer/bookbuyer.go`); creates a container and uploads it to your contaner registry; creates a deployment for the `bookbuyer` service
  - create a `bookthief` service that curls the `bookstore` domain for books (see `demo/cmd/bookthief/bookthief.go`); creates a container and uploads it to your contaner registry; creates a deployment for the `bookthief` service
  - create 2 backends for `bookstore` service `bookstore-v1` and `bookstore-v2`, composed of a single binary, a web server, which increases a counter (books bought) on every GET request/response and returns that counter in a header; creates a container and uploats it to your contaner registry
  - applies SMI traffic policies allowing `bookbuyer` to access `bookstore-v1` and `bookstore-v2`, while preventing `bookthief` from accessing the `bookstore` services
  - installs Zipkin and points all Envoy pods to it
  - finally, a command indefinitely watches the relevant pods within the Kubernetes cluster

To see the results of deploying the services and the service mesh - run the tailing scripts:
  - the scripts will connect to the respecitve Kubernetes Pod and stream its logs
  - the output will be the output of the curl command to the `bookstore` domain and the count of books sold
  - a properly working service mesh will result in HTTP 200 OK with `./demo/tail-bookbuyer.sh` along with a monotonically increasing counter appearing in the response headers, while `./demo/tail-bookthief.sh` will result in HTTP 404 Not Found. This can be automatically checked with `KUBECONFIG=$HOME/.kube/config go run ./ci/cmd/maestro.go`

## View Mesh Topology with Zipkin
The OSM demo will install a Zipkin pod, and configure all participating Envoys to send spans to it. Zipkin's UI is running on port 9411. To view the web UI, forward port 9411 from the Zipkin pod to the local workstation and navigate to http://localhost:9411/zipkin/dependency. In the `./scripts` directory we have included a helper script to find the Zipkin pod and forward the port: `./scripts/port-forward-zipkin.sh`

## Demo Web UI
The Bookstore, Bookbuyer, and Bookthief apps have simple web UI visualizing the number of requests made between the services.

  - To see the UI for Bookbuyer run `./scripts/port-forward-bookbuyer-ui.sh` and open [http://localhost:8080/](http://localhost:8080/)
  - To see the UI for Bookstore v1 run `./scripts/port-forward-bookstore-ui-v1.sh` and open [http://localhost:8081/](http://localhost:8081/)
  - To see the UI for Bookstore v2 run `./scripts/port-forward-bookstore-ui-v2.sh` and open [http://localhost:8082/](http://localhost:8082/)
  - To see the UI for BookThief run `./scripts/port-forward-bookthief-ui.sh` and open [http://localhost:8083/](http://localhost:8083/)
  - To see Zipkin run `./scripts/port-forward-zipkin.sh` and open [http://localhost:9411/zipkin/](http://localhost:9411/zipkin/)
  - To see Grafana run `./scripts/port-forward-grafana.sh` and open [http://localhost:3000/](http://localhost:3000/) - default username and password for Grafana is `admin`/`admin`
  - OSM controller has a simple debugging web endpoint - run `./scripts/port-forward-osm-debug.sh` and open [http://localhost:9091/debug](http://localhost:9091/debug)

To expose web UI ports of all components of the service mesh the local workstation use the following helper script: `/scripts/port-forward-all.sh`

## Onboarding VMs to a service mesh
*_Note: This is an experimental feature and not currently fully supported*
The following sections outline how to onboard VMs to participate in a service mesh comprising of services running in a Kubernetes cluster.


### Requirements
- Ubuntu VM on Azure
- AKS cluster with advanced networking enabled - required for direct connectivity between K8s pods and services within the Azure VNET

### Bootstrapping the VM with Envoy proxy

#### Install and set up Envoy proxy
- Install the Envoy proxy package
	```
	$ curl -sL https://getenvoy.io/gpg | sudo apt-key add -
	$ sudo add-apt-repository "deb [arch=amd64] https://dl.bintray.com/tetrate/getenvoy-deb $(lsb_release -cs) stable"
	$ sudo apt-get update
	$ sudo apt-get install -y getenvoy-envoy
	```
- Verify Envoy is installed
	```
	$ envoy --version
	```
- Copy the Envoy boostrap configuration file `osm/config/bootstrap.yaml`  to `/etc/envoy/bootstrap.yaml`
	Refer to [Envoy - Getting Started guide](https://www.envoyproxy.io/docs/envoy/latest/start/start#https://www.envoyproxy.io/docs/envoy/latest/start/start#) for setting up the bootstrap configuration.

- Add the hostname to IP address mapping for the xDS services in `/etc/hosts` file on the VM so that the envoy proxy can connect to the xDS services using their hostname specified in the bootstrap config file.

- Configure the Envoy service by creating `envoy.service` file under `/etc/systemd/system` and register it as a service
	```
	[Unit]
	Description=Envoy

	[Service]
	ExecStart=/usr/bin/envoy -c /etc/envoy/bootstrap.yaml
	Restart=always
	RestartSec=5
	KillMode=mixed
	SyslogIdentifier=envoy
	LimitNOFILE=640000

	[Install]
	WantedBy=multi-user.target
	```
	```
	$ systemctl daemon-reload
	```
- Set up the certificates required for mTLS between Envoy proxies and for Envoy proxy to OSM control plane communication
	- Copy `osm/demo/certificates/*` to `/etc/certs/` on the VM
	- Copy `osm/bin/cert.pem`, `osm/bin/key.pem` to `/etc/ssl/certs/` on the VM

- Start Envoy proxy
	```
	$ systemctl start envoy
	```

- Check `/var/log/syslog` if you encounter issues with Envoy

- Copy and run the bookstore app `osm/demo/bin/bookstore` on the VM

## Deleting the kind cluster
When you are done with the demo and want to clean up your local kind cluster, just run the following.
```shell
make kind-reset
```
