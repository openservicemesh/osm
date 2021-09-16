---
title: "Automerge Pull Requests"
description: "How to use labels and commands to automerge/autorebase your pull request."
type: docs
---

OSM uses [Mergify](https://docs.mergify.io/) to automatically merge (automerge), automatically squash and merge (autosquash), and automatically rebase (autorebase) pull requests.

# Automerge
A pull request will be automerged via a merge commit if it meets the following criteria:
 - Has the `automerge` label
 - Does not have the `wip` label
 - Does not have the `do-not-merge/hold` label
 - Successfully completed all checks
 - Has at least 2 maintainer approvals
 - Base branch is either main or a release branch
If the pull request has an `automerge` label, the OSM-PR-bot will also autorebase the pull request if the PR branch goes out-of-date.
> Note: Pull requests that are paths-ignore cannot be merged automatically.

A pull request will be autosquashed (commits will be squashed then merged) if it meets the following criteria:
 - Has the `automerge-squash` label
 - Does not have the `wip` label
 - Does not have the `do-not-merge/hold` label
 - Successfully completed all checks
 - Has at least 2 maintainer approvals
 - Base branch is main branch
 If the pull request has an `automerge-squash` label, the OSM-PR-bot will also autorebase the pull request if the PR branch goes out-of-date.
> Note: Pull requests that are paths-ignore cannot be merged automatically.

# Autorebase
A pull request will be autorebased only and not automerged/autosquashed if it has the `autorebase` label, as long as there are no conflicts. The rebase action will be completed by OSM-PR-bot.

# In Pull Request Comments
Some Mergify commands can also be triggered via comments on the pull request.
- `@Mergifyio refresh` will re-evaluate the rules
- `@Mergifyio rebase` will rebase this PR on its base branch
- `@Mergifyio backport <destination>` will [backport](https://docs.mergify.io/actions/backport.html) this PR on <destination> branch