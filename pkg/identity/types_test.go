package identity

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test pkg/service functions", func() {
	defer GinkgoRecover()

	Context("Test K8sServiceAccount struct methods", func() {
		namespace := uuid.New().String()
		serviceAccountName := uuid.New().String()
		sa := K8sServiceAccount{
			Namespace: namespace,
			Name:      serviceAccountName,
		}

		It("implements stringer interface correctly", func() {
			Expect(sa.String()).To(Equal(fmt.Sprintf("%s/%s", namespace, serviceAccountName)))
		})

		It("implements IsEmpty correctly", func() {
			Expect(sa.IsEmpty()).To(BeFalse())
			Expect(K8sServiceAccount{}.IsEmpty()).To(BeTrue())
		})

		It("implements ToServiceIdentity() correctly", func() {
			expected := ServiceIdentity(fmt.Sprintf("%s.%s.cluster.local", serviceAccountName, namespace))
			Expect(sa.ToServiceIdentity()).To(Equal(expected))
		})

		It("implements ServiceIdentity.ToK8sServiceAccount() correctly", func() {
			serviceIdent := ServiceIdentity(fmt.Sprintf("%s.%s.cluster.local", serviceAccountName, namespace))
			expectedServiceAccount := sa
			Expect(serviceIdent.ToK8sServiceAccount()).To(Equal(expectedServiceAccount))
		})
	})
})

func TestUnmarshalK8sServiceAccount(t *testing.T) {
	assert := tassert.New(t)
	require := trequire.New(t)

	namespace := "randomNamespace"
	serviceName := "randomServiceAccountName"
	svcAccount := &K8sServiceAccount{
		Namespace: namespace,
		Name:      serviceName,
	}
	str := svcAccount.String()
	fmt.Println(str)

	testCases := []struct {
		name              string
		expectedErr       bool
		serviceAccountStr string
	}{
		{
			name:              "successfully unmarshal service account",
			expectedErr:       false,
			serviceAccountStr: "randomNamespace/randomServiceAccountName",
		},
		{
			name:              "incomplete namespaced service account name 1",
			expectedErr:       true,
			serviceAccountStr: "/svnc",
		},
		{
			name:              "incomplete namespaced service account name 2",
			expectedErr:       true,
			serviceAccountStr: "svnc/",
		},
		{
			name:              "incomplete namespaced service account name 3",
			expectedErr:       true,
			serviceAccountStr: "/svnc/",
		},
		{
			name:              "incomplete namespaced service account name 3",
			expectedErr:       true,
			serviceAccountStr: "/",
		},
		{
			name:              "incomplete namespaced service account name 3",
			expectedErr:       true,
			serviceAccountStr: "",
		},
		{
			name:              "incomplete namespaced service account name 3",
			expectedErr:       true,
			serviceAccountStr: "test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := UnmarshalK8sServiceAccount(tc.serviceAccountStr)
			if tc.expectedErr {
				assert.NotNil(err)
			} else {
				require.Nil(err)
				assert.Equal(svcAccount, actual)
			}
		})
	}
}
