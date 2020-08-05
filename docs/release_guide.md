# Release Guide

This guide describes the process to create a GitHub Release for this project.

1. [Create and push a Git tag](#create-and-push-a-git-tag)
1. [Add release notes](#add-release-notes)
1. [Announce the new release](#announce-the-new-release)

## Create and push a Git tag
```console
$ export RELEASE_VERSION=<release-version> # ex: export RELEASE_VERSION=v0.1.0
$ git tag -a "$RELEASE_VERSION" -m "<add description here>"
$ git push origin "$RELEASE_VERSION"
```

A [GitHub Action](/.github/workflows/release.yml) is triggered when the tag is pushed. It will build the CLI binaries, publish a new GitHub release, upload the packaged binaries and checksums as release assets, and build and push Docker images for OSM and the demo to the [`openservicemesh` organization](https://hub.docker.com/u/openservicemesh) on Docker Hub.

## Add release notes
In the description section of the new release, add information about feature additions, bug fixes, and any other administrative tasks completed on the repository.

## Announce the new release
- Make an announcement on the [mailing list](https://groups.google.com/g/openservicemesh).
