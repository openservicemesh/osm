--- 
 title: "Scale and limits" 
 description: "Documentation regarding OSM's current scale and limitations" 
 type: docs 
--- 
# Scale and limits
This document tracks current scale and limitations known in OSM.

## Considerations
The scale limits documented here have to be put in light of current architecture. 
Current architecture relies on a global broadcast mechanism that does not account for proxy configuration deltas, therefore all proxy configurations are computed and pushed upon any change.

## Testing and measures
We currently hold a single test which attempts to scale infinitely a topology subset, test proper traffic configuration between the new pods/services being deployed in the iteration, and stop if any failure is seen.
It is also acknowledged that some of the scale constraints need to be addressed before it even makes sense to proceed with any additional scale testing, hence the lack of additional test scenarios.

## Test
The test was run in different OSM form factors, factoring in different amounts of RAM/CPU, to better qualify potential limits in case any of those were to be a constraint upon deployment.

### Test details:
- Commit-id: 4381544908261e135974bb3ea9ff6d46be8dbd56 (5/13/2021)
- 10 Node (Kubernetes v1.20, nodes: 4vcpu 16Gb)
- Envoy proxy log level Error
- 2048 bitsize RSA keys 
- OSM controller
  - Log level Error
  - Using default max 1.5 CPU
  - Using Max Memory 1GB
- OSM Injector
  - Log level Error
  - Using default max 0.5 CPU
  - Using default max 64MB Memory
- HTTP debug server disabled on OSM
- Test topology deploys each iteration:
	- 2 clients
		- 5 replicaset per client
	- 5 server services
		- 2 replicaset per server service
	- 1 TrafficSplit between the 5 server services (10 pods backed)
	- Total of 20 pods each iteration. 10 client pods will REST GET the Traffic split.
	- Correctness is ensured. It is checked that all TrafficSplit server members are eventually reached.
- Test timeout for network correctness: 150 seconds

## Assessment and Limits
*Note: Assuming proxy per pod, so pod/proxies can be used interchangeably.*

Test failed at around 1200 pods, with kubernetes unable to bring up in time a pod in the mesh.
### CPU
#### OSM Controller
- **1vcpu per 700 proxies**, giving more cpu does not scale linearly (m<1) with current architecture; horizontal scaling should be considered to increase supported mesh size. 
- Network settlment times vary from **<10s with no pods to +2min at 1000 pods**. 

<p align="center">
  <img src="../images/scale/prox.png" width="750" height="120"/>
</p>
<center><i>Fig 1. Function of number of proxies onboarded during the test. Test period is ~1h.</i></center><br>

<p align="center">
  <img src="../images/scale/cpu.png" width="750" height="225"/>
</p>
<center><i>Fig 2. Function of CPU load; OSM in yellow, CPU time for each iteration (spike) increases in time per iteration.</i></center><br>

#### ADS Performance
- With the recent ADS pipelining changes in OSM, it is ensured not too many ADS updates are scheduled for busy work at the same time, ensuring low times as granted by the available CPU.
This yields more deterministic results, with all updates always under sub 0.10s window, and serialization of number of events as opposed to arbitrary scheduling from Golang.
<p align="center">
  <img src="../images/scale/histogram.png" width="1250" height="225"/>
</p>
<center><i>Fig 3. Histogram of ADS computation timings. (All verticals) </i></center><br>

- The number of XDS updates over the test grows additively with any current number of onboarded proxies, each iteration occupying more time until basically iterations overlap, given the rate at which OSM can compute ADS updates.

#### OSM Injector
- Injector can handle onboarding 20 pods concurrently per 0.5cpu, with rather stable times to create the 2048-bit certificates and webhook handling staying regularly below 5s, with some outliers in the 5-10s and in very limited occasions in the 10-20s (and probably closer to 10).
- Since 99% of the webhook handling time happens in the RSA certificate creation context, injector should scale rather linearly with added vcpu.
  
<p align="center">
  <img src="../images/scale/sidecar-inj.png" width="850" height="220"/>
</p>
<center><i>Fig 4. Heatmap of sidecar injection timings / webhook handling times. </i></center><br>

#### Prometheus
- Our control plane qualification testing has disabled envoy scraping for the time being.
- Scraping the control plane alone, requires around 0.25vcpu per 1000 proxies (given number of metrics scraped and scrape interval used), see in orange Fig 2. 

### Memory
#### OSM Controller
Memory per pod/envoy onboarded in the network is calculated after the initial snapshot with nothing onboarded on the mesh is seen to take into account standalone memory used by OSM.
- Memory (RSS) in controller: **600~800KB per proxy**

<p align="center">
  <img src="../images/scale/mem.png" width="800" height="250"/>
</p>
<center><i>Fig 5. Function of Memory (RSS) in use during the test.</i></center><br>

#### OSM Injector
OSM injector doesn't store any intermediate state per pod, so it has no immediate memory scalability constraints at this scale order of magnitude.

#### Prometheus
- Our control plane qualification testing has disabled envoy scraping for the time being.
- Prometheus shows a memory increase per proxy of about **~0.7MB per proxy** to handle the metric listed by OSM metrics.
