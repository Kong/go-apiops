package deckformat

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/jsonpath/pkg/jsonpath"
	"gopkg.in/yaml.v3"
)

// EntityPointers is a map of entity names to an array of JSONpointers that can be used to find
// all of those entities in a deck file. For example; credentials typically can be under
// their own top-level key, or nested under a consumer.
// Additional sets are dynamically added (capitalized) for other common uses. e.g.
// "PluginOwners" is a list of all entities that can hold plugins.
var EntityPointers = map[string][]string{
	// list created from the deck source code, looking at: deck/types/*.go
	"acls": {
		"$.acls[*]",
		"$.consumers[*].acls[*]", // decK specific format for handling many-2-many relationships (consumers <-> groups	)
	},
	"basicauth_credentials": {
		"$.basicauth_credentials[*]",
		"$.consumers[*].basicauth_credentials[*]",
	},
	"ca_certificates": {
		"$.ca_certificates[*]",
	},
	"certificates": {
		"$.certificates[*]",
	},
	"consumer_group_consumers": {
		"$.consumer_group_consumers[*]",
	},
	"consumer_group_plugins": { // deprecated in Kong 3.4
		"$.consumer_group_plugins[*]",
		"$.consumer_groups[*].consumer_group_plugins[*]",
	},
	"consumer_groups": {
		"$.consumer_groups[*]",
	},
	"consumers": {
		"$.consumers[*]",
	},
	"document_objects": {
		"$.document_objects[*]",
		"$.services[*].document_objects[*]",
	},
	"hmacauth_credentials": {
		"$.hmacauth_credentials[*]",
		"$.consumers[*].hmacauth_credentials[*]",
	},
	"jwt_secrets": {
		"$.jwt_secrets[*]",
		"$.consumers[*].jwt_secrets[*]",
	},
	"keyauth_credentials": {
		"$.keyauth_credentials[*]",
		"$.consumers[*].keyauth_credentials[*]",
	},
	"mtls_auth_credentials": {
		"$.mtls_auth_credentials[*]",
		"$.consumers[*].mtls_auth_credentials[*]",
		"$.ca_certificates[*].mtls_auth_credentials[*]",
	},
	"oauth2_credentials": {
		"$.oauth2_credentials[*]",
		"$.consumers[*].oauth2_credentials[*]",
	},
	"plugins": {
		"$.plugins[*]",
		"$.routes[*].plugins[*]",
		"$.services[*].plugins[*]",
		"$.services[*].routes[*].plugins[*]",
		"$.consumers[*].plugins[*]",
		"$.consumer_group_plugins[*]",                    // the dbless format, deprecated in Kong 3.4
		"$.consumer_groups[*].consumer_group_plugins[*]", // the dbless format, deprecated in Kong 3.4
		"$.consumer_groups[*].plugins[*]",                // the deck format + new Kong 3.4 implementation
	},
	"rbac_role_endpoints": {
		"$.rbac_role_endpoints[*]",
		"$.rbac_roles[*].rbac_role_endpoints[*]",
	},
	"rbac_role_entities": {
		"$.rbac_role_entities[*]",
		"$.rbac_roles[*].rbac_role_entities[*]",
	},
	"rbac_roles": {
		"$.rbac_roles[*]",
	},
	"routes": {
		"$.routes[*]",
		"$.services[*].routes[*]",
	},
	"services": {
		"$.services[*]",
	},
	"snis": {
		"$.snis[*]",
		"$.certificates[*].snis[*]",
	},
	"targets": {
		"$.targets[*]",
		"$.upstreams[*].targets[*]",
		"$.certificates[*].upstreams[*].targets[*]",
	},
	"upstreams": {
		"$.upstreams[*]",
		"$.certificates[*].upstreams[*]",
	},
	"vaults": {
		"$.vaults[*]",
	},
}

// initPointerCollections will initialize sub-lists of useful pointer combinations.
func initPointerCollections() {
	// all entities that can hold tags (eg. have a "tags" array)
	TagOwners := make([]string, 0)
	for _, selectors := range EntityPointers {
		TagOwners = append(TagOwners, selectors...)
	}

	// all entities that can hold plugins (eg. have a "plugins" array)
	PluginOwners := make([]string, 0)
	for _, selector := range EntityPointers["plugins"] {
		if strings.HasSuffix(selector, ".plugins[*]") {
			selector := strings.TrimSuffix(selector, ".plugins[*]")
			PluginOwners = append(PluginOwners, selector)
		}
	}

	// Store them in the overall list, capitalized to prevent colissions
	EntityPointers["TagOwners"] = TagOwners
	EntityPointers["PluginOwners"] = PluginOwners
}

// getAvailableEntities returns a list of all entity types that can be found in the data.
// This is a list of the keys in the EntityPointers map.
func getAvailableEntities() []string {
	availableEntities := make([]string, 0)
	for entity := range EntityPointers {
		availableEntities = append(availableEntities, entity)
	}
	return availableEntities
}

// GetEntities returns all entities of a given type found within the yamlNode.
// The result will never be nil, but can be an empty array. The deckfile may be nil, the
// entityType must be a valid entity type, or it will panic.
func GetEntities(deckfile *yaml.Node, entityType string) []*yaml.Node {
	entitySelectors, ok := EntityPointers[entityType]
	if !ok {
		// programmer error, specified a bad name; let's be helpful and list the valid ones...
		panic(fmt.Sprintf("invalid entity type: '%s', valid ones are: %s", entityType, getAvailableEntities()))
	}

	allEntities := make([]*yaml.Node, 0)
	if deckfile == nil {
		return allEntities
	}

	for _, entitySelector := range entitySelectors {
		query, err := jsonpath.NewPath(entitySelector)
		if err != nil {
			panic("failed compiling " + entityType + " selector")
		}

		entities := query.Query(deckfile)
		allEntities = append(allEntities, entities...)
	}

	return allEntities
}
