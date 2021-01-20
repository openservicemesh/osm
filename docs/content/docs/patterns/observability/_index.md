---
title: "Observability"
description: "OSM's observability stack includes Prometheus for metrics collection, Grafana for metrics visualization and Fluent Bit for log forwarding."
type: docs
aliases: ["observability"]
---

# Observability

OSM's observability stack includes Prometheus for metrics collection, Grafana for metrics visualization and Fluent Bit for log forwarding to a user-defined endpoint. These are disabled by default but may be enabled during OSM installation with flags `--deploy-prometheus`, `--deploy-grafana` and `--enable-fluentbit`.

## Table of Contents
- [Metrics - Prometheus and Grafana](./metrics)
- [Log forwarding - Fluent Bit](./logs)
