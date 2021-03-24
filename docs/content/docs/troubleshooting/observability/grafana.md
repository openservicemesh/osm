---
title: "Troubleshoot Grafana"
description: "How to fix common issues with OSM's Grafana integration"
type: docs
---

## Grafana is unreachable

If a Grafana instance installed with OSM can't be reached, perform the following steps to identify and resolve any issues.

1. Verify a Grafana Pod exists.

    When installed with `osm install --deploy-grafana`, a Grafana Pod named something like `osm-grafana-7c88b9687d-tlzld` should exist in the namespace of the other OSM control plane components which named `osm-system` by default.

    If no such Pod is found, verify the OSM Helm chart was installed with the `OpenServiceMesh.deployGrafana` parameter set to `true` with `helm`:

    ```console
    $ helm get values -a <mesh name> -n <OSM namespace>
    ```

    If the parameter is set to anything but `true`, reinstall OSM with the `--deploy-grafana` flag on `osm install`.

1. Verify the Grafana Pod is healthy.

    The Grafana Pod identified above should be both in a Running state and have all containers ready, as shown in the `kubectl get` output:

    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl get pods -n osm-system -l app=osm-grafana
    NAME                           READY   STATUS    RESTARTS   AGE
    osm-grafana-7c88b9687d-tlzld   1/1     Running   0          58s
    ```

    If the Pod is not showing as Running or its containers ready, use `kubectl describe` to look for other potential issues:

    ```console
    $ # Assuming OSM is installed in the osm-system namespace:
    $ kubectl describe pods -n osm-system -l app=osm-grafana
    ```

    Once the Grafana Pod is found to be healthy, Grafana should be reachable.

## Dashboards show no data in Grafana

If data appears to be missing from the Grafana dashboards, perform the following steps to identify and resolve any issues.

1. Verify Prometheus is installed and healthy.

    Because Grafana queries Prometheus for data, ensure Prometheus is working as expected. See the [Prometheus troubleshooting guide](./prometheus) for more details.

1. Verify Grafana can communicate with Prometheus.

    Start by opening the Grafana UI in a browser:

    ```
    $ osm dashboard
    [+] Starting Dashboard forwarding
    [+] Issuing open browser http://localhost:3000
    ```

    Login (default username/password is admin/admin) and navigate to the "Explore" page linked on the left side of the home page. Ensure "Prometheus" is selected as the data source at the top of the page. Then, issue any query, such as `envoy_cluster_upstream_rq_xx`.

    If an error occurs, verify the Grafana configuration to ensure it is correctly pointing to the intended Prometheus instance. Specifically for the Grafana deployed by OSM, check the configured data source:

    ```
    $ # Assuming OSM is installed in the osm-system namespace
    $ kubectl get configmaps -n osm-system osm-grafana-datasources -o jsonpath='{.data.prometheus\.yaml}'
    # config file version
    apiVersion: 1

    # list of datasources that should be deleted from the database
    deleteDatasources:
      - name: Prometheus
        orgId: 1

    # list of datasources to insert/update depending
    # whats available in the database
    datasources:
      # <string, required> name of the datasource. Required
      - name: Prometheus
        # <string, required> datasource type. Required
        type: prometheus
        # <string, required> access mode. direct or proxy. Required
        access: proxy
        # <int> org id. will default to orgId 1 if not specified
        orgId: 1
        # <string> url
        url: http://osm-prometheus.osm-system.svc:7070
        version: 1
        # <bool> allow users to edit datasources from the UI.
        editable: true
    ```

    More details about configuring data sources can be found in [Grafana's docs](https://grafana.com/docs/grafana/latest/administration/provisioning/#data-sources).

For other possible issues, see [Grafana's troubleshooting documentation](https://grafana.com/docs/grafana/latest/troubleshooting/).
