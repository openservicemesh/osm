# Logs
Open Service Mesh (OSM) collects logs that are sent to stdout by default. When enabled, Fluent Bit can collect these logs, process them and send them to an output of the user's choice such as Elasticsearch, Azure Log Analytics, BigQuery, etc.


## Fluent Bit
[Fluent Bit](https://fluentbit.io/) is an open source log processor and forwarder which allows you to collect data/logs and send them to multiple destinations. It can be used with OSM to forward OSM controller logs to a variety of outputs/log consumers by using its output plugins.

OSM provides log forwarding by optionally deploying a Fluent Bit sidecar to the OSM controller using the `--enable-fluentbit` flag during installation. The user can then define where OSM logs should be forwarded using any of the available [Fluent Bit output plugins](https://docs.fluentbit.io/manual/v/1.4/pipeline/outputs).

### Configuring Log Forwarding with Fluent Bit
By default, the Fluent Bit sidecar is configured to simply send logs to the Fluent Bit container's stdout. If you have installed OSM with Fluent Bit enabled, you may access these logs using `kubectl logs -n osm-system <osm-controller-name> -c fluentbit-logger`. However, we recommend configuring log forwarding to your preferred output for more informative results.

You may configure log forwarding to an output by following these steps _before_ you install OSM.

1. Define the output plugin you would like to forward your logs to in the existing `fluentbit-configmap.yaml` file by replacing the `[OUTPUT]` section with your chosen output as described by Fluent Bit documentation [here](https://docs.fluentbit.io/manual/v/1.4/pipeline/outputs).

2. The default configuration uses CRI log format parsing. If you are using a kubernetes distribution that causes your logs to be formatted differently, you may need to update the `[PARSER]` section and the `parser` name in the `[INPUT]` section to one of the parsers defined [here](https://github.com/fluent/fluent-bit/blob/master/conf/parsers.conf).

3. To view all logs irrespective of log level, you may remove the `[FILTER]` section. To change the log level being filtered on, you can update the "error" value below to "debug", "info", "warn", "fatal", "panic" or "trace":
   ```    
   [FILTER]
         name       grep
         match      *
         regex      message /"level":"error"/
   ```

4. Once you have updated the Fluent Bit configmap, you can deploy the sidecar during OSM installation using the `--enable-fluentbit` flag. You should now be able to interact with error logs in the output of your choice as they get generated.

### Example: Using Fluent Bit to send logs to Azure Monitor
Fluent Bit has an Azure output plugin that can be used to send logs to an Azure Log Analytics workspace as follows:
1. [Create a Log Analytics workspace](https://docs.microsoft.com/en-us/azure/azure-monitor/learn/quick-create-workspace)

2. Navigate to your new workspace in Azure Portal. Find your Workspace ID and Primary key in your workspace under Advanced settings > Agents management. Locally, in `fluentbit-configmap.yaml`, update the `[OUTPUT]` section with those values as follows:
   ```
   [OUTPUT]
         Name        azure
         Match       *
         Customer_ID <Log Analytics Workspace ID>
         Shared_Key  <Log Analytics Primary key> 
   ```

3. Run through steps 3 and 4 above. 

4. Once you run OSM with Fluent Bit enabled, logs will populate under the Logs > Custom Logs section in your Log Analytics workspace. There, you may run the following query to view most recent logs first:
    ```
    fluentbit_CL
    | order by TimeGenerated desc
    ```

Once logs have been sent to Log Analytics, that can also be consumed by Application Insights as follows:
1. [Create a Workspace-based Application Insights instance](https://docs.microsoft.com/en-us/azure/azure-monitor/app/create-workspace-resource).

2. Navigate to your instance in Azure Portal. Go to the Logs section. Run this query to ensure that logs are being picked up from Log Analytics:
    ```
    workspace("<your-log-analytics-workspace-name>").fluentbit_CL
    ```

You can now interact with your logs in either of these instances.

