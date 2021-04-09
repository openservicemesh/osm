---
title: "OSM ConfigMap Troubleshooting"
description: "OSM ConfigMap Troubleshooting Guide"
type: docs
---

# OSM ConfigMap Troubleshooting Guide

## Issues with Validating Webhook
If the validating webhook is not working as expected, below is a command to delete the webhook entirely. Please also [open a GitHub issue](https://github.com/openservicemesh/osm/issues).
```bash
kubectl delete ValidatingWebhookConfiguration $(kubectl get ValidatingWebhookConfigurations --no-headers | grep osm | awk '{print $1}' | grep -E '^osm')
```

## Other Issues
If you're running into issues that are not resolved with the steps above, please [open a GitHub issue](https://github.com/openservicemesh/osm/issues).

