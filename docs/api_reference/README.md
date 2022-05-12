# Generating API Reference Documentation

This document describes the process to generate custom API reference documentation meant to be serverd on [OSM's docs website](https://docs.openservicemesh.io/).

## 1. Fork the [Open Service Mesh Docs](https://github.com/openservicemesh/osm-docs) repository

1. Visit `https://github.com/openservicemesh/osm-docs`.
1. Click the `Fork` button and clone your fork.

## 2. Set up the [Kubernetes Custom Resource API Reference Docs generator](https://github.com/ahmetb/gen-crd-api-reference-docs)

1. Visit `https://github.com/ahmetb/gen-crd-api-reference-docs`.
1. Clone to repository locally.
1. Run `go build` from the root of the repository to generate the `gen-crd-api-reference-docs` binary executable.

## 3. Generate the API reference docs

From the root of the `osm` repository, use the `gen-crd-api-reference-docs` binary to generate custom API reference documentation based on the Go API definititions present within the `osm` repository.

For example, to generate API reference docs for the `MeshConfig` custom API defined in `/pkg/apis/config/v1alpha1/`:
```bash
<path to api doc generator repo>/gen-crd-api-reference-docs -config `pwd`/docs/api_reference/config.json -api-dir "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1" -template-dir <full path to api doc generator repo>/template/ -out-file <path to osm-docs repo>/content/docs/api_reference/config/v1alpha1.md
```

## 4. Customize the generated doc for the website

[OSM's docs website](https://docs.openservicemesh.io/) is built using Hugo and requires every page to have a [Front Matter](https://gohugo.io/content-management/front-matter/) defined.

Add the `Front Matter` to the generated docs so they render correctly on the website.

For example, a `Front Matter` looks as follows:
```
---
title: "Policy v1alpha1 API Reference"
description: "Policy v1alpha1 API reference documentation."
type: docs
---
```

Add `_index.md` files to intermediary directories if necessary.

## 5. Create a pull request in the [Open Service Mesh Docs](https://github.com/openservicemesh/osm-docs) repository

Commit the generated API reference documentation and create a pull request in [Open Service Mesh Docs](https://github.com/openservicemesh/osm-docs) repository.