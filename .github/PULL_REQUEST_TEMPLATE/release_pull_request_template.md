<!--

Use the checklist below to ensure your release PR is complete before marking it ready for review.

-->

- [ ] I have cherry-picked any changes [labelled](https://github.com/openservicemesh/osm/labels) `needs-cherry-pick-vX.Y.Z`
- [ ] I have made all of the following version and patch updates:
  1. Updated the container image tag in charts/osm/values.yaml
  2. Updated the chart **and** app version in charts/osm/Chart.yaml
  3. Updated the default osm-controller image tag in osm cli
  4. Updated the image tags used in the demo manifests
  5. Regenerated the Helm chart README.md

- [ ] I have checked that the base branch for this PR is correct as defined by the release guide
<!--
  If this PR is for updating the release branch, ensure the base branch is `release-vX.Y`.
  If this PR is for making changes on the main branch, ensure the base branch is `main`. 
-->

Is this a release candidate?
