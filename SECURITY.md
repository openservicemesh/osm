# Open Service Mesh Security Policies and Procedures

This document outlines security procedures and general policies for the
Open Service Mesh Open Source projects as found on https://github.com/openservicemesh/osm.

  * [Reporting a Vulnerability](#reporting-a-vulnerability)
  * [Disclosure Policy](#disclosure-policy)

## Reporting a Vulnerability 

The OSM team and community take all security vulnerabilities
seriously. Thank you for improving the security of our open source 
software. We appreciate your efforts and responsible disclosure and will
make every effort to acknowledge your contributions.

Report security vulnerabilities by emailing the OSM security team at:

    cncf-osm@cncf.io

A maintainer will acknowledge your email within 48 hours, and will
send a more detailed response within 72 hours indicating the next steps in 
handling your report. After the initial reply to your report, the team
will endeavor to keep you informed of the progress towards a fix and
full announcement, and may ask for additional information or guidance.

Report security vulnerabilities in third-party modules to the person or 
team maintaining the module.

## Disclosure Policy

When the team receives a security bug report, they will assign it
to someone to be a primary handler. This person will coordinate the fix 
and release process, involving the following steps:

  * Confirm the problem and determine the affected versions.
  * Audit code to find any potential similar problems.
  * Prepare fixes for all releases still under maintenance. These fixes
    will be released as fast as possible.

*Inspired by the [Atomist Security Template](https://github.com/atomist/samples/blob/master/SECURITY.md)*