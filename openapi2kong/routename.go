package openapi2kong

import (
	"fmt"
	"strings"

	"github.com/kong/go-apiops/openapitools"
)

// pathKeySlug returns the slugified OAS path, as used in entity names.
func pathKeySlug(pathKey string, insoCompat bool) string {
	slug := openapitools.Slugify(insoCompat, pathKey)
	if !insoCompat && strings.HasSuffix(pathKey, "/") {
		// a common case is 2 paths, one with and one without a trailing "/" so to prevent
		// duplicate names being generated, we add a "~" suffix as a special case to cater
		// for different names. Better user solution is to use operation-id's.
		slug = slug + "~"
	}
	return slug
}

// joinName joins a base name and a name part, omitting the separator if the part is empty.
func joinName(baseName, part, nameConcatChar string) string {
	if part == "" {
		return baseName
	}
	return baseName + nameConcatChar + part
}

// buildOperationBaseName returns the base name for an operation, used as the route
// name and as namespace for UUID generation.
//
// By default the precedence is: the operation's x-kong-name -> operationId -> path+method.
// With fromPathMethod, the name is always built from the path and method, ignoring
// the x-kong-name and operationId of the operation, and the x-kong-name of the path.
// This makes the name predictable from the raw OAS path and method only.
func buildOperationBaseName(kongName, operationID, methodKey, pathKey,
	docBaseName, pathBaseName, nameConcatChar string, insoCompat, fromPathMethod bool,
) string {
	if fromPathMethod {
		// ignore any path level x-kong-name, use the actual path
		pathBaseName = joinName(docBaseName, pathKeySlug(pathKey, insoCompat), nameConcatChar)
		return pathMethodName(methodKey, pathBaseName, nameConcatChar, insoCompat)
	}

	if kongName != "" {
		if insoCompat {
			// an x-kong-name was provided, so build as "doc-name"
			return docBaseName + nameConcatChar + openapitools.Slugify(insoCompat, kongName)
		}
		// an x-kong-name was provided, so build as "doc-path-name"
		return pathBaseName + nameConcatChar + openapitools.Slugify(insoCompat, kongName)
	}

	if operationID != "" {
		// operation ID is provided, so build as "doc-operationid"
		return docBaseName + nameConcatChar + openapitools.Slugify(insoCompat, operationID)
	}

	// no operation ID provided, so build as "doc-path-method"
	return pathMethodName(methodKey, pathBaseName, nameConcatChar, insoCompat)
}

// pathMethodName builds the name as "doc-path-method".
func pathMethodName(methodKey, pathBaseName, nameConcatChar string, insoCompat bool) string {
	if insoCompat {
		return pathBaseName + nameConcatChar + openapitools.Slugify(insoCompat, strings.ToLower(methodKey))
	}
	return pathBaseName + nameConcatChar + openapitools.Slugify(insoCompat, methodKey)
}

// routeNameRegistry tracks the generated operation names, to detect duplicates when
// route names are built from path and method. In that mode, paths that slugify to the
// same value (eg. "/orders/{order_id}" and "/orders/{order-id}") produce the same
// name, which would otherwise silently create conflicting routes.
type routeNameRegistry map[string]string

// register records the name of an operation, and returns an error if it is a duplicate.
func (r routeNameRegistry) register(name, operation string) error {
	if existing, found := r[name]; found {
		return fmt.Errorf("duplicate route name '%s' generated from path and method for operations "+
			"'%s' and '%s'; paths must remain unique after conversion to names", name, existing, operation)
	}
	r[name] = operation
	return nil
}
