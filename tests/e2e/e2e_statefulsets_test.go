package e2e

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	"github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	"github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test HTTP traffic from 1 pod client -> 1 pod server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 9,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("Statefulsets", func() {
			It("pods succeed while establishing consensus", func() {
				// Install OSM (with proxyMode = podIP)
				Expect(Td.InstallOSM(Td.GetOSMInstallOpts(WithLocalProxyMode(v1alpha2.LocalProxyModePodIP)))).To(Succeed())

				const testNS = "zookeeper"

				// Create Test NS
				Expect(Td.CreateNs(testNS, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, testNS)).To(Succeed())

				helmCfg := &action.Configuration{}
				Expect(helmCfg.Init(Td.Env.RESTClientGetter(), testNS, "secret", log.Info().Msgf)).To(Succeed())

				install := action.NewInstall(helmCfg)

				install.ReleaseName = "kafka"
				install.Namespace = testNS
				saName := "zookeeper"
				replicaCount := 3

				chart, err := loader.Load("https://charts.bitnami.com/bitnami/zookeeper-9.0.2.tgz")
				Expect(err).NotTo(HaveOccurred())

				// Install zookeeper
				Expect(install.Run(chart, map[string]interface{}{
					"replicaCount": replicaCount,
					"serviceAccount": map[string]interface{}{
						"create": true,
						"name":   saName,
					},
				})).To(Succeed())

				// Create SMI resources for Zookeeper
				_, err = Td.CreateTCPRoute(testNS, v1alpha4.TCPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS,
						Name:      "zookeeper",
					},
					Spec: v1alpha4.TCPRouteSpec{
						Matches: v1alpha4.TCPMatch{
							Ports: []int{2181, 3181, 2888, 3888},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateTrafficTarget(testNS, v1alpha3.TrafficTarget{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNS,
						Name:      "zookeeper",
					},
					Spec: v1alpha3.TrafficTargetSpec{
						Sources: []v1alpha3.IdentityBindingSubject{
							{
								Kind:      "ServiceAccount",
								Name:      saName,
								Namespace: testNS,
							},
						},
						Destination: v1alpha3.IdentityBindingSubject{
							Kind:      "ServiceAccount",
							Name:      saName,
							Namespace: testNS,
						},
						Rules: []v1alpha3.TrafficTargetRule{
							{
								Kind: "TCPRoute",
								Name: "zookeeper",
							},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(testNS, 90*time.Second, replicaCount, nil)).To(Succeed())

				pods, err := Td.Client.CoreV1().Pods(testNS).List(context.TODO(), metav1.ListOptions{})

				Expect(err).NotTo(HaveOccurred())

				// this command will exit 1 if connectivity isn't established
				cmd := "/opt/bitnami/zookeeper/bin/zkServer.sh status"

				cond := Td.WaitForRepeatedSuccess(func() bool {

					var results map[string]error
					for _, po := range pods.Items {
						stdout, stderr, err := Td.RunRemote(testNS, po.GetName(), "zookeeper", strings.Fields(cmd))

						if err != nil {
							// Error codes from the execution come through err
							// Curl 'Connection refused' err code = 7
							Td.T.Logf("> (%s) Remote exec err: %v | stderr: %s", po.GetName(), err, stderr)
						}

						Td.T.Logf("> (%s) Stdout %s", po.GetName(), stdout)

						results[po.GetName()] = err
					}

					hadErr := false
					for podName, err := range results {
						if err != nil {
							Td.T.Logf("> (%s) ZK status check failed: expected nil err, got %s", podName, err)
							hadErr = true
							continue
						}

						Td.T.Logf("> (%s) ZK status check suceeded!", podName)
					}

					if hadErr {
						return false
					}

					return true
				}, 5, Td.ReqSuccessTimeout)

				Expect(cond).To(BeTrue())

			})
		})
	})
