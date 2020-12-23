# Contributing Guidelines
The OSM project accepts contributions via GitHub pull requests. This document outlines the requirements for contributing to this project.

## Sign Your Work

The sign-off is a simple line at the end of the explanation for a commit. All commits needs to be
signed. Your signature certifies that you wrote the patch or otherwise have the right to contribute
the material. The rules are pretty simple, if you can certify the below (from
[developercertificate.org](https://developercertificate.org/)):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.

Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

Then you just add a line to every git commit message:

    Signed-off-by: Joe Smith <joe.smith@example.com>

Use your real name (sorry, no pseudonyms or anonymous contributions.)

If you set your `user.name` and `user.email` git configs, you can sign your commit automatically
with `git commit -s`.

Note: If your git config information is set properly then viewing the `git log` information for your
 commit will look something like this:

```
Author: Joe Smith <joe.smith@example.com>
Date:   Thu Feb 2 11:41:15 2018 -0800

    Update README

    Signed-off-by: Joe Smith <joe.smith@example.com>
```

Notice the `Author` and `Signed-off-by` lines match. If they don't your PR will be rejected by the
automated DCO check.

## Roadmap
We use [GitHub Project Boards](https://github.com/openservicemesh/osm/projects) to help give a high level overview and track what work is going on and what stage it is in. If you want an idea of our roadmap and prioritization, this is the best place to find that information.

## Issues
If a bug is found, need clarification, or want to propose a feature, submit an issue using [GitHub issues](https://github.com/openservicemesh/osm/issues).

## Milestones
We use [GitHub Milestones] to track progress of releases. A milestone contains a set of GitHub issues we've agreed to complete for the release. A milestone will be given a semantic version and a soft `due date`. We will cut a [GitHub release](https://github.com/openservicemesh/osm/releases) once all issues tagged in the milestone are closed or moved to the next milestone.

## Semantic Versioning
This project's releases will follow [semantic versioning](https://semver.org/) for labeling releases to maintain backwards compatibility. Features and functionality may be added in major (x.0.0) and minor (0.x.0) releases. Bug fixes may be added in patch releases (0.0.x). We will follow semantic versioning to guarantee features, flags, functionality, public APIs will be backwards compatible as the versioning scheme suggests.

## Pull Requests and Commit Messages
Open a [GitHub pull request] to make a contribution.
  - Each pull request must have at least two LGTM from a core maintainer.
  - If the person who opened the pull request is a core maintainer, then only that person is expected to merge once it has the necessary LGTMs/reviews. Another maintainer can merge the pull request at their discretion if they feel the pull request must be merged urgently.
  - Commits should be squashed before merging to main.
  - Commits should follow the style guideline outlined below.
  - Pull requests not ready for review should be created as drafts.
  - Pull requests that are a work in progress should be labeled with the `wip` label.
  - Pull requests that shouldn't be merged should be labeled with the `do-not-merge/hold` label.

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
