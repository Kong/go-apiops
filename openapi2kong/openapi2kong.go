package openapi2kong

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-slugify"
)

const (
	formatVersionKey   = "_format_version"
	formatVersionValue = "3.0"
)

// O2KOptions defines the options for an O2K conversion operation
type O2kOptions struct {
	Tags          []string  // Array of tags to mark all generated entities with, taken from 'x-kong-tags' if omitted.
	DocName       string    // Base document name, will be taken from x-kong-name, or info.title (for UUID generation!)
	UUIDNamespace uuid.UUID // Namespace for UUID generation, defaults to DNS namespace for UUID v5
}

// setDefaults sets the defaults for the OpenAPI2Kong operation.
func (opts *O2kOptions) setDefaults() {
	var emptyUUID uuid.UUID

	if bytes.Equal(emptyUUID[:], opts.UUIDNamespace[:]) {
		opts.UUIDNamespace = uuid.NameSpaceDNS
	}
}

// Slugify converts a name to a valid Kong name by removing and replacing unallowed characters
// and sanitizing non-latin characters. Multiple inputs will be concatenated using '_'.
func Slugify(name ...string) string {
	slugify.ToLower = false
	slugify.Separator = "_"
	for i, elem := range name {
		name[i] = slugify.Slugify(elem)
	}

	return strings.Join(name, "-")
}

// sanitizeRegexCapture will remove illegal characters from the path-variable name.
// The returned name will be valid for PCRE regex captures; Alphanumeric + '_', starting
// with [a-zA-Z].
func sanitizeRegexCapture(varName string) string {
	varName = slugify.Slugify(varName)
	varName = strings.ReplaceAll(varName, "-", "_")
	if strings.HasPrefix(varName, "_") {
		varName = "a" + varName
	}
	return varName
}

// getKongTags returns the provided tags or if nil, then the `x-kong-tags` property,
// validated to be a string array. If there is no error, then there will always be
// an array returned for safe access later in the process.
func getKongTags(doc *openapi3.T, tagsProvided []string) ([]string, error) {
	if tagsProvided != nil {
		// the provided tags take precedence, return them
		return tagsProvided, nil
	}

	if doc.ExtensionProps.Extensions == nil || doc.ExtensionProps.Extensions["x-kong-tags"] == nil {
		// there is no extension, so return an empty array
		return make([]string, 0), nil
	}

	var tagsValue interface{}
	err := json.Unmarshal(doc.ExtensionProps.Extensions["x-kong-tags"].(json.RawMessage), &tagsValue)
	if err != nil {
		return nil, fmt.Errorf("expected 'x-kong-tags' to be an array of strings: %w", err)
	}
	var tagsArray []interface{}
	switch tags := tagsValue.(type) {
	case []interface{}:
		// got a proper array
		tagsArray = tags
	default:
		return nil, fmt.Errorf("expected 'x-kong-tags' to be an array of strings")
	}

	resultArray := make([]string, len(tagsArray))
	for i := 0; i < len(tagsArray); i++ {
		switch tag := tagsArray[i].(type) {
		case string:
			resultArray[i] = tag
		default:
			return nil, fmt.Errorf("expected 'x-kong-tags' to be an array of strings")
		}
	}
	return resultArray, nil
}

// getKongName returns the `x-kong-name` property, validated to be a string
func getKongName(props openapi3.ExtensionProps) (string, error) {
	if props.Extensions != nil && props.Extensions["x-kong-name"] != nil {
		var name string
		err := json.Unmarshal(props.Extensions["x-kong-name"].(json.RawMessage), &name)
		if err != nil {
			return "", fmt.Errorf("expected 'x-kong-name' to be a string: %w", err)
		}
		return name, nil
	}
	return "", nil
}

