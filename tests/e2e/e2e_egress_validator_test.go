package e2e

import (
	"context"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pol "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	. "github.com/openservicemesh/osm/tests/framework"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = OSMDescribe("Submit Egress Policy",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 5,
	},
	func() {
		Context("EgressValidator", func() {
			It("Tests a valid Egress Policy", func() {
				ctx := context.TODO()
				egressin := new(pol.Egress)
				egressin.SetName(uuid.New().String())
				egressin.Spec.Sources = []pol.SourceSpec{{Kind: "ServiceAccount", Name: "Scooby", Namespace: "Scooby"}}
				egressin.Spec.Ports = []pol.PortSpec{{Number: 666, Protocol: "ironmaiden"}}
				ns := Td.OsmNamespace
				Expect(ctx).ShouldNot(BeNil())
				Expect(ns).ShouldNot(BeNil())
				palpha1 := Td.PolicyClient
				Expect(palpha1).ShouldNot(BeNil())
				egressout, err := palpha1.PolicyV1alpha1().Egresses(Td.OsmNamespace).Create(ctx, egressin, v1.CreateOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(egressout).ShouldNot(BeNil())
			})
		})
	})
