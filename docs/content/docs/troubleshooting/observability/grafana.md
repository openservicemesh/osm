---
title: "Troubleshoot Grafana"
description: "How to fix common issues with OSM's Grafana integration"
type: docs
---

## Grafana is unreachable

If a Grafana instance installed with OSM can't be reached, perform the following steps to identify and resolve any issues.

1. Verify a Grafana Pod exists.

    When installed with `osm install --set=OpenServiceMesh.deployGrafana=true`, a Grafana Pod named something like `osm-grafana-7c88b9687d-tlzld` should exist in the namespace of the other OSM control plane components which named `osm-system` by default.

    If no such Pod is found, verify the OSM Helm chart was installed with the `OpenServiceMesh.deployGrafana` parameter set to `true` with `helm`:

    ```console
    $ helm get values -a <mesh name> -n <OSM namespace>
    ```

    If the parameter is set to anything but `true`, reinstall OSM with the `--set=OpenServiceMesh.deployGrafana=true` flag on `osm install`.

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

    ```console
    $ osm dashboard
    [+] Starting Dashboard forwarding
    [+] Issuing open browser http://localhost:3000
    ```

    Login (default username/password is admin/admin) and navigate to the [data source settings](http://localhost:3000/datasources). For each data source that may not be working, click it to see its configuration. At the bottom of the page is a  "Save & Test" button that will verify the settings.

    If an error occurs, verify the Grafana configuration to ensure it is correctly pointing to the intended Prometheus instance. Make changes in the Grafana settings as necessary until the "Save & Test" check shows no errors:

    ![Successful verification](https://user-images.githubusercontent.com/5503924/112394171-7e419e00-8cb9-11eb-99fc-3343c6b9fbbd.png)

    More details about configuring data sources can be found in [Grafana's docs](https://grafana.com/docs/grafana/latest/administration/provisioning/#data-sources).

For other possible issues, see [Grafana's troubleshooting documentation](https://grafana.com/docs/grafana/latest/troubleshooting/).
