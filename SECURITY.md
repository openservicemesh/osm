# Open Service Mesh Security Policies and Procedures

This document outlines security procedures and general policies for the
Open Service Mesh open source project as found on https://github.com/openservicemesh/osm.

  * [Reporting a Vulnerability](#reporting-a-vulnerability)
  * [Disclosure Policy](#disclosure-policy)

## Reporting a Vulnerability 

**IMPORTANT: Please do not open public issues on GitHub for security vulnerabilities**

The OSM team and community take all security vulnerabilities
seriously. Thank you for improving the security of our open source 
software. We appreciate your efforts and responsible disclosure and will
make every effort to acknowledge your contributions.

Report security vulnerabilities by emailing the OSM security team at:

    cncf-osm@cncf.io

Please provide the following:

  - Individual's identity and organization
  - Detailed description of the issue and the consequences of the vulnerability
  - Estimation of the attack surface
  - 3rd party software, if any, used with OSM
  - Detailed steps to reproduce the issue

A maintainer will acknowledge your email and send a detailed
response within 3 business days indicating the next steps in 
handling your report. After the initial reply to your report, the team
will endeavor to keep you informed of the progress towards a fix and
full announcement, and may ask for additional information or guidance.

Report potential security issues, or known security issues in a 
third party modules by opening a Github Issue.

### When To Send A Report

If you think you have found a vulnerability in a OSM project.

Report potential security issues, or known security issues in a 
third party modules by opening a Github Issue.

### When Not To Send A Report

* For guidance on securing OSM please open a [Github Issue](https://github.com/openservicemesh/osm/issues/new/choose) or reach out on the OSM Slack Channel within the [CNCF Slack](https://slack.cncf.io)
* For guidance on applying security updates


## Disclosure Policy

When the team receives a security bug report, they will assign it
to someone to be a primary handler. This person will coordinate the fix 
and release process, involving the following steps:

  * Confirm the problem and determine the affected versions.
  * Audit code to find any potential similar problems.
  * Prepare fixes for all releases still under maintenance. These fixes
    will be released as fast as possible.

*Inspired by the [Atomist Security Template](https://github.com/atomist/samples/blob/master/SECURITY.md)*