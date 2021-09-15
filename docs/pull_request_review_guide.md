# Pull Request Review Guide

This document outlines how to review a pull request and best practices and tips for reviewing pull requests in the OSM repository.
Reviews are welcomed from everyone although each pull request (PR) needs at least two approvals from core maintainers from the
repository before it can be merged into the main branch.

## How to Review a Pull Request

Github outlines how to review a pull request [here](https://docs.github.com/en/github/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/reviewing-proposed-changes-in-a-pull-request) and OSM does not deviate from this standard process.

## Best Practices and Tips for Reviewing Pull Requests

- Leave comments if the PR author can make adjustments to write better Go according to the [Go style guide](https://golang.org/doc/effective_go).
- Ask questions if the code does not make sense to you.
- Make sure each PR includes the appropriate unit tests and e2e tests and read the tests to make sure you understand what the author is changing and what is the expected behavior of the code.
- Although squashing commits [is not required](/CONTRIBUTING.md/#merging-pull-requests), remind the author to squash commits into a single commit when it makes sense. Each commit merged into the main branch should be able to pass all CI tests.
- If the PR author is not a core maintainer and you are a core maintainer, merge the pull request when it has the required number of approvals.
- Add to this list of best practices and tips
