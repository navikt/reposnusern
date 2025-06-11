package dbwriter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
)

func TestDbwriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DBWriter – Utils")
}

var _ = Describe("Utils-funksjoner for trygg konvertering", func() {

	Describe("safeLicense", func() {
		It("skal returnere tom streng ved nil input", func() {
			Expect(dbwriter.SafeLicense(nil)).To(Equal(""))
		})

		It("skal returnere riktig SPDX-ID når input er gyldig", func() {
			license := &struct{ SpdxID string }{SpdxID: "MIT"}
			Expect(dbwriter.SafeLicense(license)).To(Equal("MIT"))
		})
	})

	Describe("safeString", func() {
		It("skal returnere tom streng for nil", func() {
			Expect(dbwriter.SafeString(nil)).To(Equal(""))
		})

		It("skal returnere original tekst når input er en string", func() {
			Expect(dbwriter.SafeString("hello")).To(Equal("hello"))
		})
	})
})
