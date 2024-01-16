package namespace

import (
	"fmt"
	"strings"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/yamlbasics"
	"gopkg.in/yaml.v3"
)

// CheckNamespace validates the prefix namespace. Returns updated namespace. Must start with "/",
// and must have at least 1 character after the "/". If there is no trailing '/', then it will be added.
func CheckNamespace(prefix string) (string, error) {
	defaultErr := fmt.Errorf("invalid namespace; the namespace MUST start with '/', "+
		"and cannot be empty, got: '%s'", prefix)

	if !strings.HasPrefix(prefix, "/") {
		return "", defaultErr
	}

	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	if len(prefix) <= 2 {
		return "", defaultErr
	}

	return prefix, nil
}

// CheckPrefix validates the prefix argument. Returns updated prefix.
// Defaults to "", if not specified. Regexes, starting with "~", are not valid.
func CheckPrefix(prefix string) (string, error) {
	if prefix == "" {
		return "", nil
	}

	if !strings.HasPrefix(prefix, "/") {
		return "", fmt.Errorf("invalid prefix; the prefix MUST start with '/', got: '%s'", prefix)
	}

	return prefix, nil
}

// UpdateSinglePathString updates a single path string with the namespace and returns it.
func UpdateSinglePathString(path string, prefix string, namespace string) string {
	strip := "/"
	if strings.HasPrefix(path, "~") {
		prefix = "~" + prefix
		namespace = "~" + namespace
		strip = "~" + strip
	}
	if prefix == "" {
		// plain path, no prefix to match
		return namespace + strings.TrimPrefix(path, strip) // prevent double slashes
	} else if strings.HasPrefix(path, prefix) {
		return namespace + strings.TrimPrefix(path, strip)
	}

	return path // unchanged
}

// UpdateRoute returns true if the route needs stripping the namespace.
// prefix can be empty (matches all paths).
// namespace must start with a "/" and end with a "/" (a single "/" is NOT valid).
func UpdateRoute(route *yaml.Node, prefix string, namespace string) bool {
	if route.Kind != yaml.MappingNode {
		return false
	}

	pathsKeyIdx := yamlbasics.FindFieldKeyIndex(route, "paths")
	if pathsKeyIdx == -1 {
		// no "paths" property found. If a prefix was specified, we have a no-match, otherwise we can update
		// by adding a paths array, and specifying the prefix for Kong to match
		if prefix != "" {
			return false
		}

		// a prefix was specified, but there is no "paths" array, so add one
		pathsKeyIdx = len(route.Content)
		yamlbasics.SetFieldValue(route, "paths", yamlbasics.NewArray())
	}

	pathsArrayNode := route.Content[pathsKeyIdx+1]
	if pathsArrayNode.Kind != yaml.SequenceNode {
		return false
	}

	stripPath := true // the default value
	stripPathValueNode := yamlbasics.GetFieldValue(route, "strip_path")
	if stripPathValueNode != nil {
		stripPath = stripPathValueNode.Value == "true"
	}

	if len(pathsArrayNode.Content) == 0 {
		// empty array, add a single entry matching the namespace
		_ = yamlbasics.Append(pathsArrayNode, yamlbasics.NewString(namespace))
		return !stripPath
	}

	// we have a paths-array, now update them all
	updates := 0
	for _, pathNode := range pathsArrayNode.Content {
		updatedPath := UpdateSinglePathString(pathNode.Value, prefix, namespace)
		if updatedPath != pathNode.Value {
			pathNode.Value = updatedPath
			updates++
		}
	}

	return updates != 0 && !stripPath
}

// getAllRoutes returns all route nodes.
// The result will never be nil, but can be an empty array. The deckfile may be nil.
func getAllRoutes(deckfile *yaml.Node) []*yaml.Node {
	return deckformat.GetEntities(deckfile, "routes")
}

// Apply updates all route entities found within the file with the namespace.
func Apply(deckfile *yaml.Node, prefixToMatch string, namespace string) error {
	if deckfile == nil {
		panic("expected 'deckfile' to be non-nil")
	}
	namespace, err := CheckNamespace(namespace)
	if err != nil {
		return err
	}
	prefixToMatch, err = CheckPrefix(prefixToMatch)
	if err != nil {
		return err
	}

	routesNeedStripping := make([]*yaml.Node, 0)
	routesNoStripping := make([]*yaml.Node, 0)
	for _, route := range getAllRoutes(deckfile) {
		if route.Kind == yaml.MappingNode {
			if UpdateRoute(route, prefixToMatch, namespace) {
				routesNeedStripping = append(routesNeedStripping, route)
			} else {
				routesNoStripping = append(routesNoStripping, route)
			}
		}
	}

	InjectNamespaceStripping(deckfile, namespace, routesNeedStripping, routesNoStripping)

	return nil
}

// getAllServices returns all service nodes.
// The result will never be nil, but can be an empty array. The deckfile may be nil.
func getAllServices(deckfile *yaml.Node) []*yaml.Node {
	return deckformat.GetEntities(deckfile, "services")
}

