---
title: "Release Guide"
description: "OSM Release Guide"
type: docs
---

# Release Guide

This guide describes the process to create a GitHub Release for this project.

**Note**: These steps assume that all OSM components are being released together, including the CLI, container images, and Helm chart, all with the same version.

Once an RC has been found to be stable, cut a release tagged `vX.Y.Z` using the following steps.

- [Release Guide](#release-guide)
  - [Release Candidates](#release-candidates)
  - [Create a release branch](#create-a-release-branch)
  - [Add changes to be backported](#add-changes-to-be-backported)
  - [Create and push the pre-release Git tag](#create-and-push-the-pre-release-git-tag)
  - [Update release branch with patches and versioning changes](#update-release-branch-with-patches-and-versioning-changes)
  - [Create and push the release Git tag](#create-and-push-the-release-git-tag)
  - [Add release notes](#add-release-notes)
  - [Update documentation and docs.openservicemesh.io website](#update-documentation-and-docsopenservicemeshio-website)
    - [1. Create the release specific branch in osm-docs repo](#1-create-the-release-specific-branch-in-osm-docs-repo)
    - [2. Update version references to the latest version for the given Major.Minor version](#2-update-version-references-to-the-latest-version-for-the-given-majorminor-version)
    - [3. Update API reference documentation](#3-update-api-reference-documentation)
    - [4. Update error code documentation](#4-update-error-code-documentation)
  - [Announce the new release](#announce-the-new-release)
  - [Make version changes on main branch](#make-version-changes-on-main-branch)
  - [Workflow Diagram](#workflow-diagram)

## Create a release branch

Look for a branch on the upstream repo named `release-vX.Y`, where `X` and `Y` correspond to the major and minor version of the semver tag to be used for the new release. If the branch already exists, skip to the next step.

Identify the base commit in the `main` branch for the release and cut a release branch off `main`.
```console
$ git checkout -b release-<version> <commit-id> # ex: git checkout -b release-v0.4 0d05587
```
> Note: Care must be taken to ensure the release branch is created from a commit meant for the release. If unsure about the commit to use to create the release branch, please open an issue in the `osm` repo and a maintainer will assist you with this.

Push the release branch to the upstream repo (NOT forked), identified here by the `upstream` remote.
```console
$ git push upstream release-<version> # ex: git push upstream release-v0.4
```

## Add changes to be backported

Create a new branch off of the release branch to maintain updates specific to the new version. Let's call it the patch branch. The patch branch should not be created in the upstream repo.

If there are other commits on the `main` branch to be included in the release (such as for successive release candidates or patch releases), cherry-pick those onto the patch branch.

## Create and push the pre-release Git tag

The pre-release Git tag publishes the OSM control plane images to the `openservicemesh` organization in Dockerhub, and publishes the image digests as an artifact of the pre-release Github workflow. The image digests must be used in the next step to update the default control plane image referenced in the Helm charts.

The pre-release Git tag is of the form `pre-rel-<release-version>`, e.g. `pre-rel-v0.4.0`.

```console
$ PRE_RELEASE_VERSION=<pre-release-version> # ex: PRE_RELEASE_VERSION=pre-rel-v0.4.0
$ git tag "$PRE_RELEASE_VERSION"
$ git push upstream "$PRE_RELEASE_VERSION"
```

Once the pre-release Git tag has been pushed, wait for the Pre-release Github workflow to complete. Upon workflow completion, retrieve the image digests for the given release. The image digests are logged in the "Image digests" step of the Pre-release workflow.

The image digest logs contain the sha256 image digest for each control plane image as follows:
```
init: sha256:96bdf7c283ac679344ab1bc5badc406ff486ad2fecb46b209e11e19d2a0a4d3c
osm-controller: sha256:069f20906035d9b8c4c59792ee1f2b90586a6134a5e286bf015af8ee83041510
osm-injector: sha256:d2e96d99a311b120c4afd7bd3248f75d0766c98bd121a979a343e438e9cd2c35
osm-crds: sha256:359a4a6b031d0f72848d6bedc742b34b60323ebc5d5001071c0695130b694efd
osm-bootstrap: sha256:fd159fdb965cc0d3d7704afaf673862b5e92257925fc3f6345810f98bb6246f8
```

## Update release branch with patches and versioning changes

Create a new commit on the patch branch to update the hardcoded version information in the following locations:

* The control plane image digests defined by `osm.image.digest` for images in [charts/osm/values.yaml](/charts/osm/values.yaml) from the image digests obtained from the Pre-release workflow. For example, if the osm-controller image digest is `sha256:eb194138abddbe271d42b290489917168a6a891a3dabb575de02c53f13879bee`, update the value of `osm.image.digest.osmController` to `sha256:eb194138abddbe271d42b290489917168a6a891a3dabb575de02c53f13879bee`.
  - Ensure the value for `osm.image.tag` in [charts/osm/values.yaml](/charts/osm/values.yaml) is set to the empty string.
* The chart and app version in [charts/osm/Chart.yaml](/charts/osm/Chart.yaml) to the release version.
* The Helm chart [README.md](/charts/osm/README.md)
  - Necessary changes should be made automatically by running `make chart-readme`
* If this the first release on a new release branch, update the [upgrade e2e test](/tests/e2e/e2e_upgrade_test.go) install version from `i.Version = ">0.0.0-0"` to the previous minor release. e.g. When cutting the release-v1.1 branch, this should be updated to `"1.0"`.

Once patches and version information have been updated on the patch branch off of the release branch, create a pull request from the patch branch to the release branch. When creating your pull request, generate the release checklist for the description by adding the following to the PR URL: `?expand=1&template=release_pull_request_template.md`. Alternatively, copy the raw template from [release_pull_request_template.md](/.github/PULL_REQUEST_TEMPLATE/release_pull_request_template.md).

Proceed to the next step once the pull request is approved and merged.

## Create and push the release Git tag

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

The release job runs the `scripts/release-notes.sh` script to generate release notes for the specific release tag. Update the `Notable Changes` and `Deprecation Notes` section based on notable feature additions, critical bug fixes, and deprecated functionality.

## Update documentation and docs.openservicemesh.io website

### 1. Create the release specific branch in osm-docs repo

If a branch corresponding to the Major.Minor version is not available in the [osm-docs](https://github.com/openservicemesh/osm-docs) repo, create it based on https://github.com/openservicemesh/osm-docs/blob/main/README.md#adding-release-specific-docs. For example, to publish the documentation for v0.10.0, there must exist a release-v0.10 branch in the `osm-docs` repo.

*Note:
1. Do not create a branch for patch releases. The same documentation is used for patches having the same Major.Minor version.
1. Care must be taken to ensure the release branch is created from a commit that meant for the release. If unsure about the commit to use to create the release branch, please open an issue in the `osm-docs` repo and a maintainer will assist you with this.

### 2. Update version references to the latest version for the given Major.Minor version

For example, when v0.10.1 is being released, update all of the version references from v0.10.0 to v0.10.1 to reflect the latest documentation for the Major.Minor version. Instructions for updating the release version references can be found at [https://github.com/openservicemesh/osm-docs/blob/main/README.md/#update-the-release-references](https://github.com/openservicemesh/osm-docs/blob/main/README.md/#update-the-release-references). Image tags pinned to a specific version must also be updated in the demo manifests.

### 3. Update API reference documentation

Follow the [Generating API Reference Documentation](/docs/api_reference/README.md) guide to update the API references on the docs site.

### 4. Update error code documentation

On the docs site's main branch, edit the file [https://github.com/openservicemesh/osm-docs/blob/main/content/docs/guides/troubleshooting/control_plane_error_codes.md](https://github.com/openservicemesh/osm-docs/blob/main/content/docs/guides/troubleshooting/control_plane_error_codes.md) to update the OSM error code table.

1. Build OSM on the release branch.
1. Generate the mapping of OSM error codes and their descriptions using the `osm support` cli tool.

   ```
   ./bin/osm support error-info

    +------------+----------------------------------------------------------------------------------+
    | ERROR CODE |                                   DESCRIPTION                                    |
    +------------+----------------------------------------------------------------------------------+
    | E1000      | An invalid command line argument was passed to the application.                  |
    +------------+----------------------------------------------------------------------------------+
    | E1001      | The specified log level could not be set in the system.                          |
   ```

1. Copy the table and replace the existing table in the file [https://github.com/openservicemesh/osm-docs/blob/main/content/docs/guides/troubleshooting/control_plane_error_codes.md](https://github.com/openservicemesh/osm-docs/blob/main/content/docs/guides/troubleshooting/control_plane_error_codes.md).
1. If there were updates to the table, make a PR in the OSM docs repository.


## Announce the new release

Make an announcement on the [OSM mailing list](https://groups.google.com/g/openservicemesh) and [OSM Slack channel](https://cloud-native.slack.com/archives/openservicemesh) after you [join CNCF Slack](https://slack.cncf.io/).

## Make version changes on main branch

Skip this step if the release is a release candidate (RC).

Open a pull request against the `main` branch to update the `version` field in `charts/osm/Chart.yaml` to the release version.

## Workflow Diagram

Here is a diagram to illustrate the git branching strategy and workflow in the release process:

![OSM git branching strategy](../img/osm-git-release.jpg)
