package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	mapset "github.com/deckarep/golang-set"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"

	"github.com/openservicemesh/osm/pkg/policy"
)

const policyCheckConflictsDesc = `
This command checks whether API resources of the same kind conflict.
`

const policyCheckConflictsExample = `
# To check if IngressBackend API resources conflict in the 'test' namespace
osm policy check-conflicts IngressBackend -n test
`

type policyCheckConflictsCmd struct {
	stdout       io.Writer
	resourceKind string
	policyClient policyClientset.Interface
	namespaces   []string
}

func newPolicyCheckConflicts(stdout io.Writer) *cobra.Command {
	policyCheckConflictsCmd := &policyCheckConflictsCmd{
		stdout: stdout,
	}

	cmd := &cobra.Command{
		Use:   "check-conflicts RESOURCE_KIND",
		Short: "check if an API resource conflicts with another resource of the same kind",
		Long:  policyCheckConflictsDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			policyCheckConflictsCmd.resourceKind = args[0]

			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig: %w", err)
			}

			policyClient, err := policyClientset.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Error initializing %s client: %w", policyv1alpha1.SchemeGroupVersion, err)
			}
			policyCheckConflictsCmd.policyClient = policyClient

			return policyCheckConflictsCmd.run()
		},
		Example: policyCheckConflictsExample,
	}

	f := cmd.Flags()
	f.StringSliceVarP(&policyCheckConflictsCmd.namespaces, "namespaces", "n", []string{}, "One or more namespaces to limit conflict checks to")

	return cmd
}

func (cmd *policyCheckConflictsCmd) run() error {
	var err error

	switch strings.ToLower(cmd.resourceKind) {
	case "ingressbackend":
		err = cmd.checkIngressBackendConflict()

	default:
		return fmt.Errorf("Invalid resource kind %s", cmd.resourceKind)
	}

	return err
}

func (cmd *policyCheckConflictsCmd) checkIngressBackendConflict() error {
	givenNsCount := len(cmd.namespaces)
	if givenNsCount != 1 {
		return fmt.Errorf("Requires single namespace specified by '-n|--namespace' to check for conflicts, got %d", givenNsCount)
	}

	ns := cmd.namespaces[0]

	ingressBackends, err := cmd.policyClient.PolicyV1alpha1().IngressBackends(ns).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Error listing IngressBackend resources in namespace %s: %w", ns, err)
	}

	conflictsExist := false
	computed := mapset.NewSet()
	for _, x := range ingressBackends.Items {
		for _, y := range ingressBackends.Items {
			if x.Name == y.Name {
				continue
			}

			xyVisitedKey := fmt.Sprintf("%s:%s", x.Name, y.Name)
			yxVisitedKey := fmt.Sprintf("%s:%s", y.Name, x.Name)

			if computed.Contains(xyVisitedKey) || computed.Contains(yxVisitedKey) {
				continue
			}
			computed.Add(xyVisitedKey)

			if conflicts := policy.DetectIngressBackendConflicts(x, y); len(conflicts) > 0 {
				fmt.Fprintf(cmd.stdout, "[+] IngressBackend %s/%s conflicts with %s/%s:\n", ns, x.Name, ns, y.Name)
				for _, err := range conflicts {
					fmt.Fprintf(cmd.stdout, "%s\n", err)
				}
				fmt.Fprintf(cmd.stdout, "\n")
				conflictsExist = true
			}
		}
	}

	if !conflictsExist {
		fmt.Fprintf(cmd.stdout, "No conflicts among IngressBackend resources in namespace %s\n", ns)
	}

	return nil
}
