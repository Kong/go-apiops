package namespace

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kong/go-apiops/deckformat"
	"github.com/kong/go-apiops/logbasics"
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
	if err := yamlbasics.CheckType(route, yamlbasics.TypeObject); err != nil {
		logbasics.Info("ignoring route: " + err.Error())
		return false
	}

	pathsKeyIdx := yamlbasics.FindFieldKeyIndex(route, "paths")
	if pathsKeyIdx == -1 {
		// a prefix was specified, but there is no "paths" array, so add one
		pathsKeyIdx = len(route.Content)
		yamlbasics.SetFieldValue(route, "paths", yamlbasics.NewArray())
	}

	pathsArrayNode := route.Content[pathsKeyIdx+1]
	if err := yamlbasics.CheckType(pathsArrayNode, yamlbasics.TypeArray); err != nil {
		logbasics.Info("ignoring route, bad paths property: " + err.Error())
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
func Apply(deckfile *yaml.Node, selectors yamlbasics.SelectorSet, namespace string, allowEmptySelection bool) error {
	if deckfile == nil {
		panic("expected 'deckfile' to be non-nil")
	}
	err := CheckNamespace(namespace)
	if err != nil {
		return err
	}

	allRoutes := getAllRoutes(deckfile)
	var targetRoutes yamlbasics.NodeSet
	if selectors.IsEmpty() {
		// no selectors, apply to all routes
		targetRoutes = make(yamlbasics.NodeSet, len(allRoutes))
		copy(targetRoutes, allRoutes)
	} else {
		targetRoutes, err = selectors.Find(deckfile)
		if err != nil {
			return err
		}
	}

	var remainder yamlbasics.NodeSet
	targetRoutes, remainder = allRoutes.Intersection(targetRoutes) // check for non-routes
	if len(remainder) != 0 {
		return fmt.Errorf("the selectors returned non-route entities; %d", len(remainder))
	}
	if len(targetRoutes) == 0 {
		if allowEmptySelection {
			logbasics.Info("no routes matched the selectors, nothing to do")
			return nil
		}
		return errors.New("no routes matched the selectors. Check command help to suppress this error")
	}

	routesNoStripping := allRoutes.Subtract(targetRoutes) // everything not matched by the selectors
	routesNeedStripping := make(yamlbasics.NodeSet, 0)
	for _, route := range targetRoutes {
		if UpdateRoute(route, namespace) {
			routesNeedStripping = append(routesNeedStripping, route)
		} else {
			routesNoStripping = append(routesNoStripping, route)
		}
	}

	logbasics.Info("updating routes", "total", len(allRoutes), "selected", len(targetRoutes))
	InjectNamespaceStripping(deckfile, namespace, routesNeedStripping, routesNoStripping)

	return nil
}

// getAllServices returns all service nodes. Non-object entries will be skipped.
// The result will never be nil, but can be an empty array. The deckfile may be nil.
func getAllServices(deckfile *yaml.Node) yamlbasics.NodeSet {
	list := deckformat.GetEntities(deckfile, "services")
	cleanedList := make(yamlbasics.NodeSet, 0)
	for _, service := range list {
		if err := yamlbasics.CheckType(service, yamlbasics.TypeObject); err != nil {
			logbasics.Info("ignoring service: " + err.Error())
			continue
		}
		cleanedList = append(cleanedList, service)
	}
	return cleanedList
}

// getAllRoutes returns all route nodes. Non-object entries will be skipped.
// The result will never be nil, but can be an empty array. The deckfile may be nil.
func getAllRoutes(deckfile *yaml.Node) yamlbasics.NodeSet {
	list := deckformat.GetEntities(deckfile, "routes")
	cleanedList := make(yamlbasics.NodeSet, 0)
	for _, route := range list {
		if err := yamlbasics.CheckType(route, yamlbasics.TypeObject); err != nil {
			logbasics.Info("ignoring route: " + err.Error())
			continue
		}
		cleanedList = append(cleanedList, route)
	}
	return cleanedList
}

// findServiceByRoute returns the service node that matches the route.
// The result will be nil, if no service matches the route.
func findServiceByRoute(route *yaml.Node, allServices yamlbasics.NodeSet) *yaml.Node {
	if err := yamlbasics.CheckType(route, yamlbasics.TypeObject); err != nil {
		panic("route: " + err.Error())
	}

	// walk the services, to find the route as nested entity
	for _, service := range allServices {
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
		idIdx := yamlbasics.FindFieldKeyIndex(service, "id")
		if idIdx != -1 && service.Content[idIdx+1].Value == serviceRef {
			return service // Found it by ID!
		}
	}

	// find by name
	for _, service := range allServices {
		nameIdx := yamlbasics.FindFieldKeyIndex(service, "name")
		if nameIdx != -1 && service.Content[nameIdx+1].Value == serviceRef {
			return service // Found it by name!
		}
	}

	// service specified in the route object, but not found, so it is an inconsistent
	// file. Report as not found
	return nil
}

// InjectNamespaceStripping injects a namespace stripper into the deckfile.
// The namespace stripper will remove the namespace from the path, if it matches.
// updated+unchanged must together be ALL routes in the file!
func InjectNamespaceStripping(deckfile *yaml.Node, namespace string,
	routesNeedStripping yamlbasics.NodeSet, routesNoStripping yamlbasics.NodeSet,
) {
	serviceToUpdate := make(map[*yaml.Node]yamlbasics.NodeSet) // service -> routes
	routesToUpdate := make(yamlbasics.NodeSet, 0)
	allServices := getAllServices(deckfile)

	for _, route := range routesNeedStripping {
		if service := findServiceByRoute(route, allServices); service != nil {
			serviceToUpdate[service] = append(serviceToUpdate[service], route)
		} else {
			// not attached to a service, so must get its own plugin
			routesToUpdate = append(routesToUpdate, route)
		}
	}

	for _, route := range routesNoStripping {
		if service := findServiceByRoute(route, allServices); service != nil {
			if _, ok := serviceToUpdate[service]; ok {
				// this service also has routes to strip, so all the routes must individually get the plugin.
				// move the routes, and remove the service from the "updatable" services map
				routesToUpdate = append(routesToUpdate, serviceToUpdate[service]...)
				delete(serviceToUpdate, service)
			}
		}
	}

	logbasics.Info("entities to update with stripping", "routes", len(routesToUpdate), "services", len(serviceToUpdate))

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

	namespaceSuffix := namespace
	if !strings.HasSuffix(namespaceSuffix, "/") {
		namespaceSuffix = namespaceSuffix + "/"
	}

	if servicePath != "" && strings.HasSuffix(servicePath, namespaceSuffix) {
		// if the namespace matches the "tail" of the 'service.path' property, we can strip
		// it there instead of injecting a plugin.
		pathNode.Value = strings.TrimSuffix(servicePath, namespaceSuffix) + "/"
	} else {
		// inject a plugin
		injectEntityNamespaceStripping(service, namespace)
	}
}
