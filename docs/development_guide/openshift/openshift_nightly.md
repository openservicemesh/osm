# OpenShift Nightly Job

The [OpenShift Nightly Job](https://github.com/openservicemesh/osm/actions/workflows/openshift-nightly.yml) tests each day's commits on OpenShift to ensure compatibility.

The nightly job was created so that OSM developers can continue developing quickly by removing the need to individually test their changes on OpenShift clusters prior to merging. The job runs nightly instead of per pull request in order to balance the need for a quick test signal with the monthly costs of an OpenShift cluster.

## OpenShift Cluster
There is one OpenShift cluster that is used by GitHub Actions to run the pipeline. The same cluster is used each night.

### Authentication 
GitHub Actions authenticates with the OpenShift cluster using [oc-login](https://github.com/redhat-actions/oc-login) and a service account token.

## E2E Tests
The nightly job runs all of the end-to-end tests except for Ingress and Upgrade. The tests run serially.

- [Ingress Tracking Issue #3966](https://github.com/openservicemesh/osm/issues/3966)
- [Upgrade Tracking Issue #3852](https://github.com/openservicemesh/osm/issues/3852)

New e2es that fail in the pipeline should be fixed by the test writer. Exceptions will be made for developers who do not have access to an OpenShift cluster.

## Automated Demo
After running the e2es, the nightly job runs the automated demo.

Since the nightly job definition is separate from the PR job definition, there is a risk for drift between the demo configurations that run for each. GitHub Actions templates can be used to keep the two jobs in sync.

- [GitHub Actions Templates Issue #3853](https://github.com/openservicemesh/osm/issues/3853)

## Common Issues

### Insufficient SecurityContextConstraint

New components to the OSM control-plane or other components installed during the e2e tests may need an elevated [SecurityContextConstraint](https://docs.openshift.com/container-platform/4.8/authentication/managing-security-context-constraints.html).

By default, deployments are assigned the Restricted SCC which blocks the scheduling of pods that require hostPaths, privileged escalation, etc. You can check if a deployment's SCC is insufficient by checking the events for the ReplicaSet.

- SCCs can be added to the yaml file for any Role or ClusterRole with the following syntax:
    ```
    {{- if (.Capabilities.APIVersions.Has "security.openshift.io/v1") }}
    - apiGroups: ["security.openshift.io"]
        resourceNames: [<SCC name>]
        resources: ["securitycontextconstraints"]
        verbs: ["use"]
    {{- end }}
    ```
- SCCs can be added via the commandline using the `oc` binary
    `oc adm policy add-scc-to-user <scc name> -z <service account name> -n <namespace>`
- SCCs can be added in the e2e framework using [AddOpenShiftSCC](https://github.com/openservicemesh/osm/blob/abdaefcc42bd9ef6291653f4db2820cb3617e890/tests/framework/common.go#L1438)

### Incompatible Security Contexts
When adding a component to OSM with a security context, it may be incompatible with OpenShift. 

For example, the [restricted security context](https://github.com/openservicemesh/osm/blob/abdaefcc42bd9ef6291653f4db2820cb3617e890/charts/osm/templates/_helpers.tpl#L20) defined in the Helm templates have incompatible user and group values for OpenShift. That is because the acceptable range for those values are dependent on how the namespace is configured for the SCC associated with the deployment. 

These ranges differ per namespace so they cannot be configured programatically.

Ensure that the security contexts are only specified for non OpenShift clusters using Helm conditionals:
```
{{- if not (.Capabilities.APIVersions.Has "security.openshift.io/v1") }}/
   ...
{{- end }}
```

### Port Conflicts
Communication between test applications on port 80 will fail as OpenShift by default reserves port 80 for its own infrastructure. 

Test applications should therefore use a different port than port 80.

### iptables
OpenShift 4 requires that any container programming iptables has privileged escalation.
The NET_ADMIN capability is not sufficient. The errors that result from trying to program iptables without privilege can be misleading (such as claiming there are syntax errors), so ensure that the container is privileged before investigating further (and ensure the privileged SCC is added to any mesh applications).

For OSM on OpenShift, the init container must run as privileged. Instructions to enable this feature can be found in the [OSM OpenShift documentation](https://release-v0-9.docs.openservicemesh.io/docs/install/#openshift).
