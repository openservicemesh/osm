---
title: "Issue Triage"
description: "Issue Triage Guide"
type: docs
---

# Issue Triage Guide

This guide describes how to triage GitHub issues on the [issue queue](https://github.com/openservicemesh/osm/issues) for Open Service Mesh (OSM).

## Steps for Triage
1. Identify the cause of the issue as explained [below](#identifying-the-cause-of-the-issue)
1. Tag the issue with relevant labels. Relevant labels would include what area of the codebase the issue is most aligned with (ex. cli) or if it is a bug, label the issue with the bug label
1. Prioritize the issue with a P1, P2 or P3 label (if possible) where:
    - P1: product cannot ship. Should be addressed as soon as possible. Eg: high priority bug
    - P2: Product cannot ship, but it does not need to be addressed immediately. Eg: high priority feature, lower priority bug
    - P3: Resolution of the work item is optional based on resources, time, and risk. Eg: a nice to have feature
1. If you have time and can resolve the issue, resolve it. If not, put links to the code to where you see the problem arising
1. If the issue looks like it is a bug that needs to be resolved immediately, either fix the issue or reach out to the core maintainers to see if you can assign the issue to someone for follow up


## Identifying the cause of the issue
- If it's a bug: 
    - Reproduce the issue to make sure it is an actual issue
    - Label it with the `bug` label
    - If possible, fix it or else reference links to the code 

- If it's a question: 
    - Ask around, go through docs, etc.
    - Label it with the `question` label
    - Respond to the question

- If it's a feature request 
    - Ask/understand the use cases
    - Label it with the `Improvement/Feature Request` label and add other appropriate labels

