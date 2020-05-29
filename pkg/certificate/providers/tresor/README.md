# Tresor Certificate Provider

The Tresor package is a minimal certificate issuance facility, which leverages Go's `crypto` libraries to generate a CA, and issue certificates for Envoy-to-xDS communication as well as Envoy-to-Envoy (east-west) between services.
