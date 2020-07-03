# Observability
Open Service Mesh (OSM) generates detailed metrics for all services communicating within the mesh. These metrics provide insights into the behavior of services in the mesh helping users to troubleshoot, maintain and analyze their applications.

As of today OSM collects metrics directly from the sidecar proxies (Envoy). OSM provides rich metrics for incoming and outgoing traffic for all services in the mesh. With these metrics the user can get information about the overall volume of traffic, errors within traffic and the response time for requests.

# Prometheus
To facilitate consistent traffic metrics across all services in the mesh, OSM is deployed with full integration and support for [Prometheus][1].

Each service that is a part of the mesh has an Envoy sidecard and is capable of exposing metrics (proxy metrics) in the Prometheus format. Further every service that is a part of the mesh have Prometheus annotations, which make it possible for the Prometheus server (deployed as a part of OSM's control plane) to scrape the service dynamically. This mechanism automatically enables scraping of metrics whenever a new namespace/pod/service is added to the mesh.

## Querying metrics from Prometheus

### Before you begin 
Ensure that you have followed the steps to run [OSM Demo][2]

### Querying proxy metrics for request count
1. Verify that the Prometheus service is running in your cluster
    - In kubernetes, execute the following command: `kubectl get svc osm-prometheus -n osm-system`
    ![image](https://user-images.githubusercontent.com/59101963/85906800-478b3580-b7c4-11ea-8eb2-63bd83647e5f.png)
2. Open up the Prometheus UI
    - Ensure you are in root of the repository and execute the following script: `./scripts/port-forward-prometheus.sh`
    - Visit the following url [http://localhost:7070][5] in your web browser
3. Execute a Prometheus query
   - In the "Expression" input box at the top of the web page, enter the text: `envoy_cluster_upstream_rq_xx{envoy_response_code_class="2"}` and click the execute button
   - This query will return the successful http requests

Sample result will be:
![image](https://user-images.githubusercontent.com/59101963/85906690-f24f2400-b7c3-11ea-89b2-a3c42041c7a0.png)

# Grafana
By default OSM is deployed with 4 [Grafana][3] dashboards, providing users the ability to understand various metrics related to traffic from a service, pod or workload to another service or the OSM control plane

The 4  grafana dahsboards that come pre-installed with OSM are as follows:
1. OSM Data Plane
   - **OSM Service to Service Metrics**: This dashboard lets you view the traffic metrics from a given source service to a given destination service
   ![image](https://user-images.githubusercontent.com/59101963/85907233-a604e380-b7c5-11ea-95b5-9190fbc7967f.png)
   - **OSM Pod to Service Metrics**: This dashboard lets you investigate the traffic metrics from a pod to all the services it connects/talks to
   ![image](https://user-images.githubusercontent.com/59101963/85907338-03993000-b7c6-11ea-9e63-a4c189bb3080.png)
   - **OSM Workload to Service Metrics**: This dashboard prvides the traffic metrics from a workload (deployment, replicaSet) to all the services it connects/talks to
   ![image](https://user-images.githubusercontent.com/59101963/85907390-26c3df80-b7c6-11ea-98b8-5be96fc954c1.png)
2. OSM Control Plane
   - **OSM Control Plane Metrics**: This dashboard provides traffic metrics from the given service to OSM's control plane
   ![image](https://user-images.githubusercontent.com/59101963/85907465-71455c00-b7c6-11ea-9dea-f6258a1ea8d9.png)

## Visualizing Metrics with Grafana

### Before you begin 
Ensure that you have followed the steps to run [OSM Demo][2]

### Viewing a Grafana dashboard for service to service metrics
1. Verify that the Prometheus service is running in your cluster
   - In kubernetes, execute the following command: `kubectl get svc osm-prometheus -n osm-system`
   ![image](https://user-images.githubusercontent.com/59101963/85906800-478b3580-b7c4-11ea-8eb2-63bd83647e5f.png)
2. Verify that the Grafana service is running in your cluster
   - In kubernetes, execute the following command: `kubectl get svc osm-grafana -n osm-system`
   ![image](https://user-images.githubusercontent.com/59101963/85906847-70abc600-b7c4-11ea-853d-f4c9b188ab9f.png)
3. Open up the Grafana UI
   - Ensure you are in root of the repository and execute the following script: `./scripts/port-forward-grafana.sh`
   - Visit the following url [http://localhost:3000][4] in your web browser
4. The Grafana UI will request for login details, use the following default settings:
   - username: admin
   - password: admin
5. Viewing Grafana dashboard for service to service metrics
   - From the Grafana's dashboards left hand corner navigation menu you can navigate to the OSM Service to Service Dashboard in the folder OSM Data Palne  
   - Or visit the following url [http://localhost:3000/d/OSMs2sMetrics/osm-service-to-service-metrics?orgId=1][6] in your web browser

OSM Service to Service Metrics dashboard will look like:
![image](https://user-images.githubusercontent.com/59101963/85907233-a604e380-b7c5-11ea-95b5-9190fbc7967f.png)

[1]:https://prometheus.io/docs/introduction/overview/
[2]:https://github.com/open-service-mesh/osm/blob/main/demo/README.md
[3]: https://grafana.com/docs/grafana/latest/getting-started/what-is-grafana/
[4]: http://localhost:3000
[5]: http://localhost:7070
[6]: http://localhost:3000/d/OSMs2sMetrics/osm-service-to-service-metrics?orgId=1
