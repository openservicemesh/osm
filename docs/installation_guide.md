# Installing OSM

This guide shows how to install the OSM CLI. OSM can be installed either from source, or from pre-built binary releases.

## From the Binary Releases

## From Source (Linux, MacOS)
Building OSM from source is slightly more work but is the best way to test the latest changes and useful in a development environment.

You must have a working [Go](https://golang.org/doc/install) environment.

```console
$ git clone git@github.com:open-service-mesh/osm.git
$ cd osm
$ make build-osm
```
`make build-osm` will fetch any required dependencies, compile `osm` and place it in `bin/osm`.

Once you have installed the `osm` cli, you can move on to using the `osm` cli to install the OSM control plane on to a Kubernetes cluster.

Next Steps: See the [Quick Start Guide](docs/quick_start_guide.md) for more information on how to get up and running with OSM.