func dereferenceJSONObject(
	value map[string]interface{},
	components *map[string]interface{},
) (map[string]interface{}, error) {
	var pointer string

	switch value["$ref"].(type) {
	case nil: // it is not a reference, so return the object
		return value, nil

	case string: // it is a json pointer
		pointer = value["$ref"].(string)
		if !strings.HasPrefix(pointer, "#/components/x-kong/") {
			return nil, fmt.Errorf("all 'x-kong-...' references must be at '#/components/x-kong/...'")
		}

	default: // bad pointer
		return nil, fmt.Errorf("expected '$ref' pointer to be a string")
	}

	// walk the tree to find the reference
	segments := strings.Split(pointer, "/")
	path := "#/components/x-kong"
	result := components

	for i := 3; i < len(segments); i++ {
		segment := segments[i]
		path = path + "/" + segment

		switch (*result)[segment].(type) {
		case nil:
			return nil, fmt.Errorf("reference '%s' not found", pointer)
		case map[string]interface{}:
			target := (*result)[segment].(map[string]interface{})
			result = &target
		default:
			return nil, fmt.Errorf("expected '%s' to be a JSON object", path)
		}
	}

	return *result, nil
}

// getXKongObject returns specified 'key' from the extension properties if available.
// returns nil if it wasn't found, an error if it wasn't an object or couldn't be
// dereferenced. The returned object will be json encoded again.
func getXKongObject(props openapi3.ExtensionProps, key string, components *map[string]interface{}) ([]byte, error) {
	if props.Extensions != nil && props.Extensions[key] != nil {
		var jsonBlob interface{}
		_ = json.Unmarshal(props.Extensions[key].(json.RawMessage), &jsonBlob)
		jsonObject, err := jsonbasics.ToObject(jsonBlob)
		if err != nil {
			return nil, fmt.Errorf("expected '%s' to be a JSON object", key)
		}

		object, err := dereferenceJSONObject(jsonObject, components)
		if err != nil {
			return nil, err
		}
		return json.Marshal(object)
	}
	return nil, nil
}

// getXKongComponents will return a map of the '/components/x-kong/' object. If
// the extension is not there it will return an empty map. If the entry is not a
// Json object, it will return an error.
func getXKongComponents(doc *openapi3.T) (*map[string]interface{}, error) {
	var components map[string]interface{}
	switch prop := doc.Components.ExtensionProps.Extensions["x-kong"].(type) {
	case nil:
		// not available, create empty map to do safe lookups down the line
		components = make(map[string]interface{})

	default:
		// we got some json blob
		var xKong interface{}
		_ = json.Unmarshal(prop.(json.RawMessage), &xKong)

		switch val := xKong.(type) {
		case map[string]interface{}:
			components = val

		default:
			return nil, fmt.Errorf("expected '/components/x-kong' to be a JSON object")
		}
	}

	return &components, nil
}

// getServiceDefaults returns a JSON string containing the defaults
func getServiceDefaults(props openapi3.ExtensionProps, components *map[string]interface{}) ([]byte, error) {
	return getXKongObject(props, "x-kong-service-defaults", components)
}

// getUpstreamDefaults returns a JSON string containing the defaults
func getUpstreamDefaults(props openapi3.ExtensionProps, components *map[string]interface{}) ([]byte, error) {
	return getXKongObject(props, "x-kong-upstream-defaults", components)
}

// getRouteDefaults returns a JSON string containing the defaults
func getRouteDefaults(props openapi3.ExtensionProps, components *map[string]interface{}) ([]byte, error) {
	return getXKongObject(props, "x-kong-route-defaults", components)
}

// create plugin id
func createPluginID(uuidNamespace uuid.UUID, baseName string, config map[string]interface{}) string {
	pluginName := config["name"].(string) // safe because it was previously parsed

	return uuid.NewSHA1(uuidNamespace, []byte(baseName+".plugin."+pluginName)).String()
}

