package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test local rate limiting",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 3,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("HTTP request rate limiting", func() {
			testRateLimtiting()
		})
	})

func testRateLimtiting() {
	const sourceName = "client"
	const destName = "server"
	var ns = []string{sourceName, destName}

	It("Tests rate limiting of traffic from client pod -> service", func() {
		// Install OSM
		Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

		// Create Test NS
		for _, n := range ns {
			Expect(Td.CreateNs(n, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, n)).To(Succeed())
		}

		// Get simple pod definitions for the HTTP server
		svcAccDef, podDef, svcDef, err := Td.GetOSSpecificHTTPBinPod(destName, destName)
		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(destName, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destName, podDef)
		Expect(err).NotTo(HaveOccurred())
		dstSvc, err := Td.CreateService(destName, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destName, 90*time.Second, 1, nil)).To(Succeed())

		srcPod := setupSource(sourceName, false)

		By("Allow traffic via SMI policies")
		// Deploy allow rule client->server
		httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
			SimpleAllowPolicy{
				RouteGroupName:    "routes",
				TrafficTargetName: "test-target",

				SourceNamespace:      sourceName,
				SourceSVCAccountName: srcPod.Spec.ServiceAccountName,

				DestinationNamespace:      destName,
				DestinationSvcAccountName: svcAccDef.Name,
			})

		_, err = Td.CreateHTTPRouteGroup(destName, httpRG)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreateTrafficTarget(destName, trafficTarget)
		Expect(err).NotTo(HaveOccurred())

		clientToServer := HTTPRequestDef{
			SourceNs:        sourceName,
			SourcePod:       srcPod.Name,
			SourceContainer: sourceName,

			Destination: fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace),
		}

		srcToDestStr := fmt.Sprintf("%s -> %s",
			fmt.Sprintf("%s/%s", sourceName, srcPod.Name),
			clientToServer.Destination)

		// Send traffic
		cond := Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(clientToServer)

			if result.Err != nil || result.StatusCode != 200 {
				Td.T.Logf("> (%s) HTTP Req failed with status: %d, err: %s",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) HTTP Req succeeded: %d", srcToDestStr, result.StatusCode)
			return true
		}, 5, Td.ReqSuccessTimeout)
		Expect(cond).To(BeTrue())

		// Rate limit
		By("Enforce HTTP rate limit at VirtualHost level")

		upstreamTrafficSetting := &policyv1alpha1.UpstreamTrafficSetting{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dstSvc.Name,
				Namespace: dstSvc.Namespace,
			},
			Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
				Host: fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace),
				RateLimit: &policyv1alpha1.RateLimitSpec{
					Local: &policyv1alpha1.LocalRateLimitSpec{
						HTTP: &policyv1alpha1.HTTPLocalRateLimitSpec{
							Requests: 1,
							Unit:     "minute",
						},
					},
				},
			},
		}

		upstreamTrafficSetting, err = Td.PolicyClient.PolicyV1alpha1().UpstreamTrafficSettings(upstreamTrafficSetting.Namespace).Create(context.TODO(), upstreamTrafficSetting, metav1.CreateOptions{})
		Expect(err).ToNot((HaveOccurred()))

		// Expect client to receive 429 (Too Many Requests)
		cond = Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(clientToServer)

			if result.StatusCode != 429 {
				Td.T.Logf("> (%s) HTTP Req did not fail as expected, status: %d, err: %s",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}

			Td.T.Logf("> (%s) HTTP Req failed as expected with status: %d", srcToDestStr, result.StatusCode)
			return true
		}, 5, Td.ReqSuccessTimeout)
		Expect(cond).To(BeTrue())

		By("Enforce HTTP rate limit at Route level")

		upstreamTrafficSetting.Spec.RateLimit = nil
		upstreamTrafficSetting.Spec.HTTPRoutes = append(upstreamTrafficSetting.Spec.HTTPRoutes,
			policyv1alpha1.HTTPRouteSpec{
				Path: ".*", // Matches the Path allowed by HTTPRouteGroup policy
				RateLimit: &policyv1alpha1.HTTPPerRouteRateLimitSpec{
					Local: &policyv1alpha1.HTTPLocalRateLimitSpec{
						Requests: 1,
						Unit:     "minute",
					},
				},
			})

		_, err = Td.PolicyClient.PolicyV1alpha1().UpstreamTrafficSettings(upstreamTrafficSetting.Namespace).Update(context.TODO(), upstreamTrafficSetting, metav1.UpdateOptions{})
		Expect(err).ToNot((HaveOccurred()))

		// Expect client to receive 429 (Too Many Requests)
		cond = Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(clientToServer)

			if result.StatusCode != 429 {
				Td.T.Logf("> (%s) HTTP Req did not fail as expected, status: %d, err: %s",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}

			Td.T.Logf("> (%s) HTTP Req failed as expected with status: %d", srcToDestStr, result.StatusCode)
			return true
		}, 5, Td.ReqSuccessTimeout)

		Expect(cond).To(BeTrue())
	})
}
