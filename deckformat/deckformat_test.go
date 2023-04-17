package deckformat_test

import (
	. "github.com/kong/go-apiops/deckformat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("deckformat", func() {
	Describe("ParseFormatVersion", func() {
		It("parses a version", func() {
			data := map[string]interface{}{
				VersionKey: "123.456",
			}

			major, minor, err := ParseFormatVersion(data)
			Expect(major).To(Equal(123))
			Expect(minor).To(Equal(456))
			Expect(err).To(BeNil())
		})

		It("returns minor = 0 if omitted", func() {
			data := map[string]interface{}{
				VersionKey: "123",
			}

			major, minor, err := ParseFormatVersion(data)
			Expect(major).To(Equal(123))
			Expect(minor).To(Equal(0))
			Expect(err).To(BeNil())
		})

		Describe("returns an error if the version", func() {
			It("has more than 2 segments", func() {
				data := map[string]interface{}{
					VersionKey: "123.456.789",
				}

				major, minor, err := ParseFormatVersion(data)
				Expect(err).To(MatchError("expected field '._format_version' to be a string in 'x.y' format"))
				Expect(major).To(Equal(0))
				Expect(minor).To(Equal(0))
			})

			It("has a non-numeric major", func() {
				data := map[string]interface{}{
					VersionKey: "abc.456",
				}

				major, minor, err := ParseFormatVersion(data)
				Expect(err).To(MatchError("expected field '._format_version' to be a string in 'x.y' format"))
				Expect(major).To(Equal(0))
				Expect(minor).To(Equal(0))
			})

			It("has a non-numeric minor", func() {
				data := map[string]interface{}{
					VersionKey: "123.def",
				}

				major, minor, err := ParseFormatVersion(data)
				Expect(err).To(MatchError("expected field '._format_version' to be a string in 'x.y' format"))
				Expect(major).To(Equal(0))
				Expect(minor).To(Equal(0))
			})

			It("doesn't exist", func() {
				data := map[string]interface{}{}

				major, minor, err := ParseFormatVersion(data)
				Expect(err).To(MatchError("expected field '._format_version' to be a string in 'x.y' format"))
				Expect(major).To(Equal(0))
				Expect(minor).To(Equal(0))
			})

			It("doesn't exist, because data is nil", func() {
				major, minor, err := ParseFormatVersion(nil)
				Expect(err).To(MatchError("expected field '._format_version' to be a string in 'x.y' format"))
				Expect(major).To(Equal(0))
				Expect(minor).To(Equal(0))
			})
		})
	})

	Describe("application version", func() {
		It("ToolVersionSet/Get/String", func() {
			ToolVersionSet("my-name", "1.2.3", "commit-xyz")

			n, v, c := ToolVersionGet()
			Expect(n).To(BeIdenticalTo("my-name"))
			Expect(v).To(BeIdenticalTo("1.2.3"))
			Expect(c).To(BeIdenticalTo("commit-xyz"))

			Expect(ToolVersionString()).Should(Equal("my-name 1.2.3 (commit-xyz)"))

			Expect(func() {
				ToolVersionSet("another name", "1.2.3", "commit-xyz")
			}).Should(Panic())
		})
	})

	Describe("compatibility", func() {
		DescribeTable("CompatibleTransform",
			func(transform1 interface{}, transform2 interface{}, expected bool) {
				res := CompatibleTransform(
					map[string]interface{}{TransformKey: transform1},
					map[string]interface{}{TransformKey: transform2},
				)
				if expected {
					// compatible, then result is nil
					Expect(res).To(BeNil())
				} else {
					// not-compatible, then result is an error
					Expect(res).Should(HaveOccurred())
				}
			},
			// transform1, transform2, expected
			Entry("1", true, false, false),
			Entry("2", true, true, true),
			Entry("3", true, nil, true),
			Entry("4", false, false, true),
			Entry("5", false, true, false),
			Entry("6", false, nil, false),
			Entry("7", nil, false, false),
			Entry("8", nil, true, true),
			Entry("9", nil, nil, true),
		)

		DescribeTable("CompatibleVersion",
			func(version1 interface{}, version2 interface{}, expected bool) {
				res := CompatibleVersion(
					map[string]interface{}{VersionKey: version1},
					map[string]interface{}{VersionKey: version2},
				)
				if expected {
					// compatible, then result is nil
					Expect(res).To(BeNil())
				} else {
					// not-compatible, then result is an error
					Expect(res).Should(HaveOccurred())
				}
			},
			// version1, version2, expected
			Entry("same major is compatible", "1.1", "1.2", true),
			Entry("different major is incompatible", "1.1", "2.1", false),
			Entry("omitted version is compatible 1", "1.1", nil, true),
			Entry("omitted version is compatible 2", nil, "1.1", true),
			Entry("omitted version is compatible 3", nil, nil, true),
			Entry("bad version is incompatible 1", "bad", "1.1", false),
			Entry("bad version is incompatible 2", "bad", nil, false),
			Entry("bad version is incompatible 3", "1.1", "bad", false),
			Entry("bad version is incompatible 4", nil, "bad", false),
		)

		DescribeTable("CompatibleFile",
			func(version1 interface{}, transform1 interface{}, version2 interface{}, transform2 interface{}, expected bool) {
				res := CompatibleFile(
					map[string]interface{}{
						VersionKey:   version1,
						TransformKey: transform1,
					},
					map[string]interface{}{
						VersionKey:   version2,
						TransformKey: transform2,
					},
				)
				if expected {
					// compatible, then result is nil
					Expect(res).To(BeNil())
				} else {
					// not-compatible, then result is an error
					Expect(res).Should(HaveOccurred())
				}
			},
			// version1, version2, expected
			Entry("1", "1.1", true, "1.2", true, true),
			Entry("2", "1.1", true, "1.2", false, false),
			Entry("3", "1.1", true, "2.1", true, false),
		)
	})

	Describe("history", func() {
		It("the key is set to '_ignore'", func() {
			Expect(HistoryKey).To(Equal("_ignore"))
		})

		Describe("HistoryGet", func() {
			It("returns the history array", func() {
				hist := []interface{}{"one", "two"}
				data := map[string]interface{}{
					HistoryKey: hist,
				}
				res := HistoryGet(data)

				Expect(res).To(BeEquivalentTo(hist))
			})

			It("returns an empty array if none found", func() {
				res := HistoryGet(map[string]interface{}{})
				Expect(res).To(BeEquivalentTo([]interface{}{}))
			})

			It("returns an empty array if data-in is nil", func() {
				res := HistoryGet(nil)
				Expect(res).To(BeEquivalentTo([]interface{}{}))
			})

			It("appends the existing history entry if it's not an array", func() {
				data := map[string]interface{}{
					HistoryKey: "foobar",
				}
				res := HistoryGet(data)
				Expect(res).To(BeEquivalentTo([]interface{}{"foobar"}))
			})
		})

		It("HistoryNewEntry creates a new entry", func() {
			// ToolVersionSet("my-name", "1.2.3", "commit-xyz")
			cmd := "myCmd"
			entry := HistoryNewEntry(cmd)

			Expect(entry).To(BeEquivalentTo(map[string]interface{}{
				"command": cmd,
				"tool":    ToolVersionString(),
			}))
		})

		Describe("HistorySet", func() {
			PIt("sets the history array", func() {
				hist := []interface{}{"one", "two"}
				data := map[string]interface{}{}

				HistorySet(data, hist)
				res := data[HistoryKey].([]interface{})
				Expect(res).To(BeEquivalentTo(hist))
			})

			It("deletes history-array if nil", func() {
				data := map[string]interface{}{
					HistoryKey: "delete me",
				}
				HistorySet(data, nil)

				res, found := data[HistoryKey]
				Expect(res).To(BeNil())
				Expect(found).To(BeFalse())
			})
		})

		PDescribe("HistoryAppend", func() {
			It("adds an entry to an existing array", func() {
				hist := []interface{}{"one", "two"}
				data := map[string]interface{}{
					HistoryKey: hist,
				}
				HistoryAppend(data, "three")
				res := HistoryGet(data)

				Expect(res).To(BeEquivalentTo([]interface{}{"one", "two", "three"}))
			})

			It("creates an array if it doesn't exist", func() {
				data := map[string]interface{}{}

				HistoryAppend(data, "one")

				res := HistoryGet(data)
				Expect(res).To(BeEquivalentTo([]interface{}{"one"}))
			})
		})

		Describe("HistoryClear", func() {
			It("clears the history key", func() {
				data := map[string]interface{}{
					HistoryKey: "delete me",
				}

				HistoryClear(data)

				res, found := data[HistoryKey]
				Expect(res).To(BeNil())
				Expect(found).To(BeFalse())
			})
		})
	})
})