// getPluginsList returns a list of plugins retrieved from the extension properties
// (the 'x-kong-plugin<pluginname>' extensions). Applied on top of the optional
// pluginsToInclude list. The result will be sorted by plugin name.
func getPluginsList(
	props openapi3.ExtensionProps,
	pluginsToInclude *[]*map[string]interface{},
	uuidNamespace uuid.UUID,
	baseName string,
	components *map[string]interface{},
	tags []string,
) (*[]*map[string]interface{}, error) {
	plugins := make(map[string]*map[string]interface{})

	// copy inherited list of plugins
	if pluginsToInclude != nil {
		for _, config := range *pluginsToInclude {
			pluginName := (*config)["name"].(string) // safe because it was previously parsed
			configCopy := jsonbasics.DeepCopyObject(*config)

			// generate a new ID, for a new plugin, based on new basename
			configCopy["id"] = createPluginID(uuidNamespace, baseName, configCopy)

			configCopy["tags"] = tags

			plugins[pluginName] = &configCopy
		}
	}

	if props.Extensions != nil {
		// there are extensions, go check if there are plugins
		for extensionName := range props.Extensions {
			if strings.HasPrefix(extensionName, "x-kong-plugin-") {
				pluginName := strings.TrimPrefix(extensionName, "x-kong-plugin-")

				jsonstr, err := getXKongObject(props, extensionName, components)
				if err != nil {
					return nil, err
				}

				var pluginConfig map[string]interface{}
				err = json.Unmarshal(jsonstr, &pluginConfig)
				if err != nil {
					return nil, fmt.Errorf(fmt.Sprintf("failed to parse JSON object for '%s': %%w", extensionName), err)
				}

				pluginConfig["name"] = pluginName
				pluginConfig["id"] = createPluginID(uuidNamespace, baseName, pluginConfig)
				pluginConfig["tags"] = tags

				// foreign keys to service+route are not allowed (consumer is allowed)
				delete(pluginConfig, "service")
				delete(pluginConfig, "route")

				plugins[pluginName] = &pluginConfig
			}
		}
	}

	// the list is complete, sort to be deterministic in the output
	sortedNames := make([]string, len(plugins))
	i := 0
	for pluginName := range plugins {
		sortedNames[i] = pluginName
		i++
	}
	sort.Strings(sortedNames)

	sorted := make([]*map[string]interface{}, len(plugins))
	for i, pluginName := range sortedNames {
		sorted[i] = plugins[pluginName]
	}
	return &sorted, nil
}

// getValidatorPlugin will remove the request validator config from the plugin list
// and return it as a JSON string, along with the updated plugin list. If there
// is none, the returned config will be the currentConfig.
func getValidatorPlugin(list *[]*map[string]interface{}, currentConfig []byte) ([]byte, *[]*map[string]interface{}) {
	for i, plugin := range *list {
		pluginName := (*plugin)["name"].(string) // safe because it was previously parsed
		if pluginName == "request-validator" {
			// found it. Serialize to JSON and remove from list
			jsonConfig, _ := json.Marshal(plugin)
			l := append((*list)[:i], (*list)[i+1:]...)
			return jsonConfig, &l
		}
	}

	// no validator config found, so current config remains valid
	return currentConfig, list
}

// insertPlugin will insert a plugin in the list array, in a sorted manner.
// List must already be sorted by plugin-name.
func insertPlugin(list *[]*map[string]interface{}, newPlugin *map[string]interface{}) *[]*map[string]interface{} {
	if newPlugin == nil {
		return list
	}

	newPluginName := (*newPlugin)["name"].(string) // safe because it was previously parsed
	var l []*map[string]interface{}

	for _, plugin := range *list {
		pluginName := (*plugin)["name"].(string) // safe because it was previously parsed
		if pluginName == newPluginName {
			l = append(l, newPlugin)
			newPlugin = nil
		} else {
			if pluginName > newPluginName && newPlugin != nil {
				l = append(l, newPlugin)
				newPlugin = nil
			}
			l = append(l, plugin)
		}
	}

	if newPlugin != nil {
		// it's the last one, append it
		l = append(l, newPlugin)
	}

	return &l
}

