# Continuous Integration System

For CI this project uses Github Actions.


## Set up Azure Container Registry

Create Azure Container Registry and enable admin credentials.
Run `az acr update -n osmci --admin-enabled true` to enable admin.

Create a Service Principal - see file [create-service-principal-for-acr.sh](./create-service-principal-for-acr.sh)

```bash
ACR_NAME=osmci
SERVICE_PRINCIPAL_NAME=osm-ci-acr-service-principal

# Obtain the full registry ID for subsequent command args
ACR_REGISTRY_ID=$(az acr show --name $ACR_NAME --query id --output tsv)

# Create the service principal with rights scoped to the registry.
SP_PASSWD=$(az ad sp create-for-rbac --name http://$SERVICE_PRINCIPAL_NAME --scopes $ACR_REGISTRY_ID --role owner --query password --output tsv)
SP_APP_ID=$(az ad sp show --id http://$SERVICE_PRINCIPAL_NAME --query appId --output tsv)

# Output the service principal's credentials; use these in your services and applications to authenticate to the container registry.
echo "DOCKER_USER: $SP_APP_ID"
echo "DOCKER_PASS: $SP_PASSWD"
```

Verify that the docker username and password work with

```bash
docker login myregistry.azurecr.io --username DOCKER_USER --password DOCKER_PASS
```

Set the variables in Github Secrets as `DOCKER_USER` and `DOCKER_PASS`

## Github Secrets
 - `ACR` - (string) the FQDN of the Azure Container Registry; example:  `osmci.azurecr.io`
 - `AZURE_SUBSCRIPTION` - (string) Azure subscription for the components in use; example: `9a3abd07-8c53-41eb-acad-2a3e36a4b90e`
 - `CTR_REGISTRY` - (string) is he container registry created with `/osm` appended at the end; example: : `osmci.azurecr.io/osm`
 - `CTR_REGISTRY_CREDS_NAME` - (string) name of the Kubernetes secret used to pull images from Azure Container Registry; example: `acr-creds`
 - `DOCKER_PASS` - (string) enable ACR admin account; get password from https://portal.azure.com -> ACR -> Access Keys
 - `DOCKER_USER` - (string) enable ACR admin account; use the name of the ACR as the user; example: `osmci` (also see [./ci/create-service-principal-for-acr.sh](./create-service-principal-for-acr.sh) for using a Service Principal)
 - `KUBECONFIG` - (string) set this to the location of the kube config file: `".kube/config"`
 - `OSM_HUMAN_DEBUG_LOG` - set it to `true` to show human-readable log lines (vs JSON blobs)
 - `VAULT_TOKEN` - (string) random string, which will be used as a Vault token in the CI Vault setup; example: `abcd`
 - `CI_MAX_WAIT_FOR_POD_TIME_SECONDS` - (integer) max number of seconds the CI system will wait for bookbuyer and bookthief pods to be ready / running; example: `15`
 - `CI_WAIT_FOR_OK_SECONDS` - (integer) number of seconds the CI system will wait for bookbuyer and bookthief pods to poll for a success once the pods are ready; example: `15`
