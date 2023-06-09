package deckformat_test

import (
	. "github.com/kong/go-apiops/deckformat"
	. "github.com/kong/go-apiops/filebasics"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("deckformat", func() {
	Describe("ConvertDBless", func() {
		It("converts a DBless format file", func() {
			jsonData := []byte(`{
				"consumer_groups": [
					{ "name": "A-team" },
					{ "name": "K-team" },
				],

				"consumer_group_plugins": [
					{	"name": "rate-limiting-advanced",
						"consumer_group": "A-team",
						"config": {
							"limit": [ 1000 ],
							"retry_after_jitter_max": 0,
							"window_size": [ 3600 ],
							"window_type": "sliding"
						}
					}
				],

				"consumers": [
					{
						"username": "tieske",
						"custom_id": "tieske-custom"
					},{
						"username": "foo",
						"custom_id": "bar"
					},{
						"username": "mr",
						"custom_id": "bean"
					}
				],

				"consumer_group_consumers": [
					{
						"consumer": "tieske",
						"consumer_group": "A-team"
					},{
						"consumer": "foo",
						"consumer_group": "K-team"
					},{
						"consumer": "mr",
						"consumer_group": "A-team"
					},{
						"consumer": "mr",
						"consumer_group": "K-team"
					}
				]
			}`)
			deckdata, err := ConvertDBless(MustDeserialize(&jsonData))
			Expect(err).To(BeNil())

			jsonDeck := MustSerialize(deckdata, OutputFormatJSON)
			Expect(*jsonDeck).Should(MatchJSON(`{
				"consumer_groups": [
					{
						"name": "A-team",
						"plugins": [
							{
								"config": {
									"limit": [
										1000
									],
									"retry_after_jitter_max": 0,
									"window_size": [
										3600
									],
									"window_type": "sliding"
								},
								"name": "rate-limiting-advanced"
							}
						]
					},{
						"name": "K-team"
					}
				],

				"consumers": [
					{
						"username": "tieske",
						"custom_id": "tieske-custom",
						"groups": [
							{ "name": "A-team" }
						]
					},
					{
						"username": "foo",
						"custom_id": "bar",
						"groups": [
								{ "name": "K-team" }
						]
					},
					{
						"username": "mr",
						"custom_id": "bean",
						"groups": [
							{ "name": "A-team" },
							{ "name": "K-team" }
						]
					}
				]
			}`))
		})

		It("'consumer_groups[].plugins' and 'consumer_groups[].consumer_group_plugins' are retained", func() {
			jsonData := []byte(`{
				"consumer_groups": [
					{
						"name": "A-team",
						"consumer_group_plugins": [
							{
								"name": "rate-limiting-advanced",
								"config": "plugin obj A1"
							}
					 	]
					},
					{
						"name": "K-team",
						"plugins": [
							{
								"name": "rate-limiting-advanced",
								"config": "plugin obj K1"
							}
					 	]
					},
				],

				"consumer_group_plugins": [
					{
						"name": "rate-limiting-advanced",
						"consumer_group": "A-team",
						"config": "plugin obj A2"
					},
					{
						"name": "rate-limiting-advanced",
						"consumer_group": "K-team",
						"config": "plugin obj K2"
					}
				]
			}`)
			deckdata, err := ConvertDBless(MustDeserialize(&jsonData))
			Expect(err).To(BeNil())

			jsonDeck := MustSerialize(deckdata, OutputFormatJSON)
			Expect(*jsonDeck).Should(MatchJSON(`{
				"consumer_groups": [
					{
						"name": "A-team",
						"plugins": [
							{
								"name": "rate-limiting-advanced",
								"config": "plugin obj A1"
							},{
								"name": "rate-limiting-advanced",
								"config": "plugin obj A2"
							}
					 	]
					},
					{
						"name": "K-team",
						"plugins": [
							{
								"name": "rate-limiting-advanced",
								"config": "plugin obj K1"
							},
							{
								"name": "rate-limiting-advanced",
								"config": "plugin obj K2"
							}
					 	]
					}
				]
			}`))
		})

		It("'consumer_groups[].plugins' and 'consumer_groups[].consumer_group_plugins' are mutually exclusive", func() {
			jsonData := []byte(`{
				"consumer_groups": [
					{
						"name": "A-team",
						"consumer_group_plugins": [
							{
								"name": "rate-limiting-advanced",
								"config": "plugin obj A1"
							}
					 	],
						"plugins": [
							{
								"name": "rate-limiting-advanced",
								"config": "plugin obj K1"
							}
					 	]
					},
				]
			}`)
			_, err := ConvertDBless(MustDeserialize(&jsonData))
			Expect(err.Error()).To(ContainSubstring(
				"entry 'consumer_groups[0]' contains both 'consumer_group_plugins' and 'plugins'"))
		})

		It("'consumer_groups' mentioned in 'consumer_group_plugins' must exist", func() {
			jsonData := []byte(`{
				"consumer_group_plugins": [
					{
						"name": "rate-limiting-advanced",
						"consumer_group": "A-team",
						"config": "plugin obj A1"
					}
				]
			}`)
			_, err := ConvertDBless(MustDeserialize(&jsonData))
			Expect(err.Error()).To(ContainSubstring(
				"consumer_group 'A-team' referenced by 'consumer_group_plugins[0]' not found"))
		})
	})
})
