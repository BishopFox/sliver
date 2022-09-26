// +build windows

package jibberjabber_test

import (
	"regexp"

	. "github.com/bishopfox/sliver/implant/sliver/locale/jibberjabber"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	LOCALE_REGEXP    = "^[a-z]{2}-[A-Z]{2}$"
	LANGUAGE_REGEXP  = "^[a-z]{2}$"
	TERRITORY_REGEXP = "^[A-Z]{2}$"
)

var _ = Describe("Windows", func() {
	BeforeEach(func() {
		locale, err := DetectIETF()
		Ω(err).Should(BeNil())
		Ω(locale).ShouldNot(BeNil())
		Ω(locale).ShouldNot(Equal(""))
	})

	Describe("#DetectIETF", func() {
		It("detects correct IETF locale", func() {
			locale, _ := DetectIETF()
			matched, _ := regexp.MatchString(LOCALE_REGEXP, locale)
			Ω(matched).Should(BeTrue())
		})
	})

	Describe("#DetectLanguage", func() {
		It("detects correct Language", func() {
			language, _ := DetectLanguage()
			matched, _ := regexp.MatchString(LANGUAGE_REGEXP, language)
			Ω(matched).Should(BeTrue())
		})
	})

	Describe("#DetectLanguageTag", func() {
		It("detects correct Language Tag", func() {
			result, _ := DetectLanguageTag()
			base, _ := result.Base()
			matched, _ := regexp.MatchString(LANGUAGE_REGEXP, base.String())
			Ω(matched).Should(BeTrue())
			region, _ := result.Region()
			matched, _ = regexp.MatchString(TERRITORY_REGEXP, region.String())
			Ω(matched).Should(BeTrue())
		})
	})

	Describe("#DetectTerritory", func() {
		It("detects correct Territory", func() {
			territory, _ := DetectTerritory()
			matched, _ := regexp.MatchString(TERRITORY_REGEXP, territory)
			Ω(matched).Should(BeTrue())
		})
	})
})
