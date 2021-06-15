---
title: "Release Guide"
description: "OSM Release Guide"
type: docs
---

# Release Guide

This guide describes the process to create a GitHub Release for this project.

**Note**: These steps assume that all OSM components are being released together, including the CLI, container images, and Helm chart, all with the same version.

## Release Candidates

Release candidates (RCs) should be created before each significant release so final testing can be performed. RCs are tagged as `vX.Y.Z-rc.W`. After the following steps have been performed to publish the RC, perform any final testing with the published release artifacts for about one week.

If issues are found, submit patches to the RC's release branch and create a new RC with the tag `vX.Y.Z-rc.W+1`. Apply those same patches to the `main` branch. Repeat until the release is suitably stable.

Once an RC has been found to be stable, cut a release tagged `vX.Y.Z` using the following steps.

- [Release Guide](#release-guide)
  - [Release Candidates](#release-candidates)
  - [Create a release branch](#create-a-release-branch)
  - [Update release branch with patches and versioning changes](#update-release-branch-with-patches-and-versioning-changes)
  - [Create and push a Git tag](#create-and-push-a-git-tag)
  - [Add release notes](#add-release-notes)
  - [Update docs site](#update-docs-site)
  - [Announce the new release](#announce-the-new-release)
  - [Make version changes on main branch](#make-version-changes-on-main-branch)
  - [Make version changes on docs.openservicemesh.io](#make-version-changes-on-docsopenservicemeshio)

## Create a release branch

Look for a branch on the upstream repo named `release-vX.Y`, where `X` and `Y` correspond to the major and minor version of the semver tag to be used for the new release. If the branch already exists, skip to the next step.

Identify the base commit in the `main` branch for the release and cut a release branch off `main`.
```console
$ git checkout -b release-<version> <commit-id> # ex: git checkout -b release-v0.4 0d05587
```

Push the release branch to the main repo (not fork), identified here by the `upstream` remote.
```console
$ git push upstream release-<version> # ex: git push upstream release-v0.4
```

## Update release branch with patches and versioning changes

Create a new branch off of the release branch to maintain updates specific to the new version.

If there are other commits on the `main` branch to be included in the release (such as for successive release candidates or patch releases), cherry-pick those onto this new branch.

Create a new commit on the new branch to update the hardcoded version information in the following locations:

* The container image tag in [charts/osm/values.yaml](/charts/osm/values.yaml)
* The chart and app version in [charts/osm/Chart.yaml](/charts/osm/Chart.yaml)
* The default osm image tag in [osm cli mesh upgrade](/cmd/cli/mesh_upgrade.go)
* The Helm chart [README.md](/charts/osm/README.md)
  - Necessary changes should be made automatically by running `make chart-readme`
* The init container image version in [charts/osm/crds/meshconfig.yaml](/charts/osm/crds/meshconfig.yaml)
* The init container image version in [pkg/constants/constants.go](/pkg/constants/constants.go)
* The image versions contained in tests.
  - [cmd/init-osm-controller/init-osm-controller_test.go](/cmd/init-osm-controller/init-osm-controller_test.go)
  - [pkg/configurator/methods_test.go](/pkg/configurator/methods_test.go)
* The container image versions used in the examples.
  - [docs/example/manifests/apps/bookbuyer.yaml](/docs/example/manifests/apps/bookbuyer.yaml)
  - [docs/example/manifests/apps/bookstore-v2.yaml](/docs/example/manifests/apps/bookstore-v2.yaml)
  - [docs/example/manifests/apps/bookstore.yaml](/docs/example/manifests/apps/bookstore.yaml)
  - [docs/example/manifests/apps/bookthief.yaml](/docs/example/manifests/apps/bookthief.yaml)
  - [docs/example/manifests/apps/bookwarehouse.yaml](/docs/example/manifests/apps/bookwarehouse.yaml)
  - [docs/example/manifests/meshconfig/mesh-config.yaml](/docs/example/manifests/meshconfig/mesh-config.yaml)

Once patches and version information have been updated on a new branch off of the release branch, create a pull request from the new branch to the release branch. When creating your pull request, generate the release checklist for the description by adding the following to the PR URL: `?expand=1&template=release_pull_request_template.md`. Alternatively, copy the raw template from [release_pull_request_template.md](/.github/PULL_REQUEST_TEMPLATE/release_pull_request_template.md).

Proceed to the next step once the pull request is approved and merged.

## Create and push a Git tag

Ensure your local copy of the release branch has the latest changes from the PR merged above.

Once the release is ready to be published, create and push a Git tag from the release branch to
the main repo (not fork), identified here by the `upstream` remote.

```console
$ export RELEASE_VERSION=<release-version> # ex: export RELEASE_VERSION=v0.4.0
$ git tag "$RELEASE_VERSION"
$ git push upstream "$RELEASE_VERSION"
```

A [GitHub Action](/.github/workflows/release.yml) is triggered when the tag is pushed.
It will build the CLI binaries, publish a new GitHub release,
upload the packaged binaries and checksums as release assets, build and push Docker images for OSM and the demo to the
[`openservicemesh` organization](https://hub.docker.com/u/openservicemesh) on Docker Hub, and publish the Helm chart to the repo hosted at https://openservicemesh.github.io/osm.

## Add release notes

In the description section of the new release, add information about feature additions, bug fixes,
and any other administrative tasks completed on the repository.

## Update docs site

In the docs site's main branch, edit the file [https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md) to update any version references in the manual demo.

  - [This demo of OSM v0.8.4 requires:](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L13)
  - [Download the 64-bit GNU/Linux or macOS binary of OSM v0.8.4:](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L30)
  - [release=v0.8.4](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L33)
  - [Download the 64-bit Windows OSM v0.8.4 binary via Powershell:](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L40)
  - [wget  https://github.com/openservicemesh/osm/releases/download/v0.8.4/osm-v0.8.4-windows-amd64.zip -o osm.zip](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L42)
  - [image: openservicemesh/bookbuyer:v0.8.4](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L199)
  - [image: openservicemesh/bookthief:v0.8.4](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L231)
  - [image: openservicemesh/bookstore:v0.8.4](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L283)
  - [image: openservicemesh/bookwarehouse:v0.8.4](https://github.com/openservicemesh/osm-docs/blame/main/content/docs/install/manual_demo.md#L339)

## Announce the new release

Make an announcement on the [OSM mailing list](https://groups.google.com/g/openservicemesh) and [OSM Slack channel](https://cloud-native.slack.com/archives/openservicemesh) after you [join CNCF Slack](https://slack.cncf.io/).

## Make version changes on main branch

Skip this step if the release is a release candidate (RC).

Open a pull request against the `main` branch making the same version updates as [above](#update-release-branch-with-patches-and-versioning-changes) so the latest release assets are referenced there.