// getForeignKeyPlugins checks the pluginList for plugins that also have a foreign key
// for a consumer, and moves them to the docPlugins array. Returns update docPlugins and pluginList.
func getForeignKeyPlugins(
	docPlugins *[]*map[string]interface{},
	pluginList *[]*map[string]interface{},
	foreignKey string, foreignValue string,
) (*[]*map[string]interface{}, *[]*map[string]interface{}) {
	var genericPlugins []*map[string]interface{}
	if docPlugins == nil {
		genericPlugins = make([]*map[string]interface{}, 0)
	} else {
		genericPlugins = *docPlugins
	}

	var entityPlugins []*map[string]interface{}
	if pluginList == nil {
		entityPlugins = make([]*map[string]interface{}, 0)
	} else {
		entityPlugins = *pluginList
	}

	newPluginList := make([]*map[string]interface{}, 0)
	for _, plugin := range entityPlugins {
		if (*plugin)["consumer"] == nil {
			// single key, so leave it, just append to outgoing slice
			newPluginList = append(newPluginList, plugin)
		} else {
			// multiple foreign keys, so this one needs to move over
			(*plugin)[foreignKey] = foreignValue // set foreign ref to either service or route
			genericPlugins = append(genericPlugins, plugin)
		}
	}
	return &genericPlugins, &newPluginList
}

