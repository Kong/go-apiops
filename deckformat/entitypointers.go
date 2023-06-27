package deckformat

import "strings"

// EntityPointers is a map of entity names to an array of JSONpointers that can be used to find
// all of those entities in a deck file. For example; credentials typically can be under
// their own top-level key, or nested under a consumer.
// Additional sets are dynamically added (capitalized) for other common uses. e.g.
// "PluginOwners" is a list of all entities that can hold plugins.
var EntityPointers = map[string][]string{
	// list created from the deck source code, looking at: deck/types/*.go
	"acls": {
		"$.acls[*]",
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
