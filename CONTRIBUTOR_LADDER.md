# OSM Contributor Ladder

Hello! We are excited to have you contribute to Open Service Mesh (OSM)! This contributor ladder outlines the different contributor roles within the project, along with the responsibilities and privileges that come with them. Community members generally start at the first levels of the "ladder" and advance up it as their involvement in the project grows. Our project members are happy to help you advance along the contributor ladder.

Each of the contributor roles below is organized into lists of three types of things.
* "Responsibilities" are things that a contributor is expected to do
* "Requirements" are qualifications a contributor needs to meet to be in that role
* "Privileges" are things a contributor on that level is entitled to

## Community Participant

Description: A Community Participant participates in the community and contributes their time, thoughts, etc.

* Responsibilities:
    * Must follow the [CNCF CoC]
* How users can get involved with the community:
    * Participating in community discussions in Github, Slack, and meetings
    * Helping other users
    * Submitting bug reports
    * Trying out new releases
    * Attending community events
    * Talking about the project on social media, blogs, and talks


## Contributor

Description: A Contributor contributes directly to the project and adds value to it. Contributions need not be code. People at the Contributor level may be new contributors, or they may only contribute occasionally.

* Responsibilities include:
    * Following the [CNCF CoC]
    * Following the project [contributing guide]
* Requirements (one or several of the below):
    * Reports and sometimes resolves issues
    * Occasionally submits PRs
    * Contributes to the documentation
    * Regularly shows up at meetings, takes notes
    * Answers questions from other community members
    * Submits feedback on issues and PRs
    * Tests releases and patches and submits reviews
    * Runs or helps run events
    * Promotes the project in public
* Privileges:
    * Invitations to contributor events
    * Eligible to become a Reviewer


## Reviewer

Description: Reviewers are able to review code for quality and correctness on some project areas. They are knowledgeable about both the codebase and software engineering principles.

**Defined by**: *reviewers* entry in the [OWNERS file].

A Reviewer must meet the rights, responsibilities, and requirements of a Contributor, plus:

* Responsibilities include:
    * Following the reviewing guide
    * Reviewing Pull Request for project quality control
    * Reviewing most Pull Requests against their specific areas of ownership
    * Focus on code quality and correctness, including testing and refactoring
    * Prompt response to review requests as per community expectations
    * Resolving test bugs related to project area ownership
* Requirements:
    * Experience as a Contributor for at least 3 months
    * Primary reviewer for at least 5 PRs in their specific areas of ownership
    * Reviewed or merged substantial PRs to the codebase
    * Demonstrated in-depth knowledge of a specific area of the codebase
* Additional Privileges
    * Has GitHub or CI/CD rights to approve pull requests in specific directories
    * Can recommend and review other contributors to become Reviewers

Process of becoming a Reviewer:
1. A current contributor may be self-nominated or nominated by a current Maintainer or Reviewer by opening a PR against the root of the [OSM repository] and adding the nominee to the [OWNERS file] under the *reviewers* entry.  Additionally, if the nomination is for a code owner whose PR approvals are meant to satisfy the PR merge requirements against a specific area of the codebase, the nominee must be added to the [CODEOWNERS file] for their specific area of ownership in the same PR.
2. The nominee will add a comment to the PR testifying that they agree to all requirements of becoming a Reviewer.
3. A majority of the current Maintainers must then approve the PR.


## Maintainer

Description: Maintainers are very established contributors who are responsible for the entire project. As such, they have the ability to approve PRs against any area of the project, and are expected to participate in making decisions about the strategy and priorities of the project. Maintainers focus on a holistic review of contributions: performance, compatibility, adherence to convention, and overall quality.

**Defined by**: *owners* entry in the [OWNERS file].

A Maintainer must meet the rights, responsibilities, and requirements of a Reviewer, plus:

* Responsibilities include:
    * Reviewing and approving PRs that involve multiple parts of the project
    * Is supportive of new and infrequent contributors, and helps get useful PRs in shape to commit
    * Mentoring new Maintainers
    * Writing refactoring PRs
    * Participating in CNCF maintainer activities
    * Determining strategy and policy for the project
    * Participating in, and leading, community meetings
    * Helps run the project infrastructure
* Requirements:
    * Experience as a Reviewer for at least 3 months
    * Deep understanding of the technical goals and direction of the project
    * Deep understanding of the technical domain of the project across multiple areas
    * Sustained contributions to design and direction by doing all of:
      * Authoring and reviewing proposals (GitHub Issues for refactoring, enhancements, or new functionality)
      * Initiating, contributing, and resolving discussions (emails, GitHub issues, meetings)
      * Identifying subtle or complex issues in designs and implementation PRs
    * Is able to exercise judgement for the good of the project, independent of their employer, social circles, or teams
    * Mentors other Contributors and Reviewers
* Additional privileges:
    * Approve PRs to any area of the project
    * Represent the project in public as a Maintainer
    * Communicate with the CNCF on behalf of the project
    * Have a vote in Maintainer decisions

Process of becoming a Maintainer:

1. A current Reviewer may be self-nominated or be nominated by a current Maintainer by opening a PR against the root of the [OSM repository] and adding the nominee to the [OWNERS file] under the *owners* entry. Additionally, if the nomination is for a code owner whose PR approvals are meant to satisfy the PR merge requirements, the nominee must be added to the [CODEOWNERS file] in the same PR.
2. The nominee will add a comment to the PR testifying that they agree to all requirements of becoming a Maintainer.
3. A majority of the current Maintainers must then approve the PR.


## Inactivity

It is important for contributors to be and stay active to set an example and show commitment to the project. Inactivity is harmful to the project as it may lead to unexpected delays, contributor attrition, and a lost of trust in the project.

* Inactivity is measured by:
    * Periods of no contributions for longer than 3 months
    * Periods of no communication for longer than 3 months

* Consequences of being inactive include:
    * Involuntary removal or demotion
    * Being asked to move to Emeritus status


## Involuntary Removal or Demotion

Involuntary removal/demotion of a contributor (Reviewer or Maintainer) happens when responsibilities and requirements aren't being met. This may include repeated pattern of inactivity, extended period of inactivity, a period of failing to meet the requirements of your role, and/or a violation of the [CNCF CoC](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). This process is important because it protects the community and its deliverables while also opens up opportunities for new contributors to step in.


Involuntary removal or demotion is handled through a vote by a majority of the current Maintainers.


## Stepping Down/Emeritus Process

If and when contributors' commitment levels change, contributors can consider stepping down (moving down the contributor ladder) vs moving to emeritus status (completely stepping away from the project).

Contact the Maintainers about changing to Emeritus status, or reducing your contributor level.


## Review Process

- In addition to self nominated and Maintainer proposed nominations, Maintainers will meet quarterly to discuss Role promotions and demotions.
- Prior to nomination or meeting the requirements of a role, Contributors and Reviewers can express their interest in taking on new role or expanding
their ownership. The contributor can be assigned a Maintainer or area owner mentor to help guide them through the requirements for a desired role or area ownership.

## Contact

For inquiries, please reach out to: openservicemesh-maintainers@lists.cncf.io

[OWNERS file]: https://github.com/openservicemesh/osm/blob/main/OWNERS
[CODEOWNERS file]: https://github.com/openservicemesh/osm/blob/main/CODEOWNERS
[OSM repository]: https://github.com/openservicemesh/osm
[contributing guide]: https://github.com/openservicemesh/osm/blob/main/CONTRIBUTING.md
[CNCF CoC]: https://github.com/cncf/foundation/blob/master/code-of-conduct.md