// MustConvert is the same as Convert, but will panic if an error is returned.
func MustConvert(content []byte, opts O2kOptions) map[string]interface{} {
	result, err := Convert(content, opts)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

// Convert converts an OpenAPI spec to a Kong declarative file.
func Convert(content []byte, opts O2kOptions) (map[string]interface{}, error) {
	opts.setDefaults()
	logbasics.Debug("received OpenAPI2Kong options", "options", opts)

	// set up output document
	result := make(map[string]interface{})
	result[formatVersionKey] = formatVersionValue
	services := make([]interface{}, 0)
	upstreams := make([]interface{}, 0)

	var (
		err            error
		doc            *openapi3.T             // the OAS3 document we're operating on
		kongComponents *map[string]interface{} // contents of OAS key `/components/x-kong/`
		kongTags       []string                // tags to attach to Kong entities

		docBaseName         string                     // the slugified basename for the document
		docServers          *openapi3.Servers          // servers block on document level
		docServiceDefaults  []byte                     // JSON string representation of service-defaults on document level
		docService          map[string]interface{}     // service entity in use on document level
		docUpstreamDefaults []byte                     // JSON string representation of upstream-defaults on document level
		docUpstream         map[string]interface{}     // upstream entity in use on document level
		docRouteDefaults    []byte                     // JSON string representation of route-defaults on document level
		docPluginList       *[]*map[string]interface{} // array of plugin configs, sorted by plugin name
		docValidatorConfig  []byte                     // JSON string representation of validator config to generate
		foreignKeyPlugins   *[]*map[string]interface{} // top-level array of plugin configs, sorted by plugin name+id

		pathBaseName         string                     // the slugified basename for the path
		pathServers          *openapi3.Servers          // servers block on current path level
		pathServiceDefaults  []byte                     // JSON string representation of service-defaults on path level
		pathService          map[string]interface{}     // service entity in use on path level
		pathUpstreamDefaults []byte                     // JSON string representation of upstream-defaults on path level
		pathUpstream         map[string]interface{}     // upstream entity in use on path level
		pathRouteDefaults    []byte                     // JSON string representation of route-defaults on path level
		pathPluginList       *[]*map[string]interface{} // array of plugin configs, sorted by plugin name
		pathValidatorConfig  []byte                     // JSON string representation of validator config to generate

		operationBaseName         string                     // the slugified basename for the operation
		operationServers          *openapi3.Servers          // servers block on current operation level
		operationServiceDefaults  []byte                     // JSON string representation of service-defaults on ops level
		operationService          map[string]interface{}     // service entity in use on operation level
		operationUpstreamDefaults []byte                     // JSON string representation of upstream-defaults on ops level
		operationUpstream         map[string]interface{}     // upstream entity in use on operation level
		operationRouteDefaults    []byte                     // JSON string representation of route-defaults on ops level
		operationPluginList       *[]*map[string]interface{} // array of plugin configs, sorted by plugin name
		operationValidatorConfig  []byte                     // JSON string representation of validator config to generate
	)

	// Load and parse the OAS file
	loader := openapi3.NewLoader()
	doc, err = loader.LoadFromData(content)
	if err != nil {
		return nil, fmt.Errorf("error parsing OAS3 file: [%w]", err)
	}

	//
	//
	//  Handle OAS Document level
	//
	//

	// collect tags to use
	if kongTags, err = getKongTags(doc, opts.Tags); err != nil {
		return nil, err
	}
	logbasics.Info("tags after parsing x-kong-tags", "tags", kongTags)

	// set document level elements
	docServers = &doc.Servers // this one is always set, but can be empty

	// determine document name, precedence: specified -> x-kong-name -> Info.Title -> random
	docBaseName = opts.DocName
	if docBaseName == "" {
		logbasics.Debug("no document name specified, trying x-kong-name")
		if docBaseName, err = getKongName(doc.ExtensionProps); err != nil {
			return nil, err
		}
		if docBaseName == "" {
			logbasics.Debug("no x-kong-name specified, trying Info.Title")
			if doc.Info != nil && doc.Info.Title != "" {
				docBaseName = doc.Info.Title
			} else {
				logbasics.Info("no document name, x-kong-name, nor Info.Title specified, generating random name")
				id, err := uuid.NewRandom()
				if err != nil {
					return nil, fmt.Errorf("failed to generate UUID: %w", err)
				}
				docBaseName = id.String()
			}
		}
	}
	docBaseName = Slugify(docBaseName)
	logbasics.Info("document name (namespace for UUID generation)", "name", docBaseName)

	if kongComponents, err = getXKongComponents(doc); err != nil {
		return nil, err
	}

	// for defaults we keep strings, so deserializing them provides a copy right away
	if docServiceDefaults, err = getServiceDefaults(doc.ExtensionProps, kongComponents); err != nil {
		return nil, err
	}
	if docUpstreamDefaults, err = getUpstreamDefaults(doc.ExtensionProps, kongComponents); err != nil {
		return nil, err
	}
	if docRouteDefaults, err = getRouteDefaults(doc.ExtensionProps, kongComponents); err != nil {
		return nil, err
	}

	// create the top-level docService and (optional) docUpstream
	docService, docUpstream, err = CreateKongService(docBaseName, docServers, docServiceDefaults,
		docUpstreamDefaults, kongTags, opts.UUIDNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create service/upstream from document root: %w", err)
	}
	services = append(services, docService)
	if docUpstream != nil {
		upstreams = append(upstreams, docUpstream)
	}

	// attach plugins
	docPluginList, err = getPluginsList(doc.ExtensionProps, nil, opts.UUIDNamespace, docBaseName, kongComponents, kongTags)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugins list from document root: %w", err)
	}

	// Extract the request-validator config from the plugin list
	docValidatorConfig, docPluginList = getValidatorPlugin(docPluginList, docValidatorConfig)

	// move consumer bound plugins to doc level plugins list (multiple foreign keys)
	foreignKeyPlugins, docPluginList = getForeignKeyPlugins(
		foreignKeyPlugins, docPluginList, "service", docService["name"].(string))

	docService["plugins"] = docPluginList

	//
	//
	//  Handle OAS Path level
	//
	//

	// create a sorted array of paths, to be deterministic in our output order
	sortedPaths := make([]string, len(doc.Paths))
	i := 0
	for path := range doc.Paths {
		sortedPaths[i] = path
		i++
	}
	sort.Strings(sortedPaths)

	for _, path := range sortedPaths {
		logbasics.Info("processing path", "path", path)
		pathitem := doc.Paths[path]

		// determine path name, precedence: specified -> x-kong-name -> actual-path
		if pathBaseName, err = getKongName(pathitem.ExtensionProps); err != nil {
			return nil, err
		}
		if pathBaseName == "" {
			// create name from the path itself
			if path == "/" {
				// there is no path, so skip it
				pathBaseName = docBaseName
			} else {
				pathBaseName = Slugify(path)
				if strings.HasSuffix(path, "/") {
					// a common case is 2 paths, one with and one without a trailing "/" so to prevent
					// duplicate names being generated, we add a "~" suffix as a special case to cater
					// for different names. Better user solution is to use operation-id's.
					pathBaseName = pathBaseName + "~"
				}
				pathBaseName = docBaseName + "-" + pathBaseName
			}
		} else {
			// use x-kong-name
			pathBaseName = docBaseName + "-" + Slugify(pathBaseName)
		}
		logbasics.Debug("path name (namespace for UUID generation)", "name", pathBaseName)

		// Set up the defaults on the Path level
		newPathService := false
		if pathServiceDefaults, err = getServiceDefaults(pathitem.ExtensionProps, kongComponents); err != nil {
			return nil, err
		}
		if pathServiceDefaults == nil {
			pathServiceDefaults = docServiceDefaults
		} else {
			newPathService = true
		}

		newUpstream := false
		if pathUpstreamDefaults, err = getUpstreamDefaults(pathitem.ExtensionProps, kongComponents); err != nil {
			return nil, err
		}
		if pathUpstreamDefaults == nil {
			pathUpstreamDefaults = docUpstreamDefaults
		} else {
			newUpstream = true
			newPathService = true
		}

		if pathRouteDefaults, err = getRouteDefaults(pathitem.ExtensionProps, kongComponents); err != nil {
			return nil, err
		}
		if pathRouteDefaults == nil {
			pathRouteDefaults = docRouteDefaults
		}

		// if there is no path level servers block, use the document one
		pathServers = &pathitem.Servers
		if len(*pathServers) == 0 { // it's always set, so we ignore it if empty
			pathServers = docServers
		} else {
			newUpstream = true
			newPathService = true
		}

		// create a new service if we need to do so
		if newPathService {
			// create the path-level service and (optional) upstream
			logbasics.Debug("creating path-level service/upstream")
			pathService, pathUpstream, err = CreateKongService(
				pathBaseName,
				pathServers,
				pathServiceDefaults,
				pathUpstreamDefaults,
				kongTags,
				opts.UUIDNamespace)
			if err != nil {
				return nil, fmt.Errorf("failed to create service/updstream from path '%s': %w", path, err)
			}

			// collect path plugins, including the doc-level plugins since we have a new service entity
			pathPluginList, err = getPluginsList(pathitem.ExtensionProps, docPluginList,
				opts.UUIDNamespace, pathBaseName, kongComponents, kongTags)
			if err != nil {
				return nil, fmt.Errorf("failed to create plugins list from path item: %w", err)
			}

			// Extract the request-validator config from the plugin list
			pathValidatorConfig, pathPluginList = getValidatorPlugin(pathPluginList, docValidatorConfig)

			// move consumer bound plugins to doc level plugins list (multiple foreign keys)
			foreignKeyPlugins, pathPluginList = getForeignKeyPlugins(
				foreignKeyPlugins, pathPluginList, "service", pathService["name"].(string))

			pathService["plugins"] = pathPluginList

			services = append(services, pathService)
			if pathUpstream != nil {
				// we have a new upstream, but do we need it?
				if newUpstream {
					// we need it, so store and use it
					upstreams = append(upstreams, pathUpstream)
				} else {
					// we don't need it, so update service to point to 'upper' upstream
					pathService["host"] = docService["host"]
				}
			}
		} else {
			// no new path-level service entity required, so stick to the doc-level one
			pathService = docService

			// collect path plugins, only the path level, since we're on the doc-level service-entity
			pathPluginList, err = getPluginsList(pathitem.ExtensionProps, nil,
				opts.UUIDNamespace, pathBaseName, kongComponents, kongTags)
			if err != nil {
				return nil, fmt.Errorf("failed to create plugins list from path item: %w", err)
			}

			// Extract the request-validator config from the plugin list
			pathValidatorConfig, pathPluginList = getValidatorPlugin(pathPluginList, docValidatorConfig)
		}

		//
		//
		//  Handle OAS Operation level
		//
		//

		// create a sorted array of operations, to be deterministic in our output order
		operations := pathitem.Operations()
		sortedMethods := make([]string, len(operations))
		i := 0
		for method := range operations {
			sortedMethods[i] = method
			i++
		}
		sort.Strings(sortedMethods)

		// traverse all operations
		for _, method := range sortedMethods {
			operation := operations[method]
			logbasics.Info("processing operation", "method", method, "path", path, "id", operation.OperationID)

			var operationRoutes []interface{} // the routes array we need to add to

			// determine operation name, precedence: specified -> operation-ID -> method-name
			if operationBaseName, err = getKongName(operation.ExtensionProps); err != nil {
				return nil, err
			}
			if operationBaseName != "" {
				// an x-kong-name was provided, so build as "doc-path-name"
				operationBaseName = pathBaseName + "-" + Slugify(operationBaseName)
			} else {
				operationBaseName = operation.OperationID
				if operationBaseName == "" {
					// no operation ID provided, so build as "doc-path-method"
					operationBaseName = pathBaseName + "-" + Slugify(strings.ToLower(method))
				} else {
					// operation ID is provided, so build as "doc-operationid"
					operationBaseName = docBaseName + "-" + Slugify(operationBaseName)
				}
			}
			logbasics.Debug("operation base name (namespace for UUID generation)", "name", operationBaseName)

			// Set up the defaults on the Operation level
			newOperationService := false
			if operationServiceDefaults, err = getServiceDefaults(operation.ExtensionProps, kongComponents); err != nil {
				return nil, err
			}
			if operationServiceDefaults == nil {
				operationServiceDefaults = pathServiceDefaults
			} else {
				newOperationService = true
			}

			newUpstream := false
			if operationUpstreamDefaults, err = getUpstreamDefaults(operation.ExtensionProps, kongComponents); err != nil {
				return nil, err
			}
			if operationUpstreamDefaults == nil {
				operationUpstreamDefaults = pathUpstreamDefaults
			} else {
				newUpstream = true
				newOperationService = true
			}

			if operationRouteDefaults, err = getRouteDefaults(operation.ExtensionProps, kongComponents); err != nil {
				return nil, err
			}
			if operationRouteDefaults == nil {
				operationRouteDefaults = pathRouteDefaults
			}

			// if there is no operation level servers block, use the path one
			operationServers = operation.Servers
			if operationServers == nil || len(*operationServers) == 0 {
				operationServers = pathServers
			} else {
				newUpstream = true
				newOperationService = true
			}

			// create a new service if we need to do so
			if newOperationService {
				// create the operation-level service and (optional) upstream
				logbasics.Debug("creating operation-level service/upstream")
				operationService, operationUpstream, err = CreateKongService(
					operationBaseName,
					operationServers,
					operationServiceDefaults,
					operationUpstreamDefaults,
					kongTags,
					opts.UUIDNamespace)
				if err != nil {
					return nil, fmt.Errorf("failed to create service/updstream from operation '%s %s': %w", path, method, err)
				}
				services = append(services, operationService)
				if operationUpstream != nil {
					// we have a new upstream, but do we need it?
					if newUpstream {
						// we need it, so store and use it
						upstreams = append(upstreams, operationUpstream)
					} else {
						// we don't need it, so update service to point to 'upper' upstream
						operationService["host"] = pathService["host"]
					}
				}
				operationRoutes = operationService["routes"].([]interface{})
			} else {
				operationService = pathService
				operationRoutes = operationService["routes"].([]interface{})
			}

			// collect operation plugins
			if !newOperationService && !newPathService {
				// we're operating on the doc-level service entity, so we need the plugins
				// from the path and operation
				operationPluginList, err = getPluginsList(operation.ExtensionProps, pathPluginList,
					opts.UUIDNamespace, operationBaseName, kongComponents, kongTags)
			} else if newOperationService {
				// we're operating on an operation-level service entity, so we need the plugins
				// from the document, path, and operation.
				operationPluginList, _ = getPluginsList(doc.ExtensionProps, nil, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags)
				operationPluginList, _ = getPluginsList(pathitem.ExtensionProps, operationPluginList, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags)
				operationPluginList, err = getPluginsList(operation.ExtensionProps, operationPluginList, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags)
			} else if newPathService {
				// we're operating on a path-level service entity, so we only need the plugins
				// from the operation.
				operationPluginList, err = getPluginsList(operation.ExtensionProps, nil, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to create plugins list from operation item: %w", err)
			}

			// Extract the request-validator config from the plugin list, generate it and reinsert
			operationValidatorConfig, operationPluginList = getValidatorPlugin(operationPluginList, pathValidatorConfig)
			validatorPlugin := generateValidatorPlugin(operationValidatorConfig, operation, opts.UUIDNamespace,
				operationBaseName)
			operationPluginList = insertPlugin(operationPluginList, validatorPlugin)

			// construct the route
			var route map[string]interface{}
			if operationRouteDefaults != nil {
				_ = json.Unmarshal(operationRouteDefaults, &route)
				delete(route, "service") // always clear foreign keys to services, not allowed
			} else {
				route = make(map[string]interface{})
			}

			// move consumer bound plugins to doc level plugins list (multiple foreign keys)
			foreignKeyPlugins, operationPluginList = getForeignKeyPlugins(
				foreignKeyPlugins, operationPluginList, "route", operationBaseName)

			// attach the collected plugins configs to the route
			route["plugins"] = operationPluginList

			// Escape path contents for regex creation
			convertedPath := path
			charsToEscape := []string{"(", ")", ".", "+", "?", "*", "["}
			for _, char := range charsToEscape {
				convertedPath = strings.ReplaceAll(convertedPath, char, "\\"+char)
			}

			// convert path parameters to regex captures
			re, _ := regexp.Compile("{([^}]+)}")
			regexPriority := 200 // non-regexed (no params) paths have higher precedence in OAS
			if matches := re.FindAllStringSubmatch(convertedPath, -1); matches != nil {
				regexPriority = 100
				for _, match := range matches {
					varName := match[1]
					// match single segment; '/', '?', and '#' can mark the end of a segment
					// see https://github.com/OAI/OpenAPI-Specification/issues/291#issuecomment-316593913
					regexMatch := "(?<" + sanitizeRegexCapture(varName) + ">[^#?/]+)"
					placeHolder := "{" + varName + "}"
					logbasics.Debug("replacing path parameter", "parameter", placeHolder, "regex", regexMatch)
					convertedPath = strings.Replace(convertedPath, placeHolder, regexMatch, 1)
				}
			}
			route["paths"] = []string{"~" + convertedPath + "$"}
			route["id"] = uuid.NewSHA1(opts.UUIDNamespace, []byte(operationBaseName+".route")).String()
			route["name"] = operationBaseName
			route["methods"] = []string{method}
			route["tags"] = kongTags
			route["regex_priority"] = regexPriority
			if _, found := route["strip_path"]; !found {
				route["strip_path"] = false // Default to false since we do not want to strip full-regex paths by default
			}

			operationRoutes = append(operationRoutes, route)
			operationService["routes"] = operationRoutes
		}
	}

	// export arrays with services, upstreams, and plugins to the final object
	result["services"] = services
	result["upstreams"] = upstreams
	if len(*foreignKeyPlugins) > 0 {
		sort.Slice(*foreignKeyPlugins,
			func(i, j int) bool {
				p1 := *(*foreignKeyPlugins)[i]
				p2 := *(*foreignKeyPlugins)[j]
				k1 := p1["name"].(string) + p1["id"].(string)
				k2 := p2["name"].(string) + p2["id"].(string)
				return k1 < k2
			})
		result["plugins"] = foreignKeyPlugins
	}

	// we're done!
	logbasics.Debug("finished processing document")
	return result, nil
}
