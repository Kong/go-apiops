package tags_test

import (
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/tags"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("tags", func() {
	Describe("Tagger.SetData", func() {
		It("panics if data is nil", func() {
			Expect(func() {
				tagger := tags.Tagger{}
				tagger.SetData(nil)
			}).Should(PanicWith("data cannot be nil"))
		})

		It("does not panic if data is not nil", func() {
			Expect(func() {
				tagger := tags.Tagger{}
				tagger.SetData(map[string]interface{}{})
			}).ShouldNot(Panic())
		})
	})

	Describe("Tagger.SetSelectors", func() {
		It("allows nil, or 0 length", func() {
			tagger := tags.Tagger{}

			err := tagger.SetSelectors(nil)
			Expect(err).To(BeNil())

			err = tagger.SetSelectors([]string{})
			Expect(err).To(BeNil())
		})

		It("accepts a valid JSONpointer", func() {
			tagger := tags.Tagger{}
			err := tagger.SetSelectors([]string{"$..routes[*]", "$..services[*]"})
			Expect(err).To(BeNil())
		})

		It("fails on a bad JSONpointer", func() {
			tagger := tags.Tagger{}
			err := tagger.SetSelectors([]string{"bad one"})
			Expect(err).To(MatchError("selector 'bad one' is not a valid JSONpath expression; " +
				"invalid character ' ' at position 3, following \"bad\""))
		})
	})

	Describe("Tagger.RemoveTags", func() {
		It("removes tags, not changing order", func() {
			dataInput := []byte(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag1", "tag2"] },
					{ "name": "svc2", "tags": ["tag2", "tag3"] },
					{ "name": "svc3", "tags": ["tag3", "tag1"] }
				]}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.SetSelectors([]string{
				"$..anykey[*]",
			})
			tagger.RemoveTags([]string{
				"tag2",
			}, true)

			result := filebasics.MustSerialize(tagger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag1"] },
					{ "name": "svc2", "tags": ["tag3"] },
					{ "name": "svc3", "tags": ["tag3", "tag1"] }
				]}`))
		})

		It("uses default selectors if none specified", func() {
			dataInput := []byte(`
				{
					"services": [
						{ "name": "svc1", "tags": ["tag1", "tag2"] },
						{ "name": "svc2", "tags": ["tag2", "tag3"] },
						{ "name": "svc3", "tags": ["tag3", "tag1"] }
					],
					"anykey": [
						{ "name": "svc1", "tags": ["tag1", "tag2"] },
						{ "name": "svc2", "tags": ["tag2", "tag3"] },
						{ "name": "svc3", "tags": ["tag3", "tag1"] }
					]
				}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.RemoveTags([]string{
				"tag2",
			}, true)

			result := filebasics.MustSerialize(tagger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{
					"services": [
						{ "name": "svc1", "tags": ["tag1"] },
						{ "name": "svc2", "tags": ["tag3"] },
						{ "name": "svc3", "tags": ["tag3", "tag1"] }
					],
					"anykey": [
						{ "name": "svc1", "tags": ["tag1", "tag2"] },
						{ "name": "svc2", "tags": ["tag2", "tag3"] },
						{ "name": "svc3", "tags": ["tag3", "tag1"] }
					]
				}`))
		})

		It("removes empty tag array if set to", func() {
			dataInput := []byte(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag1", "tag2"] },
					{ "name": "svc2", "tags": ["tag2", "tag3"] },
					{ "name": "svc3", "tags": ["tag3", "tag1"] }
				]}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.SetSelectors([]string{
				"$..anykey[*]",
			})
			tagger.RemoveTags([]string{
				"tag2",
				"tag3",
			}, true)

			result := filebasics.MustSerialize(tagger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag1"] },
					{ "name": "svc2" },
					{ "name": "svc3", "tags": ["tag1"] }
				]}`))
		})

		It("keeps empty tag array if set to", func() {
			dataInput := []byte(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag1", "tag2"] },
					{ "name": "svc2", "tags": ["tag2", "tag3"] },
					{ "name": "svc3", "tags": ["tag3", "tag1"] }
				]}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.SetSelectors([]string{
				"$..anykey[*]",
			})
			tagger.RemoveTags([]string{
				"tag2",
				"tag3",
			}, false)

			result := filebasics.MustSerialize(tagger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag1"] },
					{ "name": "svc2", "tags": [] },
					{ "name": "svc3", "tags": ["tag1"] }
				]}`))
		})
	})

	Describe("Tagger.AddTags", func() {
		It("adds tags to existing tag arrays, in order provided", func() {
			dataInput := []byte(`
				{ "anykey": [
					{ "name": "svc1" },
					{ "name": "svc2", "tags": ["tag1"] },
					{ "name": "svc3", "tags": ["tag2", "tag3"] }
				]}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.SetSelectors([]string{
				"$..anykey[*]",
			})
			tagger.AddTags([]string{
				"tagX",
				"tagY",
			})

			result := filebasics.MustSerialize(tagger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tagX", "tagY"] },
					{ "name": "svc2", "tags": ["tag1", "tagX", "tagY"] },
					{ "name": "svc3", "tags": ["tag2", "tag3", "tagX", "tagY"] }
				]}`))
		})
	})

	Describe("Tagger.ListTags", func() {
		It("lists tags in sorted order, deduplicated", func() {
			dataInput := []byte(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag3", "tag2"] },
					{ "name": "svc2", "tags": ["tag2", "tag1"] },
					{ "name": "svc3", "tags": ["tag1", "tag3"] }
				]}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.SetSelectors([]string{
				"$..anykey[*]",
			})

			result, err := tagger.ListTags()
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal([]string{
				"tag1",
				"tag2",
				"tag3",
			}))
		})
	})

	Describe("Tagger.RemoveUnknownTags", func() {
		It("removes unknown tags without impacting order", func() {
			dataInput := []byte(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag3", "tag2"] },
					{ "name": "svc2", "tags": ["tag2", "tag1"] },
					{ "name": "svc3", "tags": ["tag1", "tag3"] }
				]}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.SetSelectors([]string{
				"$..anykey[*]",
			})

			err := tagger.RemoveUnknownTags([]string{
				"tag1",
			}, false)
			Expect(err).ToNot(HaveOccurred())

			result := filebasics.MustSerialize(tagger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{ "anykey": [
					{ "name": "svc1", "tags": [] },
					{ "name": "svc2", "tags": ["tag1"] },
					{ "name": "svc3", "tags": ["tag1"] }
				]}`))
		})

		It("removes empty tags arrays if set to", func() {
			dataInput := []byte(`
				{ "anykey": [
					{ "name": "svc1", "tags": ["tag3", "tag2"] },
					{ "name": "svc2", "tags": ["tag2", "tag1"] },
					{ "name": "svc3", "tags": ["tag1", "tag3"] }
				]}`)

			tagger := tags.Tagger{}
			tagger.SetData(filebasics.MustDeserialize(dataInput))
			tagger.SetSelectors([]string{
				"$..anykey[*]",
			})

			err := tagger.RemoveUnknownTags([]string{
				"tag1",
			}, true)
			Expect(err).ToNot(HaveOccurred())

			result := filebasics.MustSerialize(tagger.GetData(), filebasics.OutputFormatJSON)
			Expect(result).To(MatchJSON(`
				{ "anykey": [
					{ "name": "svc1" },
					{ "name": "svc2", "tags": ["tag1"] },
					{ "name": "svc3", "tags": ["tag1"] }
				]}`))
		})
	})
})
