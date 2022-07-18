# OSM Contributor Ladder

Hello! We are excited to have you contribute to Open Service Mesh (OSM)! This contributor ladder outlines the different contributor roles within the project, along with the responsibilities and privileges that come with them. Community members generally start at the first levels of the "ladder" and advance up as their involvement in the project grows. Our project members are happy to help you advance along the contributor ladder.

Each of the roles is organized into the following categories:
* "Responsibilities" list what the role is expected to do
* "Requirements" are qualifications the role needs to meet
* "Privileges" list what the role is entitled to

| Role | Responsibilities | Requirements | Privileges |
| -----| ---------------- | ------------ | -------|
| [Community Participant](#community-participant) | Following the [CNCF Code of Conduct] | --- | --- |
| [Contributor](#contributor) | Following the [CNCF Code of Conduct] and the project [contributing guide] | One or more contributions to the project | Invitations to contributor events |
| [Maintainer](#maintainer) | Active member of the community |<ul><li>Contributor for ≥3 months</li><li> Demonstrates deep understanding of multiple areas of the project</li><li>Exercise good judgement for the project</li><li>Maintain contributions to the project</li></ul> | <ul><li>Has GitHub or CI/CD rights to approve any pull requests</li><li>Represent the project as a Maintainer</li><li>Communicate with CNCF on behalf of the project</li></ul> |

## Community Participant

Description: A Community Participant participates in the community and contributes their time, thoughts, etc.

* Responsibilities:
    * Must follow the [CNCF Code of Conduct]
* How users can get involved with the community:
    * Participating in community discussions in GitHub, Slack, and meetings
    * Helping other users
    * Submitting bug reports
    * Trying out new releases
    * Attending community events
    * Talking about the project on social media, blogs, and talks

## Contributor

Description: A Contributor contributes directly to the project and adds value to it. Contributions need not be code. People at the Contributor level may be new contributors, or they may only contribute occasionally.

* Responsibilities include:
    * Following the [CNCF Code of Conduct]
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
    * Eligible to become a Maintainer

Process of becoming a Contributor:
1. There is no nomination process for this role. Being a contributor entails being a community member who does one or more of the listed requirements.

## Maintainer

Description: Maintainers are established contributors who are responsible for one or more project areas. They have the ability to approve PRs against the project areas they own and are expected to participate in making decisions about the strategy and priorities of the project.

**Defined by**: *owners* entry in the [OWNERS file].

A Maintainer must meet the rights, responsibilities, and requirements of a Contributor, plus:

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
    * Experience as a Contributor for at least 3 months
    * Deep understanding of the technical goals and direction of the project
    * Deep understanding of the technical domain of the project across their areas of ownership
    * Sustained contributions to design and direction by doing all of:
      * Authoring and reviewing proposals (GitHub Issues for refactoring, enhancements, or new functionality)
      * Initiating, contributing, and resolving discussions (emails, GitHub issues, meetings)
      * Identifying subtle or complex issues in designs and implementation PRs
    * Is able to exercise judgement for the good of the project, independent of their employer, social circles, or teams
    * Mentors other Contributors
* Additional privileges:
    * Approve PRs to areas of ownership
    * Represent the project in public as a Maintainer
    * Communicate with the CNCF on behalf of the project
    * Have a vote in Maintainer decisions

Process of becoming a Maintainer:

1. A current contributor may be self-nominated or be nominated by a current Maintainer by opening a PR against the root of the [OSM repository] and adding the nominee to the [OWNERS file] under the *owners* entry. 
   - If the nomination is for a code owner whose PR approvals are meant to satisfy the PR merge requirements against specific areas of the codebase, the nominee must be added to the [CODEOWNERS file] for their specific areas of ownership in the same PR. 
   - A codeowner is defined per the [GitHub definition](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners#about-code-owners). See this [example](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners#example-of-a-codeowners-file) on how to specify an area in the [CODEOWNERS file] .
2. The nominee will add a comment to the PR testifying that they agree to all requirements of becoming a Maintainer.
3. A majority of the current Maintainers must then approve the PR.


## Inactivity

It is important for contributors to be and stay active to set an example and show commitment to the project. Inactivity is harmful to the project as it may lead to unexpected delays, contributor attrition, and a loss of trust in the project.

* Inactivity is measured by:
    * Periods of no contributions for longer than 3 months
    * Periods of no communication for longer than 3 months

* Consequences of being inactive include:
    * Involuntary removal or demotion
    * Being asked to move to Emeritus status


## Involuntary Removal or Demotion

Involuntary removal/demotion of a Maintainer happens when responsibilities and requirements aren't being met. This may include repeated patterns of inactivity, extended periods of inactivity, a period of failing to meet the requirements of your role, and/or a violation of the [CNCF Code of Conduct]. This process is important because it protects the community and its deliverables while also opens up opportunities for new contributors to step in.

Involuntary removal or demotion is handled through a vote by a majority of the current Maintainers.


## Stepping Down/Emeritus Process

If and when a contributor's commitment level changes, they can consider one of the following options: 
1. Stepping down (moving down the contributor ladder)
2. Moving to emeritus status (completely stepping away from the project).

Contact the Maintainers about changing to Emeritus status or reducing your contributor level.


## Review Process

- In addition to self-nominated and Maintainer proposed nominations, Maintainers will meet quarterly to discuss Role promotions and demotions.
- Prior to nomination or meeting the requirements of the role, Contributors that express their interest in being a Maintainer can be assigned a Maintainer to help guide them through the requirements.

## Contact

For inquiries, please reach out to: openservicemesh-maintainers@lists.cncf.io

[OWNERS file]: https://github.com/openservicemesh/osm/blob/main/OWNERS
[CODEOWNERS file]: https://github.com/openservicemesh/osm/blob/main/CODEOWNERS
[OSM repository]: https://github.com/openservicemesh/osm
[contributing guide]: https://github.com/openservicemesh/osm/blob/main/CONTRIBUTING.md
[CNCF Code of Conduct]: https://github.com/cncf/foundation/blob/master/code-of-conduct.md