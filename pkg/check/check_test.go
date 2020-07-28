package check

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/testing"

	"github.com/openservicemesh/osm/pkg/cli"
)

var passing = Check{
	name: "pass",
	run:  func(*Checker) error { return nil },
}

var failing = Check{
	name: "fail",
	run:  func(*Checker) error { return errors.New("error") },
}

var fatalFailing = Check{
	name:  "fatal fail",
	fatal: true,
	run:   func(*Checker) error { return errors.New("error") },
}

var _ = Describe("Checker", func() {
	Describe("Run", func() {
		Describe("when a single check passes", func() {
			var (
				checker *Checker
				pass    bool
				cbCount int
			)
			BeforeEach(func() {
				cbCount = 0
				checker = NewChecker(cli.New())
				pass = checker.Run([]Check{passing}, func(*Result) { cbCount++ })
			})
			It("passes", func() {
				Expect(pass).To(BeTrue())
			})
			It("invokes the callback", func() {
				Expect(cbCount).To(Equal(1))
			})
		})

		Describe("when a non-fatal error is hit", func() {
			var (
				checker *Checker
				pass    bool
				cbCount int
			)
			BeforeEach(func() {
				cbCount = 0
				checker = NewChecker(cli.New())
				pass = checker.Run([]Check{failing, passing}, func(*Result) { cbCount++ })
			})
			It("fails", func() {
				Expect(pass).To(BeFalse())
			})
			It("continues to run other checks", func() {
				Expect(cbCount).To(Equal(2))
			})
		})

		Describe("when a fatal error is hit", func() {
			var (
				checker *Checker
				pass    bool
				cbCount int
			)
			BeforeEach(func() {
				cbCount = 0
				checker = NewChecker(cli.New())
				pass = checker.Run([]Check{fatalFailing, passing}, func(*Result) { cbCount++ })
			})
			It("fails", func() {
				Expect(pass).To(BeFalse())
			})
			It("stops as soon as the error is hit", func() {
				Expect(cbCount).To(Equal(1))
			})
		})
	})

	Describe("checkAuthz", func() {
		It("passes when an action is allowed", func() {
			k8s := fake.NewSimpleClientset()
			allowAllSelfSubjectAccessReviews(k8s)
			c := &Checker{
				k8s: k8s,
			}
			err := c.checkAuthz("create", "", "v1", "namespaces", "")
			Expect(err).NotTo(HaveOccurred())
		})

		It("fails when an action isn't allowed", func() {
			c := &Checker{
				k8s: fake.NewSimpleClientset(),
			}
			err := c.checkAuthz("create", "", "v1", "namespaces", "")
			Expect(err).To(HaveOccurred())
		})
	})
})

func allowAllSelfSubjectAccessReviews(k8s *fake.Clientset) {
	k8s.Fake.PrependReactor("create", "selfsubjectaccessreviews", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
		s := action.(testing.CreateActionImpl).GetObject().(*authv1.SelfSubjectAccessReview)
		s.Status.Allowed = true
		ret = s
		return
	})
}

type failingRESTClientGetter struct {
	*genericclioptions.TestConfigFlags
}

func (f failingRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return nil, errors.New("ToRESTConfig error")
}

type failingServerVersionGetter struct {
	*fakediscovery.FakeDiscovery
}

func (failingServerVersionGetter) ServerVersion() (*version.Info, error) {
	return nil, errors.New("server version error")
}

type failingServerVersionGettingClient struct {
	*fake.Clientset
}

func (failingServerVersionGettingClient) Discovery() discovery.DiscoveryInterface {
	return failingServerVersionGetter{}
}

