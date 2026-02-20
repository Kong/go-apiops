package lint_test

import (
	. "github.com/kong/go-apiops/lint"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Lint", func() {
	Describe("Severity", func() {
		It("parses valid severity strings", func() {
			sev, err := ParseSeverity("error")
			Expect(err).ToNot(HaveOccurred())
			Expect(sev).To(Equal(SeverityError))

			sev, err = ParseSeverity("warn")
			Expect(err).ToNot(HaveOccurred())
			Expect(sev).To(Equal(SeverityWarn))

			sev, err = ParseSeverity("info")
			Expect(err).ToNot(HaveOccurred())
			Expect(sev).To(Equal(SeverityInfo))

			sev, err = ParseSeverity("hint")
			Expect(err).ToNot(HaveOccurred())
			Expect(sev).To(Equal(SeverityHint))
		})

		It("parses case-insensitively", func() {
			sev, err := ParseSeverity("ERROR")
			Expect(err).ToNot(HaveOccurred())
			Expect(sev).To(Equal(SeverityError))
		})

		It("returns error for unknown severity", func() {
			_, err := ParseSeverity("critical")
			Expect(err).To(HaveOccurred())
		})

		It("converts severity to string", func() {
			Expect(SeverityError.String()).To(Equal("error"))
			Expect(SeverityWarn.String()).To(Equal("warn"))
			Expect(SeverityInfo.String()).To(Equal("info"))
			Expect(SeverityHint.String()).To(Equal("hint"))
		})
	})

	Describe("ParseRuleset", func() {
		It("parses a valid ruleset", func() {
			data := []byte(`
rules:
  my-rule:
    description: "Test rule"
    given: $.name
    severity: error
    then:
      function: truthy
`)
			rs, err := ParseRuleset(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Rules).To(HaveLen(1))
			Expect(rs.Rules["my-rule"].Description).To(Equal("Test rule"))
			Expect(rs.Rules["my-rule"].Given).To(Equal([]string{"$.name"}))
			Expect(rs.Rules["my-rule"].Severity).To(Equal(SeverityError))
			Expect(rs.Rules["my-rule"].Then).To(HaveLen(1))
			Expect(rs.Rules["my-rule"].Then[0].Function).To(Equal("truthy"))
		})

		It("parses ruleset with array given", func() {
			data := []byte(`
rules:
  my-rule:
    given:
      - $.name
      - $.title
    then:
      function: truthy
`)
			rs, err := ParseRuleset(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Rules["my-rule"].Given).To(Equal([]string{"$.name", "$.title"}))
		})

		It("parses ruleset with array then", func() {
			data := []byte(`
rules:
  my-rule:
    given: "$"
    then:
      - field: title
        function: truthy
      - field: description
        function: truthy
`)
			rs, err := ParseRuleset(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Rules["my-rule"].Then).To(HaveLen(2))
			Expect(rs.Rules["my-rule"].Then[0].Field).To(Equal("title"))
			Expect(rs.Rules["my-rule"].Then[1].Field).To(Equal("description"))
		})

		It("parses ruleset with functionOptions", func() {
			data := []byte(`
rules:
  my-rule:
    given: $.name
    then:
      function: pattern
      functionOptions:
        match: "^[a-z]+"
`)
			rs, err := ParseRuleset(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Rules["my-rule"].Then[0].FunctionOptions).To(HaveKeyWithValue("match", "^[a-z]+"))
		})

		It("defaults severity to warn", func() {
			data := []byte(`
rules:
  my-rule:
    given: $.name
    then:
      function: truthy
`)
			rs, err := ParseRuleset(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Rules["my-rule"].Severity).To(Equal(SeverityWarn))
		})

		It("returns error for invalid YAML", func() {
			data := []byte(`{invalid yaml`)
			_, err := ParseRuleset(data)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for missing rules key", func() {
			data := []byte(`something: else`)
			_, err := ParseRuleset(data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing required 'rules' key"))
		})

		It("returns error for missing given field", func() {
			data := []byte(`
rules:
  bad-rule:
    then:
      function: truthy
`)
			_, err := ParseRuleset(data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("given"))
		})

		It("returns error for missing then field", func() {
			data := []byte(`
rules:
  bad-rule:
    given: $.name
`)
			_, err := ParseRuleset(data)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("then"))
		})

		It("parses message field", func() {
			data := []byte(`
rules:
  my-rule:
    given: $.name
    message: "Custom message"
    then:
      function: truthy
`)
			rs, err := ParseRuleset(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(rs.Rules["my-rule"].Message).To(Equal("Custom message"))
		})
	})

	Describe("Lint (integration)", func() {
		It("validates a simple pattern rule", func() {
			ruleset := []byte(`
rules:
  version-check:
    description: "Validate version"
    given: $._format_version
    severity: error
    then:
      function: pattern
      functionOptions:
        match: "^3.1$"
`)
			document := []byte(`_format_version: "1.0"`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Severity).To(Equal(SeverityError))
			Expect(results[0].Message).To(ContainSubstring("does not match"))
		})

		It("passes when pattern matches", func() {
			ruleset := []byte(`
rules:
  version-check:
    description: "Validate version"
    given: $._format_version
    severity: error
    then:
      function: pattern
      functionOptions:
        match: "^3.1$"
`)
			document := []byte(`_format_version: "3.1"`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("validates truthy with field", func() {
			ruleset := []byte(`
rules:
  required-fields:
    description: "Must have title"
    given: "$"
    severity: warn
    then:
      field: title
      function: truthy
`)
			document := []byte(`
name: "My API"
description: "A test API"
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Severity).To(Equal(SeverityWarn))
		})

		It("validates multiple then entries", func() {
			ruleset := []byte(`
rules:
  required-fields:
    description: "Must have fields"
    given: "$"
    severity: warn
    then:
      - field: title
        function: truthy
      - field: description
        function: truthy
`)
			document := []byte(`
title: "My API"
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			// title exists (truthy), description doesn't (also truthy violation)
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("truthy"))
		})

		It("validates enumeration function", func() {
			ruleset := []byte(`
rules:
  valid-protocol:
    description: "Protocol must be http or https"
    given: $.protocol
    severity: error
    then:
      function: enumeration
      functionOptions:
        values:
          - http
          - https
`)
			document := []byte(`protocol: ftp`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("does not match"))
		})

		It("validates casing function", func() {
			ruleset := []byte(`
rules:
  camel-case-name:
    description: "Name should be camelCased"
    given: $.name
    severity: warn
    then:
      function: casing
      functionOptions:
        type: camel
`)
			document := []byte(`name: MyBadName`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("camel"))
		})

		It("validates length function on array", func() {
			ruleset := []byte(`
rules:
  tag-count:
    description: "Must have 1-3 tags"
    given: "$"
    severity: warn
    then:
      field: tags
      function: length
      functionOptions:
        min: 1
        max: 3
`)
			document := []byte(`
tags:
  - name: a
  - name: b
  - name: c
  - name: d
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("longer"))
		})

		It("validates schema function", func() {
			ruleset := []byte(`
rules:
  servers-present:
    description: "Must have servers array"
    given: "$"
    severity: error
    then:
      field: servers
      function: schema
      functionOptions:
        schema:
          type: array
          minItems: 1
`)
			document := []byte(`
servers: []
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("items"))
		})

		It("validates or function", func() {
			ruleset := []byte(`
rules:
  descriptive-text:
    description: "Must have title or description"
    given: "$"
    severity: warn
    then:
      function: or
      functionOptions:
        properties:
          - title
          - description
`)
			document := []byte(`
name: "test"
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("at least one"))
		})

		It("validates xor function", func() {
			ruleset := []byte(`
rules:
  value-xor-external:
    description: "Must have value or externalValue, not both"
    given: "$"
    severity: error
    then:
      function: xor
      functionOptions:
        properties:
          - value
          - externalValue
`)
			document := []byte(`
value: "test"
externalValue: "http://example.com"
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("just one"))
		})

		It("validates defined function", func() {
			ruleset := []byte(`
rules:
  must-have-info:
    description: "Must have info"
    given: "$"
    severity: error
    then:
      field: info
      function: defined
`)
			document := []byte(`
name: "test"
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("defined"))
		})

		It("validates undefined function", func() {
			ruleset := []byte(`
rules:
  no-x-internal:
    description: "Should not have x-internal"
    given: "$"
    severity: warn
    then:
      field: x-internal
      function: undefined
`)
			document := []byte(`
name: "test"
x-internal: true
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Message).To(ContainSubstring("undefined"))
		})

		It("validates multiple rules", func() {
			ruleset := []byte(`
rules:
  has-name:
    description: "Must have name"
    given: "$"
    severity: error
    then:
      field: name
      function: truthy
  has-version:
    description: "Must have version"
    given: "$"
    severity: warn
    then:
      field: version
      function: truthy
`)
			document := []byte(`
description: "test"
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
		})

		It("returns error for invalid ruleset", func() {
			ruleset := []byte(`{invalid`)
			document := []byte(`name: test`)
			_, err := Lint(ruleset, document, "test.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid document", func() {
			ruleset := []byte(`
rules:
  my-rule:
    given: $.name
    then:
      function: truthy
`)
			document := []byte("\t- invalid")
			_, err := Lint(ruleset, document, "test.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("handles the docs/lint/version.yaml example ruleset", func() {
			ruleset := []byte(`
rules:
  version-check:
    description: "Validate version 3.1 for decK files"
    given: $._format_version
    severity: error
    then:
      function: pattern
      functionOptions:
        match: "^3.1$"
`)
			// Should fail for version 1.0
			doc1 := []byte(`_format_version: "1.0"`)
			results, err := Lint(ruleset, doc1, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Severity).To(Equal(SeverityError))

			// Should pass for version 3.1
			doc2 := []byte(`_format_version: "3.1"`)
			results, err = Lint(ruleset, doc2, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		It("handles the docs/lint/https.yaml example ruleset", func() {
			ruleset := []byte(`
rules:
  service-https-check:
    description: "Ensure https usage in Kong GW Services"
    given: $.services[*].protocol
    severity: error
    then:
      function: pattern
      functionOptions:
        match: "^https$"
`)
			doc := []byte(`
services:
  - name: svc1
    protocol: https
  - name: svc2
    protocol: http
`)
			results, err := Lint(ruleset, doc, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Severity).To(Equal(SeverityError))
			Expect(results[0].Message).To(ContainSubstring("does not match"))
		})
	})

	Describe("LintResult", func() {
		It("formats result as string", func() {
			r := LintResult{
				Message:  "test message",
				Severity: SeverityError,
				Line:     10,
				Column:   5,
				RuleName: "test-rule",
			}
			Expect(r.String()).To(Equal("[error][10:5] test-rule: test message"))
		})

		It("formats result without location", func() {
			r := LintResult{
				Message:  "test message",
				Severity: SeverityWarn,
				RuleName: "test-rule",
			}
			Expect(r.String()).To(Equal("[warn]test-rule: test message"))
		})
	})

	Describe("SortResults", func() {
		It("sorts by severity then line number", func() {
			results := []LintResult{
				{Severity: SeverityInfo, Line: 1},
				{Severity: SeverityError, Line: 10},
				{Severity: SeverityError, Line: 5},
				{Severity: SeverityWarn, Line: 1},
			}
			SortResults(results)
			Expect(results[0].Severity).To(Equal(SeverityError))
			Expect(results[0].Line).To(Equal(5))
			Expect(results[1].Severity).To(Equal(SeverityError))
			Expect(results[1].Line).To(Equal(10))
			Expect(results[2].Severity).To(Equal(SeverityWarn))
			Expect(results[3].Severity).To(Equal(SeverityInfo))
		})
	})

	Describe("CountBySeverity", func() {
		It("counts results at or above the given threshold", func() {
			results := []LintResult{
				{Severity: SeverityError},
				{Severity: SeverityWarn},
				{Severity: SeverityInfo},
				{Severity: SeverityHint},
			}
			Expect(CountBySeverity(results, SeverityError)).To(Equal(1))
			Expect(CountBySeverity(results, SeverityWarn)).To(Equal(2))
			Expect(CountBySeverity(results, SeverityInfo)).To(Equal(3))
			Expect(CountBySeverity(results, SeverityHint)).To(Equal(4))
		})
	})

	Describe("typedEnum integration", func() {
		It("validates typedEnum via lint", func() {
			ruleset := []byte(`
rules:
  typed-enum-check:
    description: "Enum values must match type"
    given: $.properties.status
    severity: error
    then:
      function: typedEnum
`)
			document := []byte(`
properties:
  status:
    type: string
    enum:
      - active
      - inactive
`)
			results, err := Lint(ruleset, document, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("alphabetical integration", func() {
		It("validates alphabetical tags", func() {
			ruleset := []byte(`
rules:
  tags-alphabetical:
    description: "Tags must be alphabetical"
    given: "$"
    severity: warn
    then:
      field: tags
      function: alphabetical
      functionOptions:
        keyedBy: name
`)
			// Sorted tags
			doc1 := []byte(`
tags:
  - name: alpha
  - name: beta
  - name: gamma
`)
			results, err := Lint(ruleset, doc1, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(BeEmpty())

			// Unsorted tags
			doc2 := []byte(`
tags:
  - name: gamma
  - name: alpha
  - name: beta
`)
			results, err = Lint(ruleset, doc2, "test.yaml")
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(1))
		})
	})
})
