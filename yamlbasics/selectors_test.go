package yamlbasics_test

import (
	. "github.com/kong/go-apiops/yamlbasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("selectors", func() {
	Describe("NewSelectorSet", func() {
		Context("when given a valid selector", func() {
			It("should return a non-empty set", func() {
				set, err := NewSelectorSet([]string{"$.a.b"})
				Expect(err).ToNot(HaveOccurred())
				Expect(set.IsEmpty()).To(BeFalse())
			})
		})
		Context("when given an invalid selector", func() {
			It("should return an error", func() {
				_, err := NewSelectorSet([]string{"$.a.b["})
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when given an empty selector", func() {
			It("should return an empty set", func() {
				set, err := NewSelectorSet([]string{})
				Expect(err).ToNot(HaveOccurred())
				Expect(set.IsEmpty()).To(BeTrue())
			})
		})
		Context("when given a nil selector", func() {
			It("should return an empty set", func() {
				set, err := NewSelectorSet(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(set.IsEmpty()).To(BeTrue())
			})
		})
	})
	Describe("IsEmpty", func() {
		Context("when the set is empty", func() {
			It("should return true", func() {
				set := SelectorSet{}
				Expect(set.IsEmpty()).To(BeTrue())
			})
		})
		Context("when the set is not empty", func() {
			It("should return false", func() {
				set, _ := NewSelectorSet([]string{"$.a.b"})
				Expect(set.IsEmpty()).To(BeFalse())
			})
		})
	})
	Describe("GetSources", func() {
		Context("when the set is empty", func() {
			It("should return an empty list", func() {
				set := SelectorSet{}
				Expect(set.GetSources()).To(BeEmpty())
			})
		})
		Context("when the set is not empty", func() {
			It("should return a copy of the sources", func() {
				sources := []string{"$.a.b", "$.c.d"}
				set, _ := NewSelectorSet(sources)
				Expect(set.GetSources()).To(Equal(sources))
				Expect(set.GetSources()).ToNot(BeIdenticalTo(sources))
			})
		})
	})
	Describe("Find", func() {
		Context("when the set is not initialized", func() {
			It("should panic", func() {
				set := SelectorSet{}
				node := &yaml.Node{}
				Expect(func() { set.Find(node) }).To(Panic())
			})
		})
		Context("when the set is empty", func() {
			It("should return an empty list", func() {
				set, _ := NewSelectorSet([]string{})
				node := &yaml.Node{}
				Expect(set.Find(node)).To(BeEmpty())
			})
		})
		Context("when the set is not empty", func() {
			Context("when the node is nil", func() {
				It("should panic", func() {
					set, _ := NewSelectorSet([]string{"$.a.b"})
					Expect(func() { set.Find(nil) }).To(Panic())
				})
			})
			Context("when the node is not nil", func() {
				It("should return a list of nodes", func() {
					set, _ := NewSelectorSet([]string{"$.a.b"})
					node := &yaml.Node{}
					Expect(set.Find(node)).ToNot(BeNil())
				})
			})
		})
	})
})
