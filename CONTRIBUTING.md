# Contributing Guidlines
The OSM project accepts contributions via GitHub pull requests. This document outlines the requirements for contributing to this project.

## Contributor License Agreement

This repository is governed under a [Contributor License Agreement](https://cla.opensource.microsoft.com/open-service-mesh/osm). All PR submitters must accept the CLA before their contributions can be merged.

## Roadmap
We use [GitHub Project Boards](https://github.com/open-service-mesh/osm/projects) to help give a high level overview and track what work is going on and what stage it is in. If you want an idea of our roadmap and prioritization, this is the best place to find that information.

## Issues
If a bug is found, need clarification, or want to propose a feature, submit an issue using [GitHub issues](https://github.com/open-service-mesh/osm/issues).

## Milestones
We use [GitHub Milestones] to track progress of releases. A milestone contains a set of GitHub issues we've agreed to complete for the release. A milestone will be given a semantic version and a soft `due date`. We will cut a [GitHub release](https://github.com/open-service-mesh/osm/releases) once all issues tagged in the milestone are closed or moved to the next milestone.

## Semantic Versioning
This project's releases will follow [semantic versioning](https://semver.org/) for labeling releases to maintain backwards compatibility. Features and functionality may be added in major (x.0.0) and minor (0.x.0) releases. Bug fixes may be added in patch releases (0.0.x). We will follow semantic versioning to guarantee features, flags, functionality, public APIs will be backwards compatible as the versioning scheme suggests.

## Pull Requests and Commit Messages
Open a [GitHub pull request] to make a contribution.
  - Each pull request must have at least one LGTM from a core maintainer.
  - Large pull requests must have two LGTMs from two core maintainers. It is at the descretion of the author to deem their the size of their PR as `large` or `small` via labels and to ensure that it has received a sufficient number of reviews.
  - If the person who opened the pull request is a core maintainer, then only that person is expected to merge once it has the necessary LGTMs/reviews.
  - Commits should be squashed before merging or upon merging to main
  - Commits should follow the style guideline outlined below.

  ### Commit Style Guideline
  We follow a rough convention for commit messages borrowed from [Deis](https://github.com/deis/deis/blob/master/CONTRIBUTING.md#commit-style-guideline). This is an example of a commit:
  ```
  feat(scripts/test-cluster): add a cluster test command

  Adds test experience where there was none before
  and resolves #1213243.
  ```

  This is a more formal template:
  ```
  {type}({scope}): {subject}
  <BLANK LINE>
  {body}
  ```

  The `{scope}` can be anything specifying the place of the commit change. Use `*` to denote multiple areas (i.e. `scripts/*: refactored lots of stuff`).

  The `{subject}` needs to use imperative, present tense: `change` not `changed` nor `changes`. The first letter should not be capitalized, and there is no dot (`.`) at the end.

  Just like the `{subject}`, the message `{body}` needs to be in the present tense and includes the motivation for the change as well as a contrast with the previous behavior. The first letter in the paragraph must be capitalized. If there is an issue number associate with the chunk of work that should be mentioned in this section as well so that it may be closed once the PR with this commit is merged.

Any line of the commit message cannot be longer than 72 characters, with the subject line limited to 50 characters. This allows the message to be easier to read on GitHub as well as in various git tools.

The allowed types for `{type}` are as follows:
```
feat -> feature
fix -> bug fix
docs -> documentation
style -> formatting
ref -> refactoring code
test -> adding missing tests
chore -> maintenance
```
