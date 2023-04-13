package namespace

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

// Apply updates all route entities found within the yamlNode with the namespace.
func Apply(data *yaml.Node, prefix string, namespaceStr string) error {
	if !(strings.HasPrefix(prefix, "/") || strings.HasPrefix(prefix, "~/")) {
		panic(fmt.Sprintf("invalid prefix; the prefix MUST start with '/', got: '%s'", prefix))
	}

	if !strings.HasPrefix(namespaceStr, "/") {
		panic(fmt.Sprintf("invalid namespace; the namespace MUST start with '/', got: '%s'", prefix))
	}

	query, err := yamlpath.NewPath("$..routes[*]")
	if err != nil {
		panic("failed compiling route selector")
	}

	allRoutes, err := query.Find(data)
	if err != nil {
		return errors.New("failed to collect routes from the input data")
	}

	for _, route := range allRoutes {
		if route.Kind == yaml.MappingNode {
			UpdateRoute(route, prefix, namespaceStr)
		}
	}

	return nil
}

// UpdateRoute update a single route object with the namespace.
func UpdateRoute(route *yaml.Node, prefix string, ns string) {
	if route.Kind != yaml.MappingNode {
		panic("expected a MappingNode")
	}

	var updated bool
	for i := 0; i < len(route.Content); i += 2 {
		key := route.Content[i]
		if key.Value == "paths" {
			// found the 'paths' property
			value := route.Content[i+1]
			if value.Kind == yaml.SequenceNode {
				// it's an array, as expected, go update it
				updated = updatePathsArray(value, prefix, ns)
				break
			}
		}
	}

	if !updated {
		return // nothing changed, so we're done
	}

	// set strip_path & strip_prefix properties
	var stripPathNode *yaml.Node
	var stripPrefixNode *yaml.Node
	for i := 0; i < len(route.Content); i += 2 {
		key := route.Content[i]
		switch key.Value {
		case "strip_path":
			stripPathNode = route.Content[i+1]
		case "strip_prefix":
			stripPrefixNode = route.Content[i+1]
		}
	}

	if stripPathNode != nil {
		// a 'strip_path' property is present
		if stripPathNode.Value == "true" && stripPrefixNode == nil {
			// nothing to do. We were already stripping the entire path, and we're not changing that
			return
		}
		stripPathNode.Value = "true"
	} else {
		// add the 'strip_path' property
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.DoubleQuotedStyle,
			Tag:   "!!str",
			Value: "strip_path",
		}
		valueNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!bool",
			Value: "true",
		}
		stripPathNode = &valueNode
		route.Content = append(route.Content, &keyNode, stripPathNode)
	}

	if stripPrefixNode != nil {
		// we're already stripping a prefix, we just need to strip some more
		stripPrefixNode.Value = ns + stripPrefixNode.Value
	} else {
		// add the 'strip_prefix' property
		keyNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.DoubleQuotedStyle,
			Tag:   "!!str",
			Value: "strip_prefix",
		}
		valueNode := yaml.Node{
			Kind:  yaml.ScalarNode,
			Style: yaml.DoubleQuotedStyle,
			Tag:   "!!str",
			Value: ns,
		}
		stripPrefixNode = &valueNode
		route.Content = append(route.Content, &keyNode, stripPrefixNode)
	}
}

// updatePathsArray updates a paths array of a route-entity. Returns true if the
// route was updated.
func updatePathsArray(paths *yaml.Node, prefix string, ns string) bool {
	if paths.Kind != yaml.SequenceNode {
		panic("expected a SequenceNode")
	}

	for _, path := range paths.Content {
		if path.Kind != yaml.ScalarNode {
			return false // only dealing with scalar values; path strings
		}

		if !(strings.HasPrefix(path.Value, prefix) || strings.HasPrefix(path.Value, "~"+prefix)) {
			return false // prefix has no match, but all paths must match...
		}
	}

	// all path enties patch, so now update them
	for _, path := range paths.Content {
		if strings.HasPrefix(path.Value, "~") {
			// path is a regex, so insert prefix after the "~"
			path.Value = strings.Replace(path.Value, "~", "~"+ns, 1)
		} else {
			path.Value = ns + path.Value
		}
	}
	return true
}
