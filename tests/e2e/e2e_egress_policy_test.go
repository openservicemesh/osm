package e2e

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	. "github.com/openservicemesh/osm/tests/framework"
)

type testScenario int

const (
	httpEgressNoRouteMatches testScenario = iota
	httpEgressWithRouteMatches
	httpsEgress
	tcpEgress
)

var _ = OSMDescribe("Tests external traffic using egress policy",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 3,
	},
	func() {
		Context("HTTP egress policy without route matches", func() {
			// Tests HTTP egress traffic without SMI route matches
			testEgressPolicy(httpEgressNoRouteMatches)
		})

		Context("HTTP egress policy with route match", func() {
			// Tests HTTP egress traffic with SMI route match
			testEgressPolicy(httpEgressWithRouteMatches)
		})

		Context("HTTPS egress policy", func() {
			// Tests HTTPS egress traffic
			testEgressPolicy(httpsEgress)
		})

		Context("TCP egress policy", func() {
			// Tests TCP egress traffic
			testEgressPolicy(tcpEgress)
		})
	})

func testEgressPolicy(scenario testScenario) {
	It("Tests external traffic using egress policy", func() {
		const sourceNs = "client"
		const sourceName = "client"

		// Install OSM
		installOpts := Td.GetOSMInstallOpts()
		installOpts.EgressEnabled = false // Disable global egress
		Expect(Td.InstallOSM(installOpts)).To(Succeed())

		// Create source namespace and add it to the mesh
		Expect(Td.CreateNs(sourceNs, nil)).To(Succeed())
		Expect(Td.AddNsToMesh(true, sourceNs)).To(Succeed())

		// Create simple pod definitions for the source
		srcSvcAcc, srcPodDef, _, err := Td.SimplePodApp(SimplePodAppDef{
			PodName:   sourceName,
			Namespace: sourceNs,
			Command:   []string{"/bin/bash", "-c", "--"},
			Args:      []string{"while true; do sleep 30; done;"},
			Image:     "songrgg/alpine-debug",
			Ports:     []int{80},
			OS:        Td.ClusterOS,
		})
		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(sourceNs, &srcSvcAcc)
		Expect(err).NotTo(HaveOccurred())
		srcPod, err := Td.CreatePod(sourceNs, srcPodDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's namespace
		Expect(Td.WaitForPodsRunningReady(sourceNs, 60*time.Second, 1, nil)).To(Succeed())

		url, policy, smiHTTPRoute := getTestAttributes(srcSvcAcc, scenario)

		httpRequest := HTTPRequestDef{
			SourceNs:        srcPod.Namespace,
			SourcePod:       srcPod.Name,
			SourceContainer: sourceName,
			Destination:     url,
		}

		//
		// Verify traffic is allowed when policies are applied
		//
		if smiHTTPRoute != nil {
			// Create an SMI HTTP route
			_, err := Td.SmiClients.SpecClient.SpecsV1alpha4().HTTPRouteGroups(smiHTTPRoute.Namespace).Create(context.TODO(), smiHTTPRoute, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		By("Creating an Egress policy")
		policy, err = Td.PolicyClient.PolicyV1alpha1().Egresses(srcPod.Namespace).Create(context.TODO(), policy, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		cond := Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(httpRequest)

			if result.Err != nil || result.StatusCode != 200 {
				Td.T.Logf("> Request %s failed, status code: %d, err: %s", url, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> Request %s succeeded, status code: %d", url, result.StatusCode)
			return true
		}, 5, 60*time.Second)
		Expect(cond).To(BeTrue())

		//
		// Verify traffic is blocked when policies are removed
		//
		if smiHTTPRoute != nil {
			// Create an SMI HTTP route
			err := Td.SmiClients.SpecClient.SpecsV1alpha4().HTTPRouteGroups(smiHTTPRoute.Namespace).Delete(context.TODO(), smiHTTPRoute.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		By("Deleting an Egress policy")
		err = Td.PolicyClient.PolicyV1alpha1().Egresses(policy.Namespace).Delete(context.TODO(), policy.Name, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Expect client not to reach server
		cond = Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(httpRequest)

			// Curl exit code 7 == Conn refused
			if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
				Td.T.Logf("> Request %s did not fail as expected, status code: %d, err: %s", url, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> Request %s failed correctly (%s)", url, result.Err)
			return true
		}, 5, 150*time.Second)
		Expect(cond).To(BeTrue())
	})
}

// getTestAttributes returns a URL, Egress policy, SMI HTTPRouteGroup for the given test scenario and client
func getTestAttributes(srcSvcAcc corev1.ServiceAccount, scenario testScenario) (string, *policyV1alpha1.Egress, *smiSpecs.HTTPRouteGroup) {
	switch scenario {
	case httpEgressNoRouteMatches:
		return "http://httpbin.org:80/status/200", &policyV1alpha1.Egress{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Egress",
				APIVersion: "policy.openservicemesh.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "httpbin-80",
				Namespace: srcSvcAcc.Namespace,
			},
			Spec: policyV1alpha1.EgressSpec{
				Sources: []policyV1alpha1.EgressSourceSpec{
					{
						Kind:      "ServiceAccount",
						Name:      srcSvcAcc.Name,
						Namespace: srcSvcAcc.Namespace,
					},
				},
				Hosts: []string{
					"httpbin.org",
				},
				Ports: []policyV1alpha1.PortSpec{
					{
						Number:   80,
						Protocol: "http",
					},
				},
			},
		}, nil

	case httpEgressWithRouteMatches:
		return "http://httpbin.org:80/status/200", &policyV1alpha1.Egress{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Egress",
					APIVersion: "policy.openservicemesh.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httpbin-80",
					Namespace: srcSvcAcc.Namespace,
				},
				Spec: policyV1alpha1.EgressSpec{
					Sources: []policyV1alpha1.EgressSourceSpec{
						{
							Kind:      "ServiceAccount",
							Name:      srcSvcAcc.Name,
							Namespace: srcSvcAcc.Namespace,
						},
					},
					Hosts: []string{
						"httpbin.org",
					},
					Ports: []policyV1alpha1.PortSpec{
						{
							Number:   80,
							Protocol: "http",
						},
					},
					Matches: []corev1.TypedLocalObjectReference{
						{
							APIGroup: pointer.StringPtr("specs.smi-spec.io/v1alpha4"),
							Kind:     "HTTPRouteGroup",
							Name:     "httpbin-status",
						},
					},
				},
			}, &smiSpecs.HTTPRouteGroup{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Egress",
					APIVersion: "policy.openservicemesh.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "httpbin-status",
					Namespace: srcSvcAcc.Namespace,
				},
				Spec: smiSpecs.HTTPRouteGroupSpec{
					Matches: []smiSpecs.HTTPMatch{
						{
							Name:      "status",
							PathRegex: "/status.*",
						},
					},
				},
			}

	case httpsEgress:
		return "https://httpbin.org:443/status/200", &policyV1alpha1.Egress{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Egress",
				APIVersion: "policy.openservicemesh.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "httpbin-443",
				Namespace: srcSvcAcc.Namespace,
			},
			Spec: policyV1alpha1.EgressSpec{
				Sources: []policyV1alpha1.EgressSourceSpec{
					{
						Kind:      "ServiceAccount",
						Name:      srcSvcAcc.Name,
						Namespace: srcSvcAcc.Namespace,
					},
				},
				Hosts: []string{
					"httpbin.org",
				},
				Ports: []policyV1alpha1.PortSpec{
					{
						Number:   443,
						Protocol: "https",
					},
				},
			},
		}, nil

	case tcpEgress:
		return "https://httpbin.org:443/status/200", &policyV1alpha1.Egress{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Egress",
				APIVersion: "policy.openservicemesh.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "httpbin-443",
				Namespace: srcSvcAcc.Namespace,
			},
			Spec: policyV1alpha1.EgressSpec{
				Sources: []policyV1alpha1.EgressSourceSpec{
					{
						Kind:      "ServiceAccount",
						Name:      srcSvcAcc.Name,
						Namespace: srcSvcAcc.Namespace,
					},
				},
				Ports: []policyV1alpha1.PortSpec{
					{
						Number:   443,
						Protocol: "tcp",
					},
				},
			},
		}, nil

	default:
		Td.T.Fatal("Unsupportes test scenario: %v", scenario)
		return "", nil, nil
	}
}
