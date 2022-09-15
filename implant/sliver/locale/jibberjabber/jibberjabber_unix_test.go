// +build darwin freebsd linux netbsd openbsd

package jibberjabber_test

import (
	"errors"
	"os"

	. "github.com/bishopfox/sliver/implant/sliver/locale/jibberjabber"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Unix", func() {
	AfterEach(func() {
		os.Setenv("LC_MESSAGES", "")
		os.Setenv("LC_ALL", "")
		os.Setenv("LANG", "en_US.UTF-8")
	})

	Describe("#DetectIETF", func() {
		Context("Returns IETF encoded locale", func() {
			It("should return the locale set to LC_MESSAGES", func() {
				os.Setenv("LC_MESSAGES", "fr_FR.UTF-8")
				result, _ := DetectIETF()
				Ω(result).Should(Equal("fr-FR"))
			})

			It("should return the locale set to LC_ALL if LC_MESSAGES isn't set", func() {
				os.Setenv("LC_ALL", "fr_FR.UTF-8")
				result, _ := DetectIETF()
				Ω(result).Should(Equal("fr-FR"))
			})

			It("should return the locale set to LANG if LC_ALL isn't set", func() {
				os.Setenv("LANG", "fr_FR.UTF-8")
				result, _ := DetectIETF()
				Ω(result).Should(Equal("fr-FR"))
			})

			It("should return an error if it cannot detect a locale", func() {
				os.Setenv("LANG", "")
				_, err := DetectIETF()
				Ω(errors.Is(err, ErrLangDetectFail)).Should(BeTrue())
			})
		})

		Context("when the locale is simply 'fr'", func() {
			BeforeEach(func() {
				os.Setenv("LANG", "fr")
			})

			It("should return the locale without a territory", func() {
				language, err := DetectIETF()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(language).Should(Equal("fr"))
			})
		})
	})

	Describe("#DetectLanguage", func() {
		Context("Returns encoded language", func() {
			It("should return the language set to LC_MESSAGES", func() {
				os.Setenv("LC_MESSAGES", "fr_FR.UTF-8")
				result, _ := DetectLanguage()
				Ω(result).Should(Equal("fr"))
			})

			It("should return the language set to LC_ALL if LC_MESSAGES isn't set", func() {
				os.Setenv("LC_ALL", "fr_FR.UTF-8")
				result, _ := DetectLanguage()
				Ω(result).Should(Equal("fr"))
			})

			It("should return the language set to LANG if LC_ALL isn't set", func() {
				os.Setenv("LANG", "fr_FR.UTF-8")
				result, _ := DetectLanguage()
				Ω(result).Should(Equal("fr"))
			})

			It("should return an error if it cannot detect a language", func() {
				os.Setenv("LANG", "")
				_, err := DetectLanguage()
				Ω(errors.Is(err, ErrLangDetectFail)).Should(BeTrue())
			})
		})
	})

	Describe("#DetectLanguageTag", func() {
		Context("Returns encoded language tag", func() {
			It("should return language tag corresponding to the language set to LC_MESSAGES", func() {
				os.Setenv("LC_MESSAGES", "fr_FR.UTF-8")
				result, _ := DetectLanguageTag()
				base, _ := result.Base()
				Ω(base.String()).Should(Equal("fr"))
				region, _ := result.Region()
				Ω(region.String()).Should(Equal("FR"))
			})

			It("should return language tag corresponding to the language set to LC_ALL if LC_MESSAGES isn't set", func() {
				os.Setenv("LC_ALL", "fr_FR.UTF-8")
				result, _ := DetectLanguageTag()
				base, _ := result.Base()
				Ω(base.String()).Should(Equal("fr"))
				region, _ := result.Region()
				Ω(region.String()).Should(Equal("FR"))
			})

			It("should return language tag corresponding to the language set to LANG if LC_ALL isn't set", func() {
				os.Setenv("LANG", "fr_FR.UTF-8")
				result, _ := DetectLanguageTag()
				base, _ := result.Base()
				Ω(base.String()).Should(Equal("fr"))
				region, _ := result.Region()
				Ω(region.String()).Should(Equal("FR"))
			})

			It("should return an error if it cannot detect and parse a language tag", func() {
				os.Setenv("LANG", "")
				_, err := DetectLanguageTag()
				Ω(errors.Is(err, ErrLangDetectFail)).Should(BeTrue())
			})
		})
	})

	Describe("#DetectTerritory", func() {
		Context("Returns encoded territory", func() {
			It("should return the territory set to LC_MESSAGES", func() {
				os.Setenv("LC_MESSAGES", "fr_FR.UTF-8")
				result, _ := DetectTerritory()
				Ω(result).Should(Equal("FR"))
			})

			It("should return the territory set to LC_ALL if LC_MESSAGES isn't set", func() {
				os.Setenv("LC_ALL", "fr_FR.UTF-8")
				result, _ := DetectTerritory()
				Ω(result).Should(Equal("FR"))
			})

			It("should return the territory set to LANG if LC_ALL isn't set", func() {
				os.Setenv("LANG", "fr_FR.UTF-8")
				result, _ := DetectTerritory()
				Ω(result).Should(Equal("FR"))
			})

			It("should return an error if it cannot detect a territory", func() {
				os.Setenv("LANG", "")
				_, err := DetectTerritory()
				Ω(errors.Is(err, ErrLangDetectFail)).Should(BeTrue())
			})
		})
	})

})
