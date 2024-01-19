package yamlbasics_test

import (
	. "github.com/kong/go-apiops/yamlbasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("NodeSet", func() {
	node1 := yaml.Node{}
	node2 := yaml.Node{}
	node3 := yaml.Node{}
	node4 := yaml.Node{}

	set1 := NodeSet{&node1, &node2}
	set2 := NodeSet{&node2, &node3} // overlaps with set1 and set3
	set3 := NodeSet{&node3, &node4}
	set4 := NodeSet{&node4, &node4} // has duplicates
	setEmpty := NodeSet{}

	Describe("Intersection", func() {
		Context("when the mainset is empty", func() {
			It("should return an empty set", func() {
				intersection, remainder := setEmpty.Intersection(set1)
				// should be a copy
				Expect(remainder).ToNot(BeIdenticalTo(set1))
				Expect(remainder).ToNot(BeIdenticalTo(setEmpty))
				Expect(intersection).ToNot(BeIdenticalTo(set1))
				Expect(intersection).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(intersection).To(BeEmpty())
				Expect(remainder).To(BeEquivalentTo(set1))
			})
		})
		Context("when the subset is empty", func() {
			It("should return an empty set", func() {
				intersection, remainder := set1.Intersection(setEmpty)
				// should be a copy
				Expect(remainder).ToNot(BeIdenticalTo(set1))
				Expect(remainder).ToNot(BeIdenticalTo(setEmpty))
				Expect(intersection).ToNot(BeIdenticalTo(set1))
				Expect(intersection).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(intersection).To(BeEmpty())
				Expect(remainder).To(BeEmpty())
			})
		})
		Context("when the mainset and subset are not empty", func() {
			It("should return the intersection and the remainder", func() {
				intersection, remainder := set1.Intersection(set2)
				// should be a copy
				Expect(remainder).ToNot(BeIdenticalTo(set1))
				Expect(remainder).ToNot(BeIdenticalTo(set2))
				Expect(intersection).ToNot(BeIdenticalTo(set1))
				Expect(intersection).ToNot(BeIdenticalTo(set2))
				// check results
				Expect(intersection).To(BeEquivalentTo(NodeSet{&node2}))
				Expect(remainder).To(BeEquivalentTo(NodeSet{&node1}))
			})
		})
		Context("when the mainset and subset have no overlap", func() {
			It("should return an empty list and the remainder", func() {
				intersection, remainder := set1.Intersection(set3)
				// should be a copy
				Expect(remainder).ToNot(BeIdenticalTo(set1))
				Expect(remainder).ToNot(BeIdenticalTo(set3))
				Expect(intersection).ToNot(BeIdenticalTo(set1))
				Expect(intersection).ToNot(BeIdenticalTo(set3))
				// check results
				Expect(intersection).To(BeEquivalentTo(NodeSet{}))
				Expect(remainder).To(BeEquivalentTo(set3))
			})
		})
		Context("when the mainset and subset are not empty and subset has duplicates", func() {
			It("should return the intersection and the remainder", func() {
				intersection, remainder := set3.Intersection(set4)
				// should be a copy
				Expect(remainder).ToNot(BeIdenticalTo(set3))
				Expect(remainder).ToNot(BeIdenticalTo(set4))
				Expect(intersection).ToNot(BeIdenticalTo(set3))
				Expect(intersection).ToNot(BeIdenticalTo(set4))
				// check results
				Expect(intersection).To(BeEquivalentTo(NodeSet{&node4}))
				Expect(remainder).To(BeEquivalentTo(NodeSet{}))
			})
		})
		Context("when the mainset and subset are not empty and mainset has duplicates", func() {
			It("should return the intersection and the remainder", func() {
				intersection, remainder := set4.Intersection(set3)
				// should be a copy
				Expect(remainder).ToNot(BeIdenticalTo(set3))
				Expect(remainder).ToNot(BeIdenticalTo(set4))
				Expect(intersection).ToNot(BeIdenticalTo(set3))
				Expect(intersection).ToNot(BeIdenticalTo(set4))
				// check results
				Expect(intersection).To(BeEquivalentTo(NodeSet{&node4}))
				Expect(remainder).To(BeEquivalentTo(NodeSet{&node3}))
			})
		})
	})
	Describe("IsIntersection", func() {
		Context("when the mainset is empty", func() {
			It("should return false", func() {
				Expect(setEmpty.IsIntersection(set1)).To(BeFalse())
			})
		})
		Context("when the subset is empty", func() {
			It("should return true", func() {
				Expect(set1.IsIntersection(setEmpty)).To(BeTrue())
			})
		})
		Context("when the mainset and subset are empty", func() {
			It("should return true", func() {
				Expect(setEmpty.IsIntersection(setEmpty)).To(BeTrue())
			})
		})
		Context("when the mainset and subset are not empty", func() {
			It("should return true", func() {
				Expect(set3.IsIntersection(set4)).To(BeTrue())
			})
		})
		Context("when the mainset and subset have no overlap", func() {
			It("should return false", func() {
				Expect(set1.IsIntersection(set3)).To(BeFalse())
			})
		})
		Context("when the mainset and subset are not empty and subset has duplicates", func() {
			It("should return true", func() {
				Expect(set3.IsIntersection(set4)).To(BeTrue())
			})
		})
		Context("when the mainset and subset are not empty and mainset has duplicates", func() {
			It("should return true", func() {
				Expect(set4.IsIntersection(NodeSet{&node4})).To(BeTrue())
			})
		})
	})
	Describe("Union", func() {
		Context("when the mainset is empty", func() {
			It("should return a copy of the given set", func() {
				union := setEmpty.Union(set1)
				// should be a copy
				Expect(union).ToNot(BeIdenticalTo(set1))
				Expect(union).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(union).To(BeEquivalentTo(set1))
			})
		})
		Context("when the given set is empty", func() {
			It("should return a copy of the mainset", func() {
				union := set1.Union(setEmpty)
				// should be a copy
				Expect(union).ToNot(BeIdenticalTo(set1))
				Expect(union).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(union).To(BeEquivalentTo(set1))
			})
		})
		Context("when the mainset and given set are empty", func() {
			It("should return an empty list", func() {
				union := setEmpty.Union(setEmpty)
				// should be a copy
				Expect(union).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(union).To(BeEquivalentTo(setEmpty))
			})
		})
		Context("when the mainset and subset are not empty", func() {
			It("should return the union", func() {
				union := set1.Union(set2)
				// should be a copy
				Expect(union).ToNot(BeIdenticalTo(set1))
				Expect(union).ToNot(BeIdenticalTo(set2))
				// check results
				Expect(union).To(BeEquivalentTo(NodeSet{&node1, &node2, &node3}))
			})
		})
		Context("when the mainset and subset have no overlap", func() {
			It("should return the union", func() {
				union := set1.Union(set3)
				// should be a copy
				Expect(union).ToNot(BeIdenticalTo(set1))
				Expect(union).ToNot(BeIdenticalTo(set3))
				// check results
				Expect(union).To(BeEquivalentTo(NodeSet{&node1, &node2, &node3, &node4}))
			})
		})
	})
	Describe("Subtract", func() {
		Context("when the mainset is empty", func() {
			It("should return an empty set", func() {
				subtract := setEmpty.Subtract(set1)
				// should be a copy
				Expect(subtract).ToNot(BeIdenticalTo(set1))
				Expect(subtract).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(subtract).To(BeEquivalentTo(setEmpty))
			})
		})
		Context("when the given set is empty", func() {
			It("should return a copy of the mainset", func() {
				subtract := set1.Subtract(setEmpty)
				// should be a copy
				Expect(subtract).ToNot(BeIdenticalTo(set1))
				Expect(subtract).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(subtract).To(BeEquivalentTo(set1))
			})
		})
		Context("when the mainset and given set are empty", func() {
			It("should return an empty list", func() {
				subtract := setEmpty.Subtract(setEmpty)
				// should be a copy
				Expect(subtract).ToNot(BeIdenticalTo(setEmpty))
				// check results
				Expect(subtract).To(BeEquivalentTo(setEmpty))
			})
		})
		Context("when the mainset and subset are not empty", func() {
			It("should return the difference", func() {
				subtract := set1.Subtract(set2)
				// should be a copy
				Expect(subtract).ToNot(BeIdenticalTo(set1))
				Expect(subtract).ToNot(BeIdenticalTo(set2))
				// check results
				Expect(subtract).To(BeEquivalentTo(NodeSet{&node1}))
			})
		})
		Context("when the mainset and subset have no overlap", func() {
			It("should return the difference", func() {
				subtract := set1.Subtract(set3)
				// should be a copy
				Expect(subtract).ToNot(BeIdenticalTo(set1))
				Expect(subtract).ToNot(BeIdenticalTo(set3))
				// check results
				Expect(subtract).To(BeEquivalentTo(set1))
			})
		})
	})
})
