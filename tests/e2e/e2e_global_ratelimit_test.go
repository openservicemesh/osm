package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	. "github.com/openservicemesh/osm/tests/framework"
)

const (
	rateLimiterNamespace = "rls-test"
	rateLimiterSvc       = "ratelimiter"
	rateLimiterConfig    = "ratelimit-config"
	rateLimiterPort      = 8081
)

var _ = OSMDescribe("Global rate limiting for HTTP traffic",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 3,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("HTTP request rate limiting", func() {
			testHTTPGlobalRateLimiting()
		})
	})

// testHTTPGlobalRateLimiting tests rate limiting of HTTP traffic
// with the different supported descriptor types:
// genericKey, remoteAddress, requestHeader, headerValueMatch
func testHTTPGlobalRateLimiting() {
	const sourceName = "client"
	const destName = "server"

	var appNs = []string{sourceName, destName}

	It("Tests rate limiting of traffic from client pod -> service", func() {
		// Install OSM
		installOpts := Td.GetOSMInstallOpts()
		installOpts.EnablePermissiveMode = true
		Expect(Td.InstallOSM(installOpts)).To(Succeed())

		// Create the RLS service
		Expect(Td.CreateNs(rateLimiterNamespace, nil)).To(Succeed())
		Expect(deployRLS()).To(Succeed())

		// Create Test NS
		for _, n := range appNs {
			Expect(Td.CreateNs(n, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, n)).To(Succeed())
		}

		// Get simple pod definitions for the HTTP server
		svcAccDef, podDef, svcDef, err := Td.GetOSSpecificHTTPBinPod(destName, destName)
		Expect(err).NotTo(HaveOccurred())
		destNsName := types.NamespacedName{Namespace: destName, Name: destName}

		_, err = Td.CreateServiceAccount(destName, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destName, podDef)
		Expect(err).NotTo(HaveOccurred())
		dstSvc, err := Td.CreateService(destName, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destName, 90*time.Second, 1, nil)).To(Succeed())

		srcPod := setupSource(sourceName, false)

		//
		// Verify traffic is not rate limited without any rate limit policy configured
		//
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

		//
		// Test rate limit using the genericKey descriptor
		// at the VirtualHost level.
		//
		// Configure the server to rate limit requests by
		// generating the descriptor ("my_key", "my_value").
		// The rate limit policy applied on the RLS will rate
		// limit requests with this descriptor to 1 per minute.
		//
		By("Enforce genericKey rate limit at VirtualHost level")
		err = Td.ResetEnvoyStats(destNsName)
		Expect(err).ToNot((HaveOccurred()))

		upstreamTrafficSetting := &policyv1alpha1.UpstreamTrafficSetting{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dstSvc.Name,
				Namespace: dstSvc.Namespace,
			},
			Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
				Host: fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace),
				RateLimit: &policyv1alpha1.RateLimitSpec{
					Global: &policyv1alpha1.GlobalRateLimitSpec{
						HTTP: &policyv1alpha1.HTTPGlobalRateLimitSpec{
							RateLimitService: policyv1alpha1.RateLimitServiceSpec{
								Host: fmt.Sprintf("%s.%s.svc.cluster.local", rateLimiterSvc, rateLimiterNamespace),
								Port: rateLimiterPort,
							},
							Domain:   "test",
							FailOpen: pointer.BoolPtr(false),
							Descriptors: []policyv1alpha1.HTTPGlobalRateLimitDescriptor{
								{
									Entries: []policyv1alpha1.HTTPGlobalRateLimitDescriptorEntry{
										{
											GenericKey: &policyv1alpha1.GenericKeyDescriptorEntry{
												Key:   "my_key",
												Value: "my_value",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		upstreamTrafficSetting, err = Td.PolicyClient.PolicyV1alpha1().UpstreamTrafficSettings(upstreamTrafficSetting.Namespace).Create(context.TODO(), upstreamTrafficSetting, metav1.CreateOptions{})
		Expect(err).ToNot((HaveOccurred()))

		// Expect client to receive 429 (Too Many Requests)
		rlFunc := func() bool {
			result := Td.HTTPRequest(clientToServer)

			if result.StatusCode != 429 {
				Td.T.Logf("> (%s) HTTP Req did not fail as expected, status: %d, err: %s",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}

			Td.T.Logf("> (%s) HTTP Req failed as expected with status: %d", srcToDestStr, result.StatusCode)
			return true
		}
		cond = Td.WaitForRepeatedSuccess(rlFunc, 5, Td.ReqSuccessTimeout)
		Expect(cond).To(BeTrue())
		metrics, err := Td.GetEnvoyMetric(destNsName, []string{`.*over_limit.*`})
		Expect(err).ToNot((HaveOccurred()))
		Td.T.Logf("over_limit metric: %d", metrics)
		Expect(metrics[0]).To(BeNumerically(">=", 5))

		//
		// Test rate limit using the remoteAddress descriptor
		// at the VirtualHost level.
		//
		// Configure the server to generate the descriptor
		// ("remote_address", "<x-forwarded-for value>").
		// The rate limit policy applied on the RLS will rate
		// limit requests with this descriptor to 1 per minute.
		//
		By("Enforce remoteAddress rate limit at VirtualHost level")
		err = Td.ResetEnvoyStats(destNsName)
		Expect(err).ToNot((HaveOccurred()))

		upstreamTrafficSetting.Spec.RateLimit.Global.HTTP.Descriptors = []policyv1alpha1.HTTPGlobalRateLimitDescriptor{
			{
				Entries: []policyv1alpha1.HTTPGlobalRateLimitDescriptorEntry{
					{
						RemoteAddress: &policyv1alpha1.RemoteAddressDescriptorEntry{},
					},
				},
			},
		}
		upstreamTrafficSetting, err = Td.PolicyClient.PolicyV1alpha1().UpstreamTrafficSettings(upstreamTrafficSetting.Namespace).Update(context.TODO(), upstreamTrafficSetting, metav1.UpdateOptions{})
		Expect(err).ToNot((HaveOccurred()))

		cond = Td.WaitForRepeatedSuccess(rlFunc, 5, Td.ReqSuccessTimeout)
		Expect(cond).To(BeTrue())
		metrics, err = Td.GetEnvoyMetric(destNsName, []string{`.*over_limit.*`})
		Expect(err).ToNot((HaveOccurred()))
		Td.T.Logf("over_limit metric: %d", metrics)
		Expect(metrics[0]).To(BeNumerically(">=", 5))

		//
		// Test rate limit using the requestHeader descriptor
		// at the route level.
		//
		// Configure the server to generate the descriptor
		// ("my_header", "<my-header value>").
		// Requests with "my-header: foo" are rate limited
		// to 1 per minute per the rate limit policy on the RLS.
		//
		By("Enforce requestHeader rate limit at Route level")
		err = Td.ResetEnvoyStats(destNsName)
		Expect(err).ToNot((HaveOccurred()))

		upstreamTrafficSetting.Spec.RateLimit.Global.HTTP.Descriptors = nil
		upstreamTrafficSetting.Spec.HTTPRoutes = []policyv1alpha1.HTTPRouteSpec{
			{
				Path: ".*", // Matches the Path allowed by permissive mode policy
				RateLimit: &policyv1alpha1.HTTPPerRouteRateLimitSpec{
					Global: &policyv1alpha1.HTTPGlobalPerRouteRateLimitSpec{
						Descriptors: []policyv1alpha1.HTTPGlobalRateLimitDescriptor{
							{
								Entries: []policyv1alpha1.HTTPGlobalRateLimitDescriptorEntry{
									{
										RequestHeader: &policyv1alpha1.RequestHeaderDescriptorEntry{
											Name: "my-header",
											Key:  "my_header",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		upstreamTrafficSetting, err = Td.PolicyClient.PolicyV1alpha1().UpstreamTrafficSettings(upstreamTrafficSetting.Namespace).Update(context.TODO(), upstreamTrafficSetting, metav1.UpdateOptions{})
		Expect(err).ToNot((HaveOccurred()))

		req := clientToServer
		req.Headers = map[string]string{"my-header": "foo"}
		testMyHeaderFoo := func() bool {
			result := Td.HTTPRequest(req)

			if result.StatusCode != 429 {
				Td.T.Logf("> (%s) HTTP Req did not fail as expected, status: %d, err: %s",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}

			Td.T.Logf("> (%s) HTTP Req failed as expected with status: %d", srcToDestStr, result.StatusCode)
			return true
		}

		cond = Td.WaitForRepeatedSuccess(testMyHeaderFoo, 5, Td.ReqSuccessTimeout*10)
		Expect(cond).To(BeTrue())
		metrics, err = Td.GetEnvoyMetric(destNsName, []string{`.*over_limit.*`})
		Expect(err).ToNot((HaveOccurred()))
		Td.T.Logf("over_limit metric: %d", metrics)
		Expect(metrics[0]).To(BeNumerically(">=", 5))

		//
		// Test rate limit using the headerValueMatch descriptor
		// at the route level.
		//
		// Configure the server to generate the descriptor
		// ("header_match", "foo") when the request contains
		// header "my-header" AND does not contain the header
		// "other-header". Such a request will be rate limited
		// to 1 per minute per the rate limit policy on the RLS.
		// Also verifies that requests that do not match the
		// specified header match criteria are not rate limited.
		//
		By("Enforce headerValueMatch rate limit at Route level")
		err = Td.ResetEnvoyStats(destNsName)
		Expect(err).ToNot((HaveOccurred()))

		upstreamTrafficSetting.Spec.RateLimit.Global.HTTP.Descriptors = nil
		upstreamTrafficSetting.Spec.HTTPRoutes = []policyv1alpha1.HTTPRouteSpec{
			{
				Path: ".*", // Matches the Path allowed by permissive mode policy
				RateLimit: &policyv1alpha1.HTTPPerRouteRateLimitSpec{
					Global: &policyv1alpha1.HTTPGlobalPerRouteRateLimitSpec{
						Descriptors: []policyv1alpha1.HTTPGlobalRateLimitDescriptor{
							{
								Entries: []policyv1alpha1.HTTPGlobalRateLimitDescriptorEntry{
									{
										HeaderValueMatch: &policyv1alpha1.HeaderValueMatchDescriptorEntry{
											Value: "foo",
											Headers: []policyv1alpha1.HTTPHeaderMatcher{
												{
													Name:    "my-header",
													Present: pointer.BoolPtr(true),
												},
												{
													Name:    "other-header",
													Present: pointer.BoolPtr(false),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		_, err = Td.PolicyClient.PolicyV1alpha1().UpstreamTrafficSettings(upstreamTrafficSetting.Namespace).Update(context.TODO(), upstreamTrafficSetting, metav1.UpdateOptions{})
		Expect(err).ToNot((HaveOccurred()))

		cond = Td.WaitForRepeatedSuccess(testMyHeaderFoo, 5, Td.ReqSuccessTimeout*10)
		Expect(cond).To(BeTrue())
		metrics, err = Td.GetEnvoyMetric(destNsName, []string{`.*over_limit.*`})
		Expect(err).ToNot((HaveOccurred()))
		Td.T.Logf("over_limit metric: %d", metrics)
		Expect(metrics[0]).To(BeNumerically(">=", 5))

		// Confirm requests containing the header "other-header" are not rate limited
		err = Td.ResetEnvoyStats(destNsName)
		Expect(err).ToNot((HaveOccurred()))
		req = clientToServer
		req.Headers = map[string]string{"other-header": "baz"}
		cond = Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(req)

			if result.Err != nil || result.StatusCode != 200 {
				Td.T.Logf("> (%s) HTTP Req failed with status: %d, err: %s",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) HTTP Req succeeded: %d", srcToDestStr, result.StatusCode)
			return true
		}, 5, Td.ReqSuccessTimeout)
		Expect(cond).To(BeTrue())
		metrics, err = Td.GetEnvoyMetric(destNsName, []string{`.*over_limit.*`})
		Expect(err).ToNot((HaveOccurred()))
		Td.T.Logf("over_limit metric: %d", metrics)
		Expect(metrics[0]).To(Equal(0))
	})
}

func deployRLS() error {
	rateLimiterConfig := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "corev1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rateLimiterConfig,
			Namespace: rateLimiterNamespace,
		},
		// Placeholder ratelimit-config
		Data: map[string]string{"ratelimit-config.yaml": `domain: test
descriptors:
  # requests with a descriptor ("my_key", "my_value")
  # are limited to one per minute.
  - key: my_key
    value: my_value
    rate_limit:
      unit: minute
      requests_per_unit: 1

  # each unique remote (client) address is limited to 3 per minute
  - key: remote_address
    rate_limit:
      unit: minute
      requests_per_unit: 1

  # requests with the header 'my-header: foo' are rate limited to 5 per minute
  - key: my_header
    value: foo
    rate_limit:
      unit: minute
      requests_per_unit: 1

  # requests with the descriptor 'header_match: foo' are rate limited
  # to 7 per minute
  - key: header_match
    value: foo
    rate_limit:
      unit: minute
      requests_per_unit: 1`},
	}

	_, err := Td.Client.CoreV1().ConfigMaps(rateLimiterNamespace).Create(context.TODO(), rateLimiterConfig, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	ratelimiterDeployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/appsv1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rateLimiterSvc,
			Namespace: rateLimiterNamespace,
			Labels:    map[string]string{"app": rateLimiterSvc},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{
				"app": rateLimiterSvc,
			},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": rateLimiterSvc}},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "ratelimit-config",
							VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "ratelimit-config"},
							},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: "redis:alpine",
							Env: []corev1.EnvVar{
								{
									Name:  "REDIS_SOCKET_TYPE",
									Value: "tcp",
								},
								{
									Name:  "REDIS_URL",
									Value: "redis:6379",
								},
							},
						},
						{
							Name:    rateLimiterSvc,
							Image:   "docker.io/envoyproxy/ratelimit:1f4ea68e",
							Command: []string{"/bin/ratelimit"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
									Protocol:      corev1.Protocol("TCP"),
								},
								{
									Name:          "grpc",
									ContainerPort: 8081,
									Protocol:      corev1.Protocol("TCP"),
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "USE_STATSD",
									Value: "false",
								},
								{
									Name:  "LOG_LEVEL",
									Value: "debug",
								},
								{
									Name:  "REDIS_SOCKET_TYPE",
									Value: "tcp",
								},
								{
									Name:  "REDIS_URL",
									Value: "localhost:6379",
								},
								{
									Name:  "RUNTIME_ROOT",
									Value: "/data",
								},
								{
									Name:  "RUNTIME_SUBDIRECTORY",
									Value: "ratelimit",
								},
								{
									Name:  "RUNTIME_WATCH_ROOT",
									Value: "false",
								},
								{
									Name:  "RUNTIME_IGNOREDOTFILES",
									Value: "true",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ratelimit-config",
									ReadOnly:  true,
									MountPath: "/data/ratelimit/config",
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
									Path: "/healthcheck",
									Port: intstr.IntOrString{
										IntVal: 8080,
									},
								},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
				},
			},
		},
	}

	_, err = Td.Client.AppsV1().Deployments(rateLimiterNamespace).Create(context.TODO(), ratelimiterDeployment, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	ratelimiterService := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "corev1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rateLimiterSvc,
			Namespace: rateLimiterNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "grpc",
					Protocol: corev1.Protocol("TCP"),
					Port:     rateLimiterPort,
				},
			},
			Selector: map[string]string{"app": rateLimiterSvc},
			Type:     corev1.ServiceType("ClusterIP"),
		},
	}

	_, err = Td.Client.CoreV1().Services(rateLimiterNamespace).Create(context.TODO(), ratelimiterService, metav1.CreateOptions{})

	return err
}
