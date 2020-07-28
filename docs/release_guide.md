# Release Guide

This guide describes the process to create a GitHub Release for this project.

1. [Create and push a Git tag](#create-and-push-a-git-tag)
2. [Draft a GitHub Release](#draft-a-github-release)
3. [Add release notes](#add-release-notes)
4. [Build and add binaries](#build-and-add-binaries)
5. [Publish the GitHub Release](#publish-the-github-release)

## Create and push a Git tag
```console
$ export RELEASE_VERSION=<release-version> # ex: export RELEASE_VERSION=v0.1.0
$ git tag -a "$RELEASE_VERSION" -m "<add description here>"
$ git push origin "$RELEASE_VERSION"
```

## Draft a GitHub Release
Visit the [releases page](https://github.com/openservicemesh/osm/releases)
to `Draft a new release` using the tag you just created and pushed.

## Add release notes
In the description section, add information about feature additions, bug fixes, and
any other administrative tasks completed on the repository.

## Build and add binaries
`make release-artifacts` cross compiles binaries for supported platforms and creates platform
specific compressed packages for distribution.
```console
make release-artifacts
```
Upload the `*.tar.gz` and `*.zip` files to the binaries section of the Release.

## Publish the GitHub Release
Click the "Publish release" button.
