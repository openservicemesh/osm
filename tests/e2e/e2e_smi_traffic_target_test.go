package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test HTTP traffic with SMI TrafficTarget",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 1,
	},
	func() {
		Context("SMI TrafficTarget", func() {
			const sourceOne = "client-one"
			const sourceTwo = "client-two"
			const destName = "server"
			var ns = []string{sourceOne, sourceTwo, destName}

			It("Tests HTTP traffic for client pod -> server pod", func() {
				// Install OSM
				Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

				// Create Test NS
				for _, n := range ns {
					Expect(Td.CreateNs(n, nil)).To(Succeed())
					Expect(Td.AddNsToMesh(true, n)).To(Succeed())
				}

				// Set up the destination HTTP server
				svcAccDef, podDef, svcDef := Td.SimplePodApp(
					SimplePodAppDef{
						Name:      destName,
						Namespace: destName,
						Image:     "kennethreitz/httpbin",
						Ports:     []int{80},
					})
				_, err := Td.CreateServiceAccount(destName, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreatePod(destName, podDef)
				Expect(err).NotTo(HaveOccurred())
				dstSvc, err := Td.CreateService(destName, svcDef)
				Expect(err).NotTo(HaveOccurred())

				// Set up the HTTP client that is allowed access to the destination
				allowedSvcAccDef, allowedSrcPodDef, _ := Td.SimplePodApp(SimplePodAppDef{
					Name:      sourceOne,
					Namespace: sourceOne,
					Command:   []string{"sleep", "365d"},
					Image:     "curlimages/curl",
					Ports:     []int{80},
				})
				_, err = Td.CreateServiceAccount(sourceOne, &allowedSvcAccDef)
				Expect(err).NotTo(HaveOccurred())
				allowedSrcPod, err := Td.CreatePod(sourceOne, allowedSrcPodDef)
				Expect(err).NotTo(HaveOccurred())

				// Set up the HTTP client that is denied access to the destination
				deniedSvcAccDef, deniedSrcPodDef, _ := Td.SimplePodApp(SimplePodAppDef{
					Name:      sourceTwo,
					Namespace: sourceTwo,
					Command:   []string{"sleep", "365d"},
					Image:     "curlimages/curl",
					Ports:     []int{80},
				})
				_, err = Td.CreateServiceAccount(sourceTwo, &deniedSvcAccDef)
				Expect(err).NotTo(HaveOccurred())
				deniedSrcPod, err := Td.CreatePod(sourceTwo, deniedSrcPodDef)
				Expect(err).NotTo(HaveOccurred())

				// Wait for client and server pods to be ready
				Expect(Td.WaitForPodsRunningReady(sourceOne, 90*time.Second, 1)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(sourceTwo, 90*time.Second, 1)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(destName, 90*time.Second, 1)).To(Succeed())

				// The test creates 2 policies:
				// 1. Policy to allow client 'sourceOne' to access server 'destName' on the HTTP path '/anything'
				// 2. Policy to allow client 'sourceTwo' to access server 'destName' on the HTTP path '/foo'
				//
				// The test verifies that client 'sourceOne' is able to access server 'destName' on path '/anything',
				// while client 'sourceTwo' is not able to access server 'destName' on path '/anything' even though it
				// is allowed access to the path '/foo'.
				By("Creating SMI policies")
				// Deploy policies to allow 'sourceOne' to access destination at HTTP path '/anything'
				anythingPath := "/anything"
				httpRGOne, trafficTargetOne := createPolicyForRoutePath(sourceOne, destName, anythingPath)
				_, err = Td.CreateHTTPRouteGroup(sourceOne, httpRGOne)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(sourceOne, trafficTargetOne)
				Expect(err).NotTo(HaveOccurred())

				// Deploy policies to allow 'sourceTwo' to access destination at HTTP path '/foo'
				// This is done so that 'sourceTwo' can access the destination server but not on
				// path '/anything' which is used to demonstrate RBAC per route.
				fooPath := "/foo"
				httpRGTwo, trafficTargetTwo := createPolicyForRoutePath(sourceTwo, destName, fooPath)
				_, err = Td.CreateHTTPRouteGroup(sourceTwo, httpRGTwo)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(sourceTwo, trafficTargetTwo)
				Expect(err).NotTo(HaveOccurred())

				// HTTP request from 'sourceOne': http://<address>/anything
				allowedClientToServer := HTTPRequestDef{
					SourceNs:        sourceOne,
					SourcePod:       allowedSrcPod.Name,
					SourceContainer: sourceOne,

					Destination: fmt.Sprintf("%s.%s%s", dstSvc.Name, dstSvc.Namespace, anythingPath),
				}

				allowedSrcToDestStr := fmt.Sprintf("%s -> %s",
					fmt.Sprintf("%s/%s", sourceOne, allowedSrcPod.Name),
					allowedClientToServer.Destination)

				// Verify HTTP requests succeed from allowed client to destination server
				cond := Td.WaitForRepeatedSuccess(func() bool {
					result := Td.HTTPRequest(allowedClientToServer)

					if result.Err != nil || result.StatusCode != 200 {
						Td.T.Logf("> (%s) HTTP Req failed %d %s",
							allowedSrcToDestStr, result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> (%s) HTTP Req succeeded: %d", allowedSrcToDestStr, result.StatusCode)
					return true
				}, 5, 90*time.Second)
				Expect(cond).To(BeTrue())

				// HTTP request from 'sourceTwo': http://<address>/anything
				deniedClientToServer := HTTPRequestDef{
					SourceNs:        sourceTwo,
					SourcePod:       deniedSrcPod.Name,
					SourceContainer: sourceTwo,

					Destination: fmt.Sprintf("%s.%s%s", dstSvc.Name, dstSvc.Namespace, anythingPath),
				}

				deniedSrcToDestStr := fmt.Sprintf("%s -> %s",
					fmt.Sprintf("%s/%s", sourceTwo, deniedSrcPod.Name),
					deniedClientToServer.Destination)

				// Verify HTTP requests fail from denied client to destination server
				cond = Td.WaitForRepeatedSuccess(func() bool {
					result := Td.HTTPRequest(deniedClientToServer)

					// 403 means the request is forbidden, expected due to RBAC policy on destination
					if result.StatusCode != 403 {
						Td.T.Logf("> (%s) HTTP Req did not fail, incorrect expected result: %d, %s", deniedSrcToDestStr, result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> (%s) HTTP Req failed correctly with status code %d", deniedSrcToDestStr, result.StatusCode)
					return true
				}, 10, 90*time.Second)
				Expect(cond).To(BeTrue())

				By("Deleting SMI policies")
				Expect(Td.SmiClients.AccessClient.AccessV1alpha3().TrafficTargets(sourceOne).Delete(context.TODO(), trafficTargetOne.Name, metav1.DeleteOptions{})).To(Succeed())
				Expect(Td.SmiClients.SpecClient.SpecsV1alpha4().HTTPRouteGroups(sourceOne).Delete(context.TODO(), httpRGOne.Name, metav1.DeleteOptions{})).To(Succeed())

				// Verify HTTP requests fail from allowed client to destination server after SMI policies are deleted
				cond = Td.WaitForRepeatedSuccess(func() bool {
					result := Td.HTTPRequest(allowedClientToServer)

					// Curl exit code 7 == Conn refused
					if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
						Td.T.Logf("> (%s) HTTP Req did not fail, incorrect expected result: %d, %s", allowedSrcToDestStr, result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> (%s) HTTP Req failed correctly: %s", allowedSrcToDestStr, result.Err)
					return true
				}, 5, 150*time.Second)
				Expect(cond).To(BeTrue())
			})
		})
	})

// createPolicyForRoutePath creates an HTTPRouteGroup and TrafficTarget policy for the given source, destination and HTTP path regex
func createPolicyForRoutePath(source string, destination string, pathRegex string) (smiSpecs.HTTPRouteGroup, smiAccess.TrafficTarget) {
	routeGroupName := "test-route"
	routeMatchName := "allowed-route"

	routeGroup := smiSpecs.HTTPRouteGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: routeGroupName,
		},
		Spec: smiSpecs.HTTPRouteGroupSpec{
			Matches: []smiSpecs.HTTPMatch{
				{
					Name:      routeMatchName,
					PathRegex: pathRegex,
					Methods:   []string{"*"},
				},
			},
		},
	}

	trafficTargetOne := smiAccess.TrafficTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", source, destination),
		},
		Spec: smiAccess.TrafficTargetSpec{
			Sources: []smiAccess.IdentityBindingSubject{
				{
					Kind:      "ServiceAccount",
					Name:      source,
					Namespace: source,
				},
			},
			Destination: smiAccess.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      destination,
				Namespace: destination,
			},
			Rules: []smiAccess.TrafficTargetRule{
				{
					Kind: "HTTPRouteGroup",
					Name: routeGroupName,
					Matches: []string{
						routeMatchName,
					},
				},
			},
		},
	}

	return routeGroup, trafficTargetOne
}