var _ = Describe("checks", func() {
	Describe("initK8sClient", func() {
		It("fails if an error occurs", func() {
			c := &Checker{
				restClientGetter: failingRESTClientGetter{},
			}
			err := initK8sClient.run(c)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("queryK8sAPI", func() {
		It("passes", func() {
			c := &Checker{
				k8s: fake.NewSimpleClientset(),
			}
			err := queryK8sAPI.run(c)
			Expect(err).NotTo(HaveOccurred())
		})
		It("fails when getting the server version fails", func() {
			c := &Checker{
				k8s: failingServerVersionGettingClient{},
			}
			err := queryK8sAPI.run(c)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("k8sVersion", func() {
		It("passes for v1.18.2", func() {
			c := &Checker{
				k8sVersion: &version.Info{
					GitVersion: "v1.18.2",
				},
			}
			err := k8sVersion.run(c)
			Expect(err).NotTo(HaveOccurred())
		})
		It("passes for v1.16.0-beta.0", func() {
			c := &Checker{
				k8sVersion: &version.Info{
					GitVersion: "v1.16.0-beta.0",
				},
			}
			err := k8sVersion.run(c)
			Expect(err).NotTo(HaveOccurred())
		})
		It("fails for v1.10.0", func() {
			c := &Checker{
				k8sVersion: &version.Info{
					GitVersion: "v1.10.0",
				},
			}
			err := k8sVersion.run(c)
			Expect(err).To(MatchError("Kubernetes version v1.10.0 does not match supported versions ^1.15-0"))
		})
	})

	Describe("modifyIptables", func() {
		It("passes when no PSPs are found", func() {
			c := &Checker{
				k8s: fake.NewSimpleClientset(),
			}
			err := modifyIptables.run(c)
			Expect(err).NotTo(HaveOccurred())
		})
		It("passes when a PSP defines the capabilities explicitly", func() {
			k8s := fake.NewSimpleClientset(&policyv1beta1.PodSecurityPolicy{
				Spec: policyv1beta1.PodSecurityPolicySpec{
					AllowedCapabilities: []corev1.Capability{"NET_ADMIN", "NET_RAW"},
				},
			})
			allowAllSelfSubjectAccessReviews(k8s)
			c := &Checker{
				k8s: k8s,
			}
			err := modifyIptables.run(c)
			Expect(err).NotTo(HaveOccurred())
		})
		It("passes when a PSP defines the wildcard capability", func() {
			k8s := fake.NewSimpleClientset(&policyv1beta1.PodSecurityPolicy{
				Spec: policyv1beta1.PodSecurityPolicySpec{
					AllowedCapabilities: []corev1.Capability{"*"},
				},
			})
			allowAllSelfSubjectAccessReviews(k8s)
			c := &Checker{
				k8s: k8s,
			}
			err := modifyIptables.run(c)
			Expect(err).NotTo(HaveOccurred())
		})
		It("fails if a PSP that defines the capabilities isn't usable", func() {
			k8s := fake.NewSimpleClientset(&policyv1beta1.PodSecurityPolicy{
				Spec: policyv1beta1.PodSecurityPolicySpec{
					AllowedCapabilities: []corev1.Capability{"*"},
				},
			})
			c := &Checker{
				k8s: k8s,
			}
			err := modifyIptables.run(c)
			Expect(err).To(HaveOccurred())
		})
		It("fails if a usable PSP doesn't define the capabilities", func() {
			k8s := fake.NewSimpleClientset(&policyv1beta1.PodSecurityPolicy{
				Spec: policyv1beta1.PodSecurityPolicySpec{
					AllowedCapabilities: []corev1.Capability{"NET_OTHER"},
				},
			})
			allowAllSelfSubjectAccessReviews(k8s)
			c := &Checker{
				k8s: k8s,
			}
			err := modifyIptables.run(c)
			Expect(err).To(HaveOccurred())
		})
		It("fails if a usable PSP defines only some of the capabilities", func() {
			k8s := fake.NewSimpleClientset(&policyv1beta1.PodSecurityPolicy{
				Spec: policyv1beta1.PodSecurityPolicySpec{
					AllowedCapabilities: []corev1.Capability{"NET_ADMIN"},
				},
			})
			allowAllSelfSubjectAccessReviews(k8s)
			c := &Checker{
				k8s: k8s,
			}
			err := modifyIptables.run(c)
			Expect(err).To(HaveOccurred())
		})
	})
})
