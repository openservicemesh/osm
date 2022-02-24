# Release process

## Frequency

Open Service Mesh will target a release every quarter. These releases will be targeted in the first half of February, May, August, and November. These target release months are subject to change if a release is pushed. In other words, if a release misses February and is pushed out in March then the next release will target June instead of May.

## Process for requesting a change

If you would like to see a new feature added to Open Service Mesh, please [open an issue](https://github.com/openservicemesh/osm/issues). This issue will be discussed and, if approved, prioritized in the backlog.

## Creating a new release

When there is a new release being planned and developed, a [milestone](https://github.com/openservicemesh/osm/milestones) will be created. This new milestone will have issues added to it that will be prioritized from the [vFuture milestone](https://github.com/openservicemesh/osm/milestone/32). Once the release milestone is complete, a new release of Open Service Mesh will be created and this milestone will be closed. At that moment, the next release milestone will be created and items from vFuture will be added.

## Larger projects

For larger projects where development will likely span multiple releases we should have a separate feature milestone created. This feature milestone should have a projected end date. Once a release milestone is opened up that has an end date that includes the feature milestone end date, this feature should be considered to be included in the next release.

## Backlog handling

The progression of approved work items will go through this lifecycle:

- Added to the [vFuture milestone](https://github.com/openservicemesh/osm/milestone/32) when approved
- Added to the [next release milestone](https://github.com/openservicemesh/osm/milestones) when set for the next release
- Added to the [Todo](https://github.com/orgs/openservicemesh/projects/1) column when prioritized for "next work"
- Moved to the **In progress** column when being actively worked on
- Moved to the **Done** column when completed

At the end of this lifecycle it will be included in the next release of OSM.

## Release Candidates

Release candidates (RCs) should be created before each significant release so final testing can be performed. RCs are tagged as `vX.Y.Z-rc.W`. After the following steps have been performed to publish the RC, perform any final testing with the published release artifacts for about one week.

If issues are found, submit patches to the RC's release branch and create a new RC with the tag `vX.Y.Z-rc.W+1`. Apply those same patches to the `main` branch. Repeat until the release is suitably stable.
