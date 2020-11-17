# Observability

OSM's observability stack includes Prometheus for metrics collection, Grafana for metrics visualization and Fluent Bit for log forwarding to a user-defined endpoint. These are disabled by default but may be enabled during OSM installation with flags `--deploy-grafana`, `--deploy-grafana` and `--enable-fluentbit`.

## Table of Contents
- [Metrics - Prometheus and Grafana](/docs/patterns/observability/metrics.md)
- [Log forwarding - Fluent Bit](/docs/patterns/observability/logs.md)
