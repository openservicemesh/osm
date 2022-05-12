# Scale testing
## Overview
This folder contains scale tests and related/explicit scale helpers to test OSM.

Scale testing will have several use cases:
- It will seek to qualify and document the behavior of OSM (and related control plane services) under certain loads associated to one or more dimensions (number of services, number of policies, rate of change, etc).
- It will help find, understand and validate soft ceilings and bottlenecks on current design and implementation.
- It should also provide instrumentation to gate at PR level (possibly an on-demand pipeline), in order to find possible performance/scale regressions before merge.
- It should also provide meaningful insight on how the performance and scalability of the overall product offering evolves at each milestone/release.

## Design

Current tests leverage the existing test framework to deploy OSM and repeat a work operation (iteration) till the validation phase of the iteration fails. How and what does the iteration do is up to the test, so implementation is free to scale any resource/s till failure.

The framework provides helpers to track profiling information through the overall test and individual iterations. Some of most relevant metrics are automatically captured during the test, but more exclusive metrics for speicific tests might have to be implemented. 
Gathering these metrics requires a Prometheus instance scraping both the envoy endpoints and the K8s api servers.

The current set of metrics aimed to be automatically tracked across iteration for any set of resources specified in the test are:
- CPU Average loads, for each iteration for each tracked resource.
- RSS footprint and related relative increases per iteration per tracked resource.
- Visual representation of the previous trends, provided by Grafana.
- Control plane profiling (pprof), cpu and mem (Todo)
- Envoy config latency trends (time to create an envoy config by osm-controller, latency increase per pod, test dependent) (Todo)
- Envoy config latency apply trends (from  `SMI apply` to `200` network requests) (Todo)


## Usage examples
Create a test using GinkGo, and import the test framework. As we do for our E2E, we require a separate test per file.
```go
import . "github.com/openservicemesh/osm/tests/framework"

var  _ = Describe("Example Skeleton for scale test", func() {
	Context("ScaleTest", func() {
		// sd to store test state and iteration information
		var  sd *DataHandle

		// WrapUp will call to evaluate and output results on test's on os.Stdout.
		// Additional outputs can be defined through `sd` API
	AfterEach(func() {
		if sd != nil {
			sd.WrapUp()
		}
	})
})
```

Initialize the body of the test, pretty standard from E2E framework semantics:
```go
It("Deploys OSM and scales traffic Splits indefinitely", func() {
	// Install OSM, enable Grafana and Prometheus self-deployment
	t := Td.GetOSMInstallOpts()
	t.DeployGrafana = true
	t.DeployPrometheus = true
	Td.InstallOSM(t)

	// Helpers to get OSM's install handlers, but arbitrary ones can be provided
	pHandle := Td.GetOSMPrometheusHandle()
	gHandle := Td.GetOSMGrafanaHandle()

	// This could/should be called on `Context`, subject to when are resources available
	sd = NewDataHandle(pHandle, gHandle, GetTrackedResources(), GetSaveDashboards())	

	// Repetitive scale loop
	sd.Iterate(func() {
		// Code goes here
	}
})
```

Tracked Resources are defined by labels, and they select the resources which are monitored during the test:
```go
func  GetTrackedResources() []TrackedLabel {
	return []TrackedLabel{
		{
			Namespace: "some-namespace",
			Label: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "osm-controller",
				},
			},
		},
		{
			Namespace: "some-namespace",
			Label: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "osm-prometheus",
				},
			},
		},
	}
}
```

Similarly, Grafana dashboards can be saved upon test exit if Grafana is available. The dashboard prefix/name and the panel Id must be known:
```go
func  GetSaveDashboards() []GrafanaPanel {
	return []GrafanaPanel{
		{
			Filename: "cpu",
			Dashboard: "ABCDEF/mesh-cpu-and-mem",
			Panel: 14,
		},
		{
			Filename: "mem",
			Dashboard: "ABCDEF/mesh-cpu-and-mem",
			Panel: 12,
		},
	}
}
```

## Results
Iteration results will display on test logs to keep an idea of progress.
Only observed resources at this point in time are displayed.
```
-- Successfully completed iteration 8 - took 1m36.700005257s
+------------------------+--------------+---------+
|        Resource        | CpuAvg (96s) | Mem RSS |
+------------------------+--------------+---------+
| osm-cont..p2rb/osm-c.. |         0.44 | 160 MB  |
| osm-prom..n8gt/prome.. |         0.19 | 1.2 GB  |
+------------------------+--------------+---------+
```

Upon `WrapUp` call time, a more comprehensive view of the evolution of resources is displayed at the end of the test, displaying both CPU avg for iteration and RSS footprint at the end, as well as relative increase to previous iteration.

Note that this will show all possible resources that ever appeared during the test that matched any tracked label (so in case of restarted/crashed pods, all metrics and iterations present will also show):
```[AfterEach] ScaleClientServerTrafficSplit
  /home/eserra/src/osm/tests/scale/scale_trafficSplit_test.go:22
+----+---------------------------+-------+-------------------------+-------------------------+
| It |         Duration          | NPods | osm-prom..n8gt/prome..  | osm-cont..p2rb/osm-c..  |
+----+---------------------------+-------+-------------------------+-------------------------+
|  0 | 32.593299891s             |    20 | err / 126 MB            | 0.03 / 46 MB            |
|  1 | 32.54950816s (-0.13%)     |    40 | err / 272 MB (+115.60%) | err / 55 MB (+19.84%)   |
|  2 | 32.608644378s (+0.18%)    |    60 | 0.13 / 376 MB (+38.23%) | err / 66 MB (+20.46%)   |
|  3 | 34.7493147s (+6.56%)      |    80 | 0.14 / 535 MB (+42.28%) | 0.55 / 82 MB (+24.04%)  |
|  4 | 40.077845174s (+15.33%)   |   100 | 0.15 / 625 MB (+16.91%) | 0.66 / 96 MB (+16.16%)  |
|  5 | 38.266130228s (-4.52%)    |   120 | 0.13 / 725 MB (+16.00%) | 0.59 / 102 MB (+7.16%)  |
|  6 | 49.027804296s (+28.12%)   |   140 | 0.20 / 883 MB (+21.72%) | 0.94 / 124 MB (+21.51%) |
|  7 | 56.293969628s (+14.82%)   |   160 | 0.19 / 1.0 GB (+15.16%) | 1.37 / 140 MB (+12.44%) |
|  8 | 1m36.700005257s (+71.78%) |   180 | 0.21 / 1.2 GB (+17.39%) | 0.44 / 160 MB (+14.70%) |
+----+---------------------------+-------+-------------------------+-------------------------+
```
A per-pod RSS footprint average increase is also provided for the runtime of the test:
```
+------------------------+-----------+
|        Resource        | Mem / pod |
+------------------------+-----------+
| osm-prom..n8gt/prome.. | 6.7 MB    |
| osm-cont..p2rb/osm-c.. | 715 kB    |
+------------------------+-----------+
```

Additionally, if Grafana rendering was present and enabled, any added dashboards will be saved in a test folder.
