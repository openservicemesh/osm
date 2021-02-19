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

1. [Create a release branch](#create-a-release-branch)
1. [Update release branch with patches and versioning changes](#update-release-branch-with-patches-and-versioning-changes)
1. [Create and push a Git tag](#create-and-push-a-git-tag)
1. [Add release notes](#add-release-notes)
1. [Announce the new release](#announce-the-new-release)
1. [Make version changes on main branch](#make-version-changes-on-main-branch)

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

* The container image tag in [charts/osm/values.yaml](https://github.com/openservicemesh/osm/tree/main/charts/osm/values.yaml)
* The chart and app version in [charts/osm/Chart.yaml](https://github.com/openservicemesh/osm/tree/main/charts/osm/Chart.yaml)
* The default osm-controller image tag in [osm cli](https://github.com/openservicemesh/osm/blob/main/cmd/cli/install.go)
* The image tags used in the [demo manifests](https://github.com/openservicemesh/osm/blob/main/docs/example/manifests/apps)
* The Helm chart [README.md](https://github.com/openservicemesh/osm/blob/main/charts/osm/README.md)
  - Necessary changes should be made automatically by running `make chart-readme`

Once patches and version information have been updated on a new branch off of the release branch, create a pull request from the new branch to the release branch. Proceed to the next step once the pull request is approved and merged.

## Create and push a Git tag

Ensure your local copy of the release branch has the latest changes from the PR merged above.

Once the release is ready to be published, create and push a Git tag from the release branch to
the main repo (not fork), identified here by the `upstream` remote.

```console
$ export RELEASE_VERSION=<release-version> # ex: export RELEASE_VERSION=v0.4.0
$ git tag "$RELEASE_VERSION"
$ git push upstream "$RELEASE_VERSION"
```

A [GitHub Action](https://github.com/openservicemesh/osm/blob/main/.github/workflows/release.yml) is triggered when the tag is pushed.
It will build the CLI binaries, publish a new GitHub release,
upload the packaged binaries and checksums as release assets, build and push Docker images for OSM and the demo to the
[`openservicemesh` organization](https://hub.docker.com/u/openservicemesh) on Docker Hub, and publish the Helm chart to the repo hosted at https://openservicemesh.github.io/osm.

## Add release notes

In the description section of the new release, add information about feature additions, bug fixes,
and any other administrative tasks completed on the repository.

## Announce the new release

Make an announcement on the [mailing list](https://groups.google.com/g/openservicemesh) and [Slack channel](https://cloud-native.slack.com/archives/C018794NV1C).

## Make version changes on main branch

Skip this step if the release is a release candidate (RC).

Open a pull request against the `main` branch making the same version updates as [above](#update-release-branch-with-versioning-changes) so the latest release assets are referenced there.

## Make version changes on docs.openservicemesh.io

To add the new version to the 'Releases' dropdown menu on [docs.openservicemesh.io](https://docs.openservicemesh.io/), refer to [this section](https://github.com/openservicemesh/osm/tree/main/docs#versioning-the-docs-site) of the site Readme.
