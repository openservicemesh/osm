---
title: "Logs"
description: "Logs"
type: docs
---

# Logs
Open Service Mesh (OSM) collects logs that are sent to stdout by default. When enabled, Fluent Bit can collect these logs, process them and send them to an output of the user's choice such as Elasticsearch, Azure Log Analytics, BigQuery, etc.


## Fluent Bit
[Fluent Bit](https://fluentbit.io/) is an open source log processor and forwarder which allows you to collect data/logs and send them to multiple destinations. It can be used with OSM to forward OSM controller logs to a variety of outputs/log consumers by using its output plugins.

OSM provides log forwarding by optionally deploying a Fluent Bit sidecar to the OSM controller using the `--enable-fluentbit` flag during installation. The user can then define where OSM logs should be forwarded using any of the available [Fluent Bit output plugins](https://docs.fluentbit.io/manual/pipeline/outputs).

### Configuring Log Forwarding with Fluent Bit
By default, the Fluent Bit sidecar is configured to simply send logs to the Fluent Bit container's stdout. If you have installed OSM with Fluent Bit enabled, you may access these logs using `kubectl logs -n osm-system <osm-controller-name> -c fluentbit-logger`. This command will also help you find how your logs are formatted in case you need to change your parsers and filters. Once you have tried this out however, we recommend configuring log forwarding to your preferred output for more informative results.

To customize log forwarding to your output, follow these steps and then reinstall OSM with Fluent Bit enabled.

1. Find the output plugin you would like to forward your logs to in [Fluent Bit documentation](https://docs.fluentbit.io/manual/pipeline/outputs). Replace the `[OUTPUT]` section in `fluentbit-configmap.yaml` with appropriate values.

2. The default configuration uses CRI log format parsing. If you are using a kubernetes distribution that causes your logs to be formatted differently, you may need to add a new parser to the `[PARSER]` section and change the `parser` name in the `[INPUT]` section to one of the parsers defined [here](https://github.com/fluent/fluent-bit/blob/master/conf/parsers.conf).

3. The logs are currently filtered to match "error" level logs and multiple filters have been used to cover differences in log formatting on various Kubernetes distros. 
    * To change the log level being filtered on, you can update the `logLevel` value in `values.yaml` to "debug", "info", "warn", "fatal", "panic", "disabled" or "trace". 
    * To view all logs irrespective of log level, you may remove the `[FILTER]` sections. 
    * If you wish to apply further filtering, explore [Fluent Bit filters](https://docs.fluentbit.io/manual/pipeline/filters).
     
4. Once you have updated the Fluent Bit configmap, you can deploy the sidecar during OSM installation using the `--enable-fluentbit` flag. You should now be able to interact with error logs in the output of your choice as they get generated.

5. The `controller_pod_name` key/value pair has been added to the logs to help you query logs in your output by refining results on pod name (see example usage below).

### Example: Using Fluent Bit to send logs to Azure Monitor
Fluent Bit has an Azure output plugin that can be used to send logs to an Azure Log Analytics workspace as follows:
1. [Create a Log Analytics workspace](https://docs.microsoft.com/en-us/azure/azure-monitor/learn/quick-create-workspace)

2. Navigate to your new workspace in Azure Portal. Find your Workspace ID and Primary key in your workspace under Agents management. In `values.yaml`, under `fluentBit`, update the `outputPlugin` to `azure` and keys `workspaceId` and `primaryKey` with the corresponding values from Azure Portal (without quotes). Alternatively, you may replace entire output section in `fluentbit-configmap.yaml` as you would for any other output plugin.

3. Run through steps 3 and 4 above. 

4. Once you run OSM with Fluent Bit enabled, logs will populate under the Logs > Custom Logs section in your Log Analytics workspace. There, you may run the following query to view most recent logs first:
    ```
    fluentbit_CL
    | order by TimeGenerated desc
    ```
5. Refine your log results on a specific deployment of the OSM controller pod:
    ```
    | where controller_pod_name_s == "<desired osm controller pod name>"
    ```

Once logs have been sent to Log Analytics, they can also be consumed by Application Insights as follows:
1. [Create a Workspace-based Application Insights instance](https://docs.microsoft.com/en-us/azure/azure-monitor/app/create-workspace-resource).

2. Navigate to your instance in Azure Portal. Go to the Logs section. Run this query to ensure that logs are being picked up from Log Analytics:
    ```
    workspace("<your-log-analytics-workspace-name>").fluentbit_CL
    ```

You can now interact with your logs in either of these instances.


### Configuring Outbound Proxy Support for Fluent Bit
You may require outbound proxy support if your egress traffic is configured to go through a proxy server. To enable this, make the following changes in `values.yaml`:
    
1. Change `enableProxySupport` to `true`

2. Update the httpProxy and httpsProxy values to `"http://<host>:<port>"`. If your proxy server requires basic authentication, you may include its username and password as: `http://<username>:<password>@<host>:<port>`

> NOTE: Ensure that the [Fluent Bit image tag](https://github.com/openservicemesh/osm/blob/main/charts/osm/values.yaml) is `1.6.4` or greater as it is required for this feature. 
