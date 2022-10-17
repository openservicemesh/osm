# Design Docs

The Open Service Mesh (OSM) project fully-embraces design docs as a vehicle for feature proposals and discussion. This document describes the process for submitting a design doc and getting it accepted.

## Prerequisites

Before creating a design doc, please check the project's [issues](https://github.com/openservicemesh/osm/issues) to see if there's already ongoing effort being done. If you wold like to validate your approach or get some early feedback before starting design doc, consider starting a GitHub [discussion](https://github.com/openservicemesh/osm/discussions) on the OSM repo or asking a question in the #openservicemesh Slack [channel](https://cloud-native.slack.com/archives/C018794NV1C).

## Creating a design doc

Though it's not required, feel free to use the [design doc template](https://docs.google.com/document/d/1qLibGd-s3vVNakQ4e97wjuLnK4L0vSLQKIRedrzVM_k/edit#). If you choose to forgo the design doc template, make sure the same general information is present in your design doc.

## Submitting a design doc

To submit a design doc, submit a PR to this repo [openservicemesh/osm](github.com/openservicemesh/osm) with the title and link to your design doc added to the [index.md](./index.md) file in this directory. When submitting your PR, add the `kind/design-doc` label to it.

## Design doc approval process

OSM uses the [auto-assign GitHub Action](https://github.com/marketplace/actions/auto-assign-action) to automatically assign reviewers to the design doc: 2 maintainers and 2 contributors (see the [Contributor ladder](../../CONTRIBUTOR_LADDER.md) for details). Those who are auto-assigned should prioritize the design doc review the same as any code PR view. Feel free to review the design doc and add yourself as a reviewer even if you aren't auto-assigned! Comments regarding the content of the design doc should be made inline on the design doc itself if the editor supports it (e.g. Google Docs). Once a reviewer finds the design doc to be satisfactory, they should approve the PR on GitHub. Upon the approval of 2 maintainers, the PR will be merged and that design doc will be considered "approved".
