package openapi2kong

import (
	"encoding/json"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// dereferenceSchema walks the schema and adds every subschema to the seenBefore map.
// This is safe to recursive schemas.
func dereferenceSchema(sr *base.SchemaProxy, seenBefore map[string]*base.SchemaProxy) {
	if sr == nil {
		return
	}

	srRef := sr.GetReference()

	if srRef != "" {
		if seenBefore[srRef] != nil {
			return
		}
		seenBefore[srRef] = sr
	}

	s := sr.Schema()
	allSchemas := [][]*base.SchemaProxy{s.AllOf, s.AnyOf, s.OneOf}
	for _, schemas := range allSchemas {
		for _, schema := range schemas {
			dereferenceSchema(schema, seenBefore)
		}
	}

	schemaMap := s.Properties
	schema := schemaMap.First()
	for schema != nil {
		dereferenceSchema(schema.Value(), seenBefore)
		schema = schema.Next()
	}

	dereferenceSchema(s.Not, seenBefore)

	if s.AdditionalProperties != nil && s.AdditionalProperties.IsA() {
		dereferenceSchema(s.AdditionalProperties.A, seenBefore)
	}

	if s.Items != nil && s.Items.IsA() {
		dereferenceSchema(s.Items.A, seenBefore)
	}
}

// extractSchema will extract a schema, including all sub-schemas/references and
// return it as a single JSONschema string. All components will be moved under the
// "#/definitions/" key.
func extractSchema(s *base.SchemaProxy) string {
	if s == nil || s.Schema() == nil {
		return ""
	}

	seenBefore := make(map[string]*base.SchemaProxy)
	dereferenceSchema(s, seenBefore)

	finalSchema := make(map[string]interface{})

	if s.IsReference() {
		finalSchema["$ref"] = s.GetReference()
	} else {
		// copy the primary schema, if no ref string is present
		jConf, _ := s.Schema().MarshalJSON()
		_ = json.Unmarshal(jConf, &finalSchema)
	}

	// inject subschema's referenced
	if len(seenBefore) > 0 {
		definitions := make(map[string]interface{})
		for key, schema := range seenBefore {
			// copy the subschema
			var copySchema map[string]interface{}

			if schema.Schema() == nil {
				continue
			}

			jConf, _ := schema.Schema().MarshalJSON()
			_ = json.Unmarshal(jConf, &copySchema)

			// store under new key
			definitions[strings.Replace(key, "#/components/schemas/", "", 1)] = copySchema
		}
		finalSchema["definitions"] = definitions
	}

	result, _ := json.Marshal(finalSchema)
	// update the $ref values; this is safe because plain " (double-quotes) would be escaped if in actual values
	return strings.ReplaceAll(string(result), "\"$ref\":\"#/components/schemas/", "\"$ref\":\"#/definitions/")
}
