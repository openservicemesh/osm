package check

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/open-service-mesh/osm/pkg/cli"
	authv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// Check encapsulates a single check against k8s/osm.
type Check struct {
	name  string
	fatal bool
	run   func(*Checker) error
}

// Result is the result of running a Check.
type Result struct {
	Name string
	Err  error
}

// NewChecker initializes a new *Checker.
func NewChecker(settings *cli.EnvSettings) *Checker {
	return &Checker{
		restClientGetter: settings.RESTClientGetter(),
		namespace:        settings.Namespace(),
	}
}

// Checker encapsulates all the data needed to run Checks against a Kubernetes cluster.
type Checker struct {
	restClientGetter genericclioptions.RESTClientGetter
	namespace        string

	k8s        kubernetes.Interface
	k8sVersion *version.Info
}

// Run runs all the checks, calling the callback function cb after each
// completes. It returns a boolean representing the overall success.
func (c *Checker) Run(checks []Check, cb func(*Result)) bool {
	pass := true

	for _, check := range checks {
		res := &Result{
			Name: check.name,
			Err:  check.run(c),
		}
		cb(res)
		if res.Err != nil {
			pass = false
			if check.fatal {
				break
			}
		}
	}

	return pass
}

func (c *Checker) checkAuthz(verb, group, version, resource, name string) error {
	ssar := &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: c.namespace,
				Verb:      verb,
				Group:     group,
				Version:   version,
				Resource:  resource,
				Name:      name,
			},
		},
	}

	result, err := c.k8s.
		AuthorizationV1().
		SelfSubjectAccessReviews().
		Create(context.TODO(), ssar, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if !result.Status.Allowed {
		return fmt.Errorf("cannot %s %s", verb, resource)
	}
	return nil
}

func authzCheck(verb, group, version, resource string) Check {
	return Check{
		name: fmt.Sprintf("can %s %s", verb, resource),
		run: func(c *Checker) error {
			return c.checkAuthz(verb, group, version, resource, "")
		},
	}
}

var (
	initK8sClient = Check{
		name:  "initialize Kubernetes client",
		fatal: true,
		run: func(c *Checker) (err error) {
			restConfig, err := c.restClientGetter.ToRESTConfig()
			if err != nil {
				return
			}
			c.k8s, err = kubernetes.NewForConfig(restConfig)
			return
		},
	}

	queryK8sAPI = Check{
		name:  "query Kubernetes API",
		fatal: true,
		run: func(c *Checker) (err error) {
			c.k8sVersion, err = c.k8s.Discovery().ServerVersion()
			return
		},
	}

	k8sVersion = Check{
		name: "Kubernetes version",
		run: func(c *Checker) (err error) {
			v, _ := semver.NewVersion(c.k8sVersion.String())
			minVer, _ := semver.NewConstraint("^1.15")
			if !minVer.Check(v) {
				err = fmt.Errorf("Kubernetes version %s does not match supported versions %s", c.k8sVersion, minVer)
			}
			return
		},
	}

	controlPlaneNs = Check{
		name: "control plane namespace doesn't exist",
		run: func(c *Checker) (err error) {
			_, err = c.k8s.CoreV1().Namespaces().Get(context.TODO(), c.namespace, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				err = nil
			} else if err == nil {
				err = errors.New("namespace already exists")
			}
			return
		},
	}

	modifyIptables = Check{
		name: "can modify iptables",
		run: func(c *Checker) error {
			psps, err := c.k8s.PolicyV1beta1().PodSecurityPolicies().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return err
			}

			if len(psps.Items) == 0 {
				// no PodSecurityPolicies found, assume PodSecurityPolicy admission controller is disabled
				return nil
			}

			// if PodSecurityPolicies are found, validate one exists that:
			// 1) permits usage
			// AND
			// 2) provides the specified capability
			requiredCaps := []string{"NET_ADMIN", "NET_RAW"}
			for _, psp := range psps.Items {
				err := c.checkAuthz(
					"use",
					"policy",
					"v1beta1",
					"podsecuritypolicies",
					psp.GetName(),
				)
				if err == nil {
					var caps []string
					for _, capability := range psp.Spec.AllowedCapabilities {
						caps = append(caps, string(capability))
					}
					set := sets.NewString(caps...)
					if set.Has("*") || set.HasAll(requiredCaps...) {
						return nil
					}
				}
			}
			return fmt.Errorf("No PodSecurityPolicies found providing the capabilities %q, sidecar injection will fail if the PSP admission controller is running", requiredCaps)
		},
	}
)

// PreinstallChecks validate that a Kubernetes clsuter is configured properly to
// allow a successful installation of OSM.
func PreinstallChecks() []Check {
	return []Check{
		// Order is important here as these two Checks populate fields in the Checker used by the other Checks.
		initK8sClient,
		queryK8sAPI,

		k8sVersion,
		controlPlaneNs,
		authzCheck("create", "", "v1", "namespaces"),
		// TODO: instead of enumerating the authz checks, render the Helm chart
		// to figure out which resources we need to be able to install.
		authzCheck("create", "apiextensions.k8s.io", "v1beta1", "customresourcedefinitions"),
		authzCheck("create", "rbac.authorization.k8s.io", "v1", "clusterroles"),
		authzCheck("create", "rbac.authorization.k8s.io", "v1", "clusterrolebindings"),
		authzCheck("create", "admissionregistration.k8s.io", "v1", "mutatingwebhookconfigurations"),
		authzCheck("create", "", "v1", "serviceaccounts"),
		authzCheck("create", "", "v1", "services"),
		authzCheck("create", "apps", "v1", "deployments"),
		authzCheck("create", "", "v1", "configmaps"),
		authzCheck("read", "", "v1", "secrets"),
		modifyIptables,
	}
}
