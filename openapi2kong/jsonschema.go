package openapi2kong

import (
	"encoding/json"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// dereferenceSchema walks the schema and adds every subschema to the seenBefore map.
// This is safe to recursive schemas.
func dereferenceSchema(sr *openapi3.SchemaRef, seenBefore map[string]*openapi3.Schema) {
	if sr == nil {
		return
	}

	if sr.Ref != "" {
		if seenBefore[sr.Ref] != nil {
			return
		}
		seenBefore[sr.Ref] = sr.Value
	}

	s := sr.Value

	for _, list := range []openapi3.SchemaRefs{s.AllOf, s.AnyOf, s.OneOf} {
		for _, s2 := range list {
			dereferenceSchema(s2, seenBefore)
		}
	}
	for _, s2 := range s.Properties {
		dereferenceSchema(s2, seenBefore)
	}
	for _, ref := range []*openapi3.SchemaRef{s.Not, s.AdditionalProperties, s.Items} {
		dereferenceSchema(ref, seenBefore)
	}
}

// extractSchema will extract a schema, including all sub-schemas/references and
// return it as a single JSONschema string. All components will be moved under the
// "#/definitions/" key.
func extractSchema(s *openapi3.SchemaRef) string {
	if s == nil || s.Value == nil {
		return ""
	}

	seenBefore := make(map[string]*openapi3.Schema)
	dereferenceSchema(s, seenBefore)

	var finalSchema map[string]interface{}
	// copy the primary schema
	jConf, _ := s.MarshalJSON()
	_ = json.Unmarshal(jConf, &finalSchema)

	// inject subschema's referenced
	if len(seenBefore) > 0 {
		definitions := make(map[string]interface{})
		for key, schema := range seenBefore {
			// copy the subschema
			var copySchema map[string]interface{}
			jConf, _ := schema.MarshalJSON()
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
