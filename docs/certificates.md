# mTLS and Certificate Issuance
Open Service Mesh uses mTLS for encryption of data between pods as well as Envoy and service identity. Certificates are created and distributed to each Envoy proxy via the SDS protocol by the OSM control plane.

There exist 2 kinds of certificates in the OSM ecosystem:

1. Certificates used for Envoy proxies to connect to xDS control plane - identifies the proxy and pod connecting to xDS.
2. Certificates used for service to service communication (one Envoy connects to another) - identifies the services connecting to each other.

Open Service Mesh supports 3 methods of issuing certificates:
  - using an internal OSM package, called [Tresor](./pkg/certificate/providers/tresor/). This is the defualt for a first time installation.
  - using [Hashicorp Vault](https://www.vaultproject.io/)
  - using [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/)


## Using OSM's Tresor certificate issuer

Open Service Mesh includes a package, [tresor](./pkg/certificate/providers/tresor/). This is a minimal implementation of the `certificate.Manager` interface. It issues certificates leveraging the `crypto` Go library, and stores these certificates as Kubernetes secrets.

  - To use the `tresor` package during development set `export CERT_MANAGER=tresor` in the `.env` file of this repo.

  - To use this package in your Kubernetes cluster set the `CERT_MANAGER=tresor` variable in the Helm chart prior to deployment.

Additionally:
  - `--ca-bundle-secret-name` - this string is the name of the Kubernetes secret, where the CA root certificate and private key will be saved.


## Using Hashicorp Vault

Service Mesh operators, who consider storing their service mesh's CA root key in Kubernetes insecure have the option to integrate with a [Hashicorp Vault](https://www.vaultproject.io/) installation. In such scenarios a pre-configured Hashi Vault is required. Open Service Mesh's control plane connects to the URL of the Vault, authenticates, and begins requesting certificates. This setup shifts the responsibility of correctly and securely configuring Vault to the operator.

The following configuration parameters will be required for OSM to integrate with an existing Vault installation:
  - Vault address
  - Vault token
  - Validity period for certificates

CLI flags control how OSM integrates with Vault. The following OSM command line parameters must be configured to issue certificates with Vault:
  - `--cert-manager` - set this to `vault`
  - `--vault-host` - host name of the Vault server (example: `vault.contoso.com`)
  - `--vault-protocol` - protocol for Vault connection (`http` or `https`)
  - `--vault-token` - token to be used by OSM to connect to Vault (this is issued on the Vault server for the particular role)
  - `--vault-role` - role created on Vault server and dedicated to Open Service Mesh (example: `open-service-mesh`)
  - `--service-cert-validity-minutes` - number of minutes - period for which each new certificate issued for service-to-service communication will be valid

Additionally:
  - `--ca-bundle-secret-name` - this string is the name of the Kubernetes secret where the service mesh root certificate will be stored. When using Vault (unlike Tresor) the root key will **not** be exported to this secret.


### Installing Hashi Vault

Installation of Hashi Vault is out of scope for the Open Service Mesh project. Typically this is the responsibility of dedicated security teams. Documentation on how to deploy Vault securely and make it highly available is available on [Vault's website](https://learn.hashicorp.com/vault/getting-started/install).

This repository does contain a [script (deploy-vault.sh)](./demo/deploy-vault.sh), which is used to automate the deployment of Hashi Vault for continuous integration. This is strictly for development purposes only. Running the script will deploy Vault in a Kubernetes namespace defined by the `$K8S_NAMESPACE` environment variable in your [.env](./.env.example) file. This script can be used for demonstration purposes. It requires the following environment variables:
```
export K8S_NAMESPACE=osm-system-ns
export VAULT_TOKEN=xyz
```

Running the `./demo/deploy-vault.sh` script will result in a dev Vault installation:
```
NAMESPACE         NAME                                    READY   STATUS    RESTARTS   AGE
osm-system-ns     vault-5f678c4cc5-9wchj                  1/1     Running   0          28s
```

Fetching the logs of the pod will show details on the Vault installation:
```
==> Vault server configuration:

             Api Address: http://0.0.0.0:8200
                     Cgo: disabled
         Cluster Address: https://0.0.0.0:8201
              Listener 1: tcp (addr: "0.0.0.0:8200", cluster address: "0.0.0.0:8201", max_request_duration: "1m30s", max_request_size: "33554432", tls: "disabled")
               Log Level: info
                   Mlock: supported: true, enabled: false
           Recovery Mode: false
                 Storage: inmem
                 Version: Vault v1.4.0

WARNING! dev mode is enabled! In this mode, Vault runs entirely in-memory
and starts unsealed with a single unseal key. The root token is already
authenticated to the CLI, so you can immediately begin using Vault.

You may need to set the following environment variable:

    $ export VAULT_ADDR='http://0.0.0.0:8200'

The unseal key and root token are displayed below in case you want to
seal/unseal the Vault or re-authenticate.

Unseal Key: cZzYxUaJaN10sa2UrPu7akLoyU6rKSXMcRt5dbIKlZ0=
Root Token: xyz

Development mode should NOT be used in production installations!

==> Vault server started! Log data will stream in below:
...
```

The outcome of deploying Vault in your system is a URL and a token. For instance the URL of Vauld could be `http://vault.osm-system-ns.svc.cluster.local` and the token `xxx`.

### Configure OSM with Vault

After Vault installation and before we use Helm to deploy OSM, the following parameters must be provided provided in the Helm chart:
```
CERT_MANAGER=vault
VAULT_HOST="vault.${K8S_NAMESPACE}.svc.cluster.local"
VAULT_PROTOCOL=http
VAULT_TOKEN=xyz
VAULT_ROLE=open-service-mesh
```

When running OSM on your local workstation, use the following CLI parameters:
```
--cert-manager="vault"
--vault-host="localhost"  # or the host where Vault is installed
--vault-protocol="http"
--vault-token="xyz"
--vault-role="open-service-mesh'
--service-cert-validity-minutes=60
```

### How OSM Integrates with Vault

When the OSM control plane starts, a new certificate issuer is instantiated.
The kind of cert issuer is determined by the `--cert-manager` CLI parameter.
When this is set to `vault` OSM uses a Vault cert issuer.
This is a Hasticorp Vault client, which satisfies the `certificate.Manager`
interface. It provides the following methods:
```
  - IssueCertificate - issues new certificates
  - GetCertificate - retrieves a certificate given its Common Name (CN)
  - RotateCertificate - rotates expiring certificates
  - GetAnnouncementsChannel - returns a channel, which is used to announce when certificates have been issued or rotated
```

OSM assumes that a CA has already been created on the Vault server.
OSM also requires a dedicated Vault role (for instance `pki/roles/open-service-mesh`).
The Vault role created by the `./demo/deploy-vault.sh` script applies the following configuration, which is only appropriate for development purposes:

  - `allow_any_name`: `true`
  - `allow_subdomains`: `true`
  - `allow_baredomains`: `true`
  - `allow_localhost`: `true`
  - `max_ttl`: `24h`


Hashi Vault's site has excellent [documentation](https://learn.hashicorp.com/vault/secrets-management/sm-pki-engine)
on how to create a new CA. The `./demo/deplay-vault.sh` script uses the
following commands to setup the dev environment:

    export VAULT_TOKEN="xyz"
    export VAULT_ADDR="http://localhost:8200"
    export VAULT_ROLE="open-service-mesh

    # Launch the Vault server in dev mode
    vault server -dev -dev-listen-address=0.0.0.0:8200 -dev-root-token-id=${VAULT_TOKEN}

    # Also save the token locally so this is available
    echo $VAULT_TOKEN>~/.vault-token;

    # Enable the PKI secrets engine (See: https://www.vaultproject.io/docs/secrets/pki#pki-secrets-engine)
    vault secrets enable pki;

    # Set the max lease TTL to a decade
    vault secrets tune -max-lease-ttl=87600h pki;

    # Set URL configuration (See: https://www.vaultproject.io/docs/secrets/pki#set-url-configuration)
    vault write pki/config/urls issuing_certificates='http://127.0.0.1:8200/v1/pki/ca' crl_distribution_points='http://127.0.0.1:8200/v1/pki/crl';

    # Configure a role named "open-service-mesh" (See: https://www.vaultproject.io/docs/secrets/pki#configure-a-role)
    vault write pki/roles/${VAULT_ROLE} allow_any_name=true allow_subdomains=true;

    # Create a root certificate named "osm.root" (See: https://www.vaultproject.io/docs/secrets/pki#setup)
    vault write pki/root/generate/internal common_name='osm.root' ttl='87600h'


The OSM control plane provides verbose logs on operations done with the Vault installations.
