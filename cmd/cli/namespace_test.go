package main

import (
	"bytes"
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-service-mesh/osm/pkg/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"
)

var (
	testNamespace     = "namespace"
	testMeshName      = "mesh"
	incorrectMeshName = "incorrectMesh"
)

var _ = Describe("Running the namespace add command", func() {

	Describe("with pre-existing namespace", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			fakeClientSet = fake.NewSimpleClientset()

			nsSpec := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
			fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

			namespaceAddCmd := &namespaceAddCmd{
				out:       out,
				meshName:  testMeshName,
				namespace: testNamespace,
				clientSet: fakeClientSet,
			}

			err = namespaceAddCmd.run()
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal(fmt.Sprintf("Namespace [%s] succesfully added to mesh [%s]\n", testNamespace, testMeshName)))
		})
	})

	Describe("with non-existent namespace", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			fakeClientSet = fake.NewSimpleClientset()

			namespaceAddCmd := &namespaceAddCmd{
				out:       out,
				meshName:  testMeshName,
				namespace: testNamespace,
				clientSet: fakeClientSet,
			}

			err = namespaceAddCmd.run()
		})

		It("should error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Could not label namespace [%s]: namespaces \"%s\" not found", testNamespace, testNamespace)))
		})
	})
})

var _ = Describe("Running the namespace remove command", func() {

	Describe("with pre-existing namespace and correct label", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			fakeClientSet = fake.NewSimpleClientset()

			labelMap := make(map[string]string)
			labelMap[constants.OSMKubeResourceMonitorAnnotation] = testMeshName
			nsSpec := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespace,
					Labels: labelMap,
				},
			}
			fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

			namespaceRemoveCmd := &namespaceRemoveCmd{
				out:       out,
				meshName:  testMeshName,
				namespace: testNamespace,
				clientSet: fakeClientSet,
			}

			err = namespaceRemoveCmd.run()
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal(fmt.Sprintf("Namespace [%s] succesfully removed from mesh [%s]\n", testNamespace, testMeshName)))
		})
	})

	Describe("with pre-existing namespace and incorrect label", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			fakeClientSet = fake.NewSimpleClientset()

			labelMap := make(map[string]string)
			labelMap[constants.OSMKubeResourceMonitorAnnotation] = testMeshName
			nsSpec := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   testNamespace,
					Labels: labelMap,
				},
			}
			fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

			namespaceRemoveCmd := &namespaceRemoveCmd{
				out:       out,
				meshName:  incorrectMeshName,
				namespace: testNamespace,
				clientSet: fakeClientSet,
			}

			err = namespaceRemoveCmd.run()
		})

		It("should error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Namespace belongs to mesh [%s], not mesh [%s]. Please specify the correct mesh", testMeshName, incorrectMeshName)))

		})
	})

	Describe("with pre-existing namespace and no label", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			fakeClientSet = fake.NewSimpleClientset()

			nsSpec := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			}
			fakeClientSet.CoreV1().Namespaces().Create(context.TODO(), nsSpec, metav1.CreateOptions{})

			namespaceRemoveCmd := &namespaceRemoveCmd{
				out:       out,
				meshName:  testMeshName,
				namespace: testNamespace,
				clientSet: fakeClientSet,
			}

			err = namespaceRemoveCmd.run()
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message saying namespace was not part of mesh", func() {
			Expect(out.String()).To(Equal(fmt.Sprintf("Namespace [%s] already does not belong to any mesh\n", testNamespace)))
		})
	})

	Describe("with non-existent namespace", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			fakeClientSet = fake.NewSimpleClientset()

			namespaceRemoveCmd := &namespaceRemoveCmd{
				out:       out,
				meshName:  testMeshName,
				namespace: testNamespace,
				clientSet: fakeClientSet,
			}

			err = namespaceRemoveCmd.run()
		})

		It("should error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("Could not get namespace [%s]: namespaces \"%s\" not found", testNamespace, testNamespace)))
		})
	})
})
