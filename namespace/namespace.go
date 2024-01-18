package namespace

import (
	"fmt"
	"strings"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/yamlbasics"
	"gopkg.in/yaml.v3"
)

// CheckNamespace validates the prefix namespace. Returns updated namespace. Must start with "/",
// and must have at least 1 character after the "/".
func CheckNamespace(ns string) error {
	defaultErr := fmt.Errorf("invalid namespace; the namespace MUST start with '/', "+
		"and cannot be empty, got: '%s'", ns)

	if !strings.HasPrefix(ns, "/") {
		return defaultErr
	}

	if len(ns) == 1 {
		return defaultErr
	}

	if strings.HasPrefix(ns, "//") {
		return defaultErr
	}

	return nil
}

// UpdateSinglePathString updates a single path string with the namespace and returns it.
func UpdateSinglePathString(path string, namespace string) string {
	strip := "/"
	if strings.HasPrefix(path, "~") {
		namespace = "~" + namespace
		strip = "~" + strip
	}
	extraSlash := ""
	if !strings.HasSuffix(namespace, "/") {
		// normal path we need to add a slash, except if the path is an empty one; just "/"
		if path != "/" && path != "~/" && path != "~/$" {
			extraSlash = "/"
		}
	}
	return namespace + extraSlash + strings.TrimPrefix(path, strip) // prevent double slashes
}

// UpdateRoute returns true if the route needs stripping the namespace.
// namespace must start with a "/" and end with a "/" (a single "/" is NOT valid).
func UpdateRoute(route *yaml.Node, namespace string) bool {
	if route.Kind != yaml.MappingNode {
		return false
	}

	pathsKeyIdx := yamlbasics.FindFieldKeyIndex(route, "paths")
	if pathsKeyIdx == -1 {
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
		updatedPath := UpdateSinglePathString(pathNode.Value, namespace)
		if updatedPath != pathNode.Value {
			pathNode.Value = updatedPath
			updates++
		}
	}

	return updates != 0 && !stripPath
}

// Apply updates route entities with the namespace. The selectors should select the routes
// to update. If the selectors are empty, then all routes will be updated.
func Apply(deckfile *yaml.Node, selectors yamlbasics.SelectorSet, namespace string) error {
	if deckfile == nil {
		panic("expected 'deckfile' to be non-nil")
	}
	err := CheckNamespace(namespace)
	if err != nil {
		return err
	}

	allRoutes := deckformat.GetEntities(deckfile, "routes")
	var targetRoutes []*yaml.Node
	if selectors.IsEmpty() {
		// no selectors, apply to all routes
		targetRoutes = make([]*yaml.Node, len(allRoutes))
		copy(targetRoutes, allRoutes)
	} else {
		targetRoutes, err = selectors.Find(deckfile)
		if err != nil {
			return err
		}
	}
	targetRoutes = yamlbasics.Intersection(allRoutes, targetRoutes) // ignore everything not a route
	if len(targetRoutes) == 0 {
		return nil // nothing to do
	}

	routesNoStripping := yamlbasics.SubtractSet(allRoutes, targetRoutes) // everything not matched by the selectors
	routesNeedStripping := make([]*yaml.Node, 0)
	for _, route := range targetRoutes {
		if UpdateRoute(route, namespace) {
			routesNeedStripping = append(routesNeedStripping, route)
		} else {
			routesNoStripping = append(routesNoStripping, route)
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

// GetLuaStripFunction returns the Lua function that strips the namespace from the upstream_uri.
func GetLuaStripFunction(ns string) string {
	// code is optimized to be SHORT, not readable. Upon updates only the "namespace" will change
	// it will be on the first line, so diffs in a gitops pipeline remain easy to grok.
	return `local ns='` + ns + `' -- this strips the '` + ns + `' namespace from the path
local nst=ns:sub(-1,-1)=='/' and ns or (ns..'/')
local function sn(u)
	local s,e=u:find(nst,1,true)
	if s then
		return u:sub(1,s)..u:sub(e+1,-1)
	end
	if u:sub(-#ns,-1)==ns then
		u=u:sub(1,-#ns-1)
		if u=='' then u='/' end
	end
	return u
end
ngx.var.upstream_uri=sn(ngx.var.upstream_uri)`
}

// GetPreFunctionPlugin returns a plugin that strips the namespace from the upstream_uri.
func GetPreFunctionPlugin(namespace string) *yaml.Node {
	plugin := map[string]interface{}{
		"name": "pre-function",
		"config": map[string]interface{}{
			"access": []string{
				GetLuaStripFunction(namespace),
			},
		},
	}
	pluginNode, err := yamlbasics.FromObject(plugin)
	if err != nil {
		panic(err)
	}
	return pluginNode
}

// injectEntityNamespaceStripping adds a namespace stripper to the entity.
func injectEntityNamespaceStripping(entity *yaml.Node, namespace string) {
	pluginsIdx := yamlbasics.FindFieldKeyIndex(entity, "plugins")
	if pluginsIdx == -1 {
		// no plugins array, add a new array
		pluginsIdx = len(entity.Content)
		yamlbasics.SetFieldValue(entity, "plugins", yamlbasics.NewArray())
	}

	pluginsArrayNode := entity.Content[pluginsIdx+1]

	if pluginsArrayNode.Kind == yaml.SequenceNode {
		// add the plugin to the array
		_ = yamlbasics.Append(pluginsArrayNode, GetPreFunctionPlugin(namespace))
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