// findServiceByRoute returns the service node that matches the route.
// The result will be nil, if no service matches the route.
func findServiceByRoute(route *yaml.Node, deckfile *yaml.Node) *yaml.Node {
	if route.Kind != yaml.MappingNode {
		return nil
	}
	allServices := getAllServices(deckfile)

	// walk the services, to find the route as nested entity
	for _, service := range allServices {
		if service.Kind != yaml.MappingNode {
			continue
		}
		routeIdx := yamlbasics.FindFieldKeyIndex(service, "routes")
		if routeIdx == -1 {
			continue
		}
		routes := service.Content[routeIdx+1]
		if routes.Kind != yaml.SequenceNode {
			continue
		}
		for _, routeNode := range routes.Content {
			if routeNode == route {
				return service // Found it!
			}
		}
	}

	// Find the service by id or name
	serviceIdx := yamlbasics.FindFieldKeyIndex(route, "service")
	if serviceIdx == -1 {
		return nil // a service-less route
	}
	serviceRef := route.Content[serviceIdx+1].Value

	// find by ID
	for _, service := range allServices {
		if service.Kind != yaml.MappingNode {
			continue
		}
		idIdx := yamlbasics.FindFieldKeyIndex(service, "id")
		if idIdx != -1 && service.Content[idIdx+1].Value == serviceRef {
			return service // Found it by ID!
		}
	}

	// find by name
	for _, service := range allServices {
		if service.Kind != yaml.MappingNode {
			continue
		}
		nameIdx := yamlbasics.FindFieldKeyIndex(service, "name")
		if nameIdx != -1 && service.Content[nameIdx+1].Value == serviceRef {
			return service // Found it by name!
		}
	}

	// service specified in the route object, but not found, so it is an inconsistent
	// file. Report not as not found
	return nil
}

// routesNeedStripping, routesNoStripping

// InjectNamespaceStripping injects a namespace stripper into the deckfile.
// The namespace stripper will remove the namespace from the path, if it matches.
// updated+unchanged must together be ALL routes in the file!
func InjectNamespaceStripping(deckfile *yaml.Node, namespace string,
	routesNeedStripping []*yaml.Node, routesNoStripping []*yaml.Node,
) {
	serviceToUpdate := make(map[*yaml.Node][]*yaml.Node) // service -> routes
	routesToUpdate := make([]*yaml.Node, 0)

	for _, route := range routesNeedStripping {
		if service := findServiceByRoute(route, deckfile); service != nil {
			serviceToUpdate[service] = append(serviceToUpdate[service], route)
		} else {
			// not attached to a service, so must get its own plugin
			routesToUpdate = append(routesToUpdate, route)
		}
	}

	for _, route := range routesNoStripping {
		if service := findServiceByRoute(route, deckfile); service != nil {
			if _, ok := serviceToUpdate[service]; ok {
				// this service also has routes to strip, so all the routes must individually get the plugin.
				// move the routes, and remove the service from the "updatable" services map
				routesToUpdate = append(routesToUpdate, serviceToUpdate[service]...)
				delete(serviceToUpdate, service)
			}
		}
	}

	// inject stripping logic into the entities
	for _, route := range routesToUpdate {
		injectRouteNamespaceStripping(route, namespace)
	}

	for service := range serviceToUpdate {
		injectServiceNamespaceStripping(service, namespace)
	}
}

// injectEntityNamespaceStripping adds a namespace stripper to the entity.
func injectEntityNamespaceStripping(entity *yaml.Node, namespace string) {
	pluginconfig := `{
		"name": "pre-function",
		"config": {
			"access": [
				"local u,s,e=ngx.var.upstream_uri s,e=u:find('` + namespace + `',1,true)` +
		`ngx.var.upstream_uri=u:sub(1,s)..u:sub(e+1,-1)"
			]
		}
	}`

	pluginsIdx := yamlbasics.FindFieldKeyIndex(entity, "plugins")
	if pluginsIdx == -1 {
		// no plugins array, add a new array
		pluginsIdx = len(entity.Content)
		yamlbasics.SetFieldValue(entity, "plugins", yamlbasics.NewArray())
	}

	pluginsArrayNode := entity.Content[pluginsIdx+1]

	if pluginsArrayNode.Kind == yaml.SequenceNode {
		// add the plugin to the array
		plugin, _ := yamlbasics.FromObject(filebasics.MustDeserialize([]byte(pluginconfig)))
		_ = yamlbasics.Append(pluginsArrayNode, plugin)
	}
}

// injectRouteNamespaceStripping adds a namespace stripper to the route.
func injectRouteNamespaceStripping(route *yaml.Node, namespace string) {
	injectEntityNamespaceStripping(route, namespace)
}

// injectServiceNamespaceStripping adds a namespace stripper to the service.
func injectServiceNamespaceStripping(service *yaml.Node, namespace string) {
	servicePath := ""
	pathNode := yamlbasics.GetFieldValue(service, "path")
	if pathNode != nil && pathNode.Kind == yaml.ScalarNode && pathNode.Tag == "!!str" {
		servicePath = pathNode.Value
		if !strings.HasSuffix(servicePath, "/") {
			servicePath = servicePath + "/"
		}
		if servicePath == "/" {
			servicePath = ""
		}
	}

	if servicePath != "" && strings.HasSuffix(servicePath, namespace) {
		// if the namespace matches the "tail" of the 'service.path' property, we can strip
		// it there instead of injecting a plugin.
		pathNode.Value = strings.TrimSuffix(servicePath, namespace) + "/"
	} else {
		// inject a plugin
		injectEntityNamespaceStripping(service, namespace)
	}
}
