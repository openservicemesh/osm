#!/usr/bin/env bash

set -euo pipefail

# Generate release notes using the changes between the given tag and its
# predecessor, calculated by git's version sorting. When a stable tag (i.e.
# without a pre-release tag like alpha, beta, etc.) is provided, then the
# previous tag will be the next latest stable tag, skipping any intermediate
# pre-release tags.

# This script will break or produce weird output if:
# - Tags are not formatted in a way that can be interpreted by git tag's --sort=version:refname
# - Pre-release tags other than "nightly", "alpha", "beta", and "rc" are used.

tag=$1

# No release notes for nightlies
if [[ "$tag" =~ "nightly" ]]; then
  exit 0
fi

tags=$(git tag | tr - \~ | sort -V | tr \~ - | sed "/^$tag$/q" )
! [[ "$tag" =~ -(alpha|beta|rc) ]] && tags=$(grep -Eve '-(alpha|beta|rc)' <<< "$tags")
prev=$(tail -2 <<< "$tags" | head -1)

changelog=$(git log "$prev".."$tag" --no-merges --format="* %s %H (%aN)")

# Determine if any CRDs were updated between tags
# CRD upgrades require manually deleting prior CRDs before upgrading Helm chart
crd_changes=$(git diff --name-only "$prev".."$tag" -- charts/osm/crds)
if [[ -z "$crd_changes" ]]; then
   crd_changes="No CRD changes between tags ${prev} and ${tag}"
fi

cat <<EOF
## Notable Changes

## Deprecation Notes

## CRD Updates

$crd_changes

## Changelog

$changelog
EOF
