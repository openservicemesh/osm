# Package: Certificate

This package contains tools necessary to manage certificates for the service mesh. There are 3 kinds of certificates:

  - the Envoy-to-xDS certificates - used for Envoys to connect and identify themselves to the xDS control plane.
  - the Envoy-to-Envoy (east-west) certificates - used for encryption and identification of sidecars in communication between services.
  - the root certificate for the entire mesh, which signs the leaf certs above

## Interfaces

In `types.go` we define 2 interfaces:

  1. `certificate.Manager` - is the interface exposing a particular certificate provider. The certificate manager is responsible for issuing and renewing certificates. It abstracts away the particular methods of signing, renewing, and storing certificates away from the rest of the service mesh components.
  2. `certificate.Certificater` - an abstraction over an actual certificate, which is signed by our CA, has an expiration, and certain properties common to all PEM encoded certificates issued by any certificate provider implemented.


## Providers
The directory `providers` contains implementations of certificate issuers (`certificate.Manager`s):

  1. `tresor` is a minimal internal implementation of a certificate issuer, which leverages Go's `crypto` library and uses Kubernetes' etcd for storage.
  2. `keyvault` is a certificate issuer leveraging Azure Key Vault for secrets storage.
  3. `vault` is another implementation of the `certificate.Manager` interface, which provides a way for all service mesh certificates to be stored on and signed by [Hashicorp Vault](https://www.vaultproject.io/).
  4. `cert-manager` is a certificate issuer leveraging [cert-manager](https://cert-manager.io) to sign certificates from [Issuers](https://cert-manager.io/docs/concepts/issuer/).

## Certificate Rotation
In the `rotor` directory we implement a certificate rotation mechanism, which may or may not be leveraged by the certificate issuers (`providers`).
