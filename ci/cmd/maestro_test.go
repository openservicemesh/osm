package main

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/demo/cmd/common"
)

var _ = Describe("Test maestro", func() {

	Context("Test cutIt", func() {
		It("cuts it at success", func() {
			str := fmt.Sprintf("foo bar %s baz", common.Success)
			actual := cutIt(str)
			expected := fmt.Sprintf("foo bar %s", common.Success)
			Expect(actual).To(Equal(expected))
		})

		It("cuts it at failure", func() {
			str := fmt.Sprintf("foo bar %s baz baz", common.Failure)
			actual := cutIt(str)
			expected := fmt.Sprintf("foo bar %s", common.Failure)
			Expect(actual).To(Equal(expected))
		})
	})

})
