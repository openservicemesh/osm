#!/usr/bin/env bash

set -euo pipefail

# Generate release notes using the changes between the given tag and its
# predecessor, calculated by git's version sorting. When a stable tag (i.e.
# without a pre-release tag like alpha, beta, etc.) is provided, then the
# previous tag will be the next latest stable tag, skipping any intermediate
# pre-release tags.

# This script will break or produce weird output if:
# - Tags are not formatted in a way that can be interpreted by git tag's --sort=version:refname
# - Pre-release tags other than "alpha", "beta", and "rc" are used.

tag=$1

tags=$(git -c 'versionsort.suffix=-alpha,-beta,-rc' tag -l --sort=version:refname | sed "/^$tag$/q" )
! [[ "$tag" =~ -(alpha|beta|rc) ]] && tags=$(grep -Eve '-(alpha|beta|rc)' <<< "$tags")
prev=$(tail -2 <<< "$tags" | head -1)

changelog=$(git log "$prev".."$tag" --format="* %s %H (%aN)")

# Determine if any CRDs were updated between tags
# CRD upgrades require manually deleting prior CRDs before upgrading Helm chart
crd_changes=$(git diff --name-only "$prev".."$tag" -- charts/osm/crds)
if [[ -z "$crd_changes" ]]; then
   crd_changes="No CRD changes between tags ${prev} and ${tag}"
fi

cat <<EOF
## CRD Updates

$crd_changes

## Changelog

$changelog
EOF
