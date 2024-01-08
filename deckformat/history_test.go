package deckformat_test

import (
	. "github.com/kong/go-apiops/deckformat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("deckformat", func() {
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
