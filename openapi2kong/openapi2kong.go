package openapi2kong

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kong/go-apiops/filebasics"
	"github.com/kong/go-apiops/jsonbasics"
	"github.com/kong/go-apiops/logbasics"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	openapibase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

const (
	formatVersionKey   = "_format_version"
	formatVersionValue = "3.0"

	// default regex priorities to assign to routes
	regexPriorityWithPathParams = 100
	regexPriorityPlain          = 200 // non-regexed (no params) paths have higher precedence in OAS
)

// O2KOptions defines the options for an O2K conversion operation
type O2kOptions struct {
	// Array of tags to mark all generated entities with, taken from 'x-kong-tags' if omitted.
	Tags []string
	// Base document name, will be taken from x-kong-name, or info.title (for UUID generation!)
	DocName string
	// Namespace for UUID generation, defaults to DNS namespace for UUID v5
	UUIDNamespace uuid.UUID
	// Enable Inso compatibility mode
	InsoCompat bool
	// Skip ID generation (UUIDs)
	SkipID bool
	// Enable OIDC plugin generation
	OIDC bool
	// Ignore security errors (non-OIDC and AND/OR logic)
	IgnoreSecurityErrors bool
	// Ignore circular references
	IgnoreCircularRefs bool
}

// setDefaults sets the defaults for the OpenAPI2Kong operation.
func (opts *O2kOptions) setDefaults() {
	var emptyUUID uuid.UUID

	if bytes.Equal(emptyUUID[:], opts.UUIDNamespace[:]) {
		opts.UUIDNamespace = uuid.NameSpaceDNS
	}
}

// getKongTags returns the provided tags or if nil, then the `x-kong-tags` property,
// validated to be a string array. If there is no error, then there will always be
// an array returned for safe access later in the process.
func getKongTags(doc v3.Document, tagsProvided []string) ([]string, error) {
	if tagsProvided != nil {
		// the provided tags take precedence, return them
		return tagsProvided, nil
	}

	if doc.Extensions == nil {
		// there is no extension, so return an empty array
		return make([]string, 0), nil
	}

	kongTags, ok := doc.Extensions.Get("x-kong-tags")
	if !ok {
		// there is no extension by the name "x-kong-tag", so return an empty array
		return make([]string, 0), nil
	}

	resultArray := make([]string, len(kongTags.Content))
	for i, v := range kongTags.Content {
		var tagsValue interface{}
		err := yaml.Unmarshal([]byte(v.Value), &tagsValue)
		if err != nil {
			return nil, fmt.Errorf("expected 'x-kong-tags' to be an array of strings: %w", err)
		}

		switch tag := tagsValue.(type) {
		case string:
			resultArray[i] = tag
		default:
			return nil, fmt.Errorf("expected 'x-kong-tags' to be an array of strings")
		}
	}

	return resultArray, nil
}

// getKongName returns the `x-kong-name` property, validated to be a string
func getKongName(extensions *orderedmap.Map[string, *yaml.Node]) (string, error) {
	if extensions == nil {
		return "", nil
	}

	xKongName, ok := extensions.Get("x-kong-name")

	if !ok {
		return "", nil
	}

	var name string
	err := yaml.Unmarshal([]byte(xKongName.Value), &name)
	if err != nil {
		return "", fmt.Errorf("expected 'x-kong-name' to be a string: %w", err)
	}

	return name, nil
}

// getXKongObject returns specified 'key' from the extension properties if available.
// returns nil if it wasn't found, an error if it wasn't an object or couldn't be
// dereferenced. The returned object will be json encoded again.
func getXKongObject(
	extensions *orderedmap.Map[string, *yaml.Node],
	key string, components *map[string]interface{},
) ([]byte, error) {
	if extensions == nil {
		return nil, nil
	}

	xKongObject, ok := extensions.Get(key)
	if !ok || xKongObject == nil {
		return nil, nil
	}

	xKongObjectBytes, err := convertYamlNodeToBytes(xKongObject)
	if err != nil {
		return nil, fmt.Errorf("expected '%s' to be a YAML object", key)
	}

	var jsonBlob interface{}
	_ = yaml.Unmarshal(xKongObjectBytes, &jsonBlob)
	jsonObject, err := jsonbasics.ToObject(jsonBlob)
	if err != nil {
		return nil, fmt.Errorf("expected '%s' to be a JSON/YAML object", key)
	}

	object, err := dereferenceJSONObject(jsonObject, components)
	if err != nil {
		return nil, err
	}
	return json.Marshal(object)
}

// getXKongComponents will return a map of the '/components/x-kong/' object. If
// the extension is not there it will return an empty map. If the entry is not a
// yaml object, it will return an error.
func getXKongComponents(doc v3.Document) (*map[string]interface{}, error) {
	var components map[string]interface{}

	if doc.Components == nil || doc.Components.Extensions == nil {
		return &map[string]interface{}{}, nil
	}

	xKongComponents, ok := doc.Components.Extensions.Get("x-kong")

	if !ok || xKongComponents == nil {
		return &components, nil
	}

	xKongComponentsBytes, err := convertYamlNodeToBytes(xKongComponents)
	if err != nil {
		return nil, fmt.Errorf("expected '/components/x-kong' to be a YAML object")
	}

	var xKong interface{}
	_ = yaml.Unmarshal(xKongComponentsBytes, &xKong)
	components, err = jsonbasics.ToObject(xKong)
	if err != nil {
		return nil, fmt.Errorf("expected '/components/x-kong' to be a JSON/YAML object")
	}

	return &components, nil
}

// getServiceDefaults returns a JSON string containing the defaults
func getServiceDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-service-defaults", components)
}

// getUpstreamDefaults returns a JSON string containing the defaults
func getUpstreamDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-upstream-defaults", components)
}

// getRouteDefaults returns a JSON string containing the defaults
func getRouteDefaults(
	extensions *orderedmap.Map[string, *yaml.Node],
	components *map[string]interface{},
) ([]byte, error) {
	return getXKongObject(extensions, "x-kong-route-defaults", components)
}

// getOIDCdefaults returns a JSON string containing the defaults from the SecurityRequirements. The type must
// be "openIdConnect" or an error is returned. If there are no security requirements, it returns the "inherited" value.
// If the extension is not there it will return an empty map. If the entry is not a
// Json object, it will return an error.
func getOIDCdefaults(
	requirements []*openapibase.SecurityRequirement, // the security requirements to parse
	doc v3.Document, // the complete OAS document
	inherited []byte, // the inherited OIDC defaults
	ignoreSecurityErrors bool, // ignore unsupported security requirements (return "inherited" instead of error)
) ([]byte, error) {
	// Collect the OAS specific properties
	var (
		// requirements openapi3.SecurityRequirements // the security requirements to parse
		schemeName string             // the name of the security-scheme
		scopes     []string           // the scopes required for the security-scheme
		scheme     *v3.SecurityScheme // the security-scheme object
	)
	{
		if len(requirements) == 0 || ignoreSecurityErrors {
			// no security requirements or nothing is defined
			// so return inherited (can be nil)
			return inherited, nil
		}

		if len(requirements) > 1 && !ignoreSecurityErrors {
			return nil, fmt.Errorf("only a single security-requirement is supported")
		}

		requirement := requirements[0].Requirements
		if requirement.Len() == 0 || ignoreSecurityErrors {
			return inherited, nil // there is nothing defined, so return inherited (can be nil)
		}

		if requirement.Len() > 1 && !ignoreSecurityErrors {
			// multiple schemes are a logical AND, which is not supported
			return nil, fmt.Errorf("within a security-requirement only a single security-scheme is supported")
		}

		// requirement has only 1 entry
		// So, we won't iterate
		reqPair := requirement.First()
		schemeName = reqPair.Key()
		scopes = reqPair.Value()

		schemes := doc.Components.SecuritySchemes
		scheme, _ = schemes.Get(schemeName)

		if scheme.Type != "openIdConnect" {
			// non-OIDC security directives are not supported
			if !ignoreSecurityErrors {
				return nil, fmt.Errorf("only security-schemes of type 'openIdConnect' are supported")
			} else {
				return inherited, nil
			}
		}
	}

	// Construct the base plugin object from x-kong-security...
	var (
		pluginBase   map[string]interface{} // the plugin object
		pluginConfig map[string]interface{} // the plugin.config object
	)
	{
		kongComponents, err := getXKongComponents(doc)
		if err != nil {
			return nil, err
		}

		// grab the base plugin config from the x-kong-... directive
		pluginBaseData, err := getXKongObject(scheme.Extensions, "x-kong-security-openid-connect", kongComponents)
		if err != nil {
			return nil, err
		}
		if pluginBaseData == nil {
			// no x-kong-... plugin config, so create an empty one
			pluginBase = make(map[string]interface{})
		} else {
			pluginBase, _ = filebasics.Deserialize(pluginBaseData)
		}

		// ensure we have a plugin.config object
		if pluginBase["config"] == nil {
			pluginBase["config"] = make(map[string]interface{})
		}
		pluginConfig, err = jsonbasics.ToObject(pluginBase["config"])
		if err != nil {
			return nil, err
		}
	}

	// Collect all required scopes, from OAS and x-kong-security..
	var ScopesRequired []string
	{
		var err error
		ScopesRequired, err = jsonbasics.GetStringArrayField(pluginConfig, "scopes_required")
		if err != nil {
			return nil, err
		}

		// merge the scopes from the security-requirement with the scopes from the plugin config
		for _, scope1 := range scopes {
			duplicate := false
			for _, scope2 := range ScopesRequired {
				if scope1 == scope2 {
					duplicate = true
					break
				}
			}
			if !duplicate {
				ScopesRequired = append(ScopesRequired, scope1)
			}
		}
		// sort scopesRequired array for deterministic output
		sort.Strings(ScopesRequired)
	}

	// construct the final plugin
	pluginBase["name"] = "openid-connect"
	pluginConfig["scopes_required"] = ScopesRequired
	if scheme.OpenIdConnectUrl != "" && pluginConfig["issuer"] == nil {
		// only set the issuer if it wasn't already set in the plugin config, because the
		// x-kong-... specifies the Kong behaviour, the OAS specifies the service-behind-kong
		// behaviour. So the former should win.
		pluginConfig["issuer"] = scheme.OpenIdConnectUrl
	}

	return filebasics.Serialize(pluginBase, filebasics.OutputFormatJSON)
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
	extensions *orderedmap.Map[string, *yaml.Node],
	componentExtensions *orderedmap.Map[string, *yaml.Node],
	pluginsToInclude *[]*map[string]interface{},
	uuidNamespace uuid.UUID,
	baseName string,
	components *map[string]interface{},
	tags []string,
	skipID bool,
) (*[]*map[string]interface{}, error) {
	plugins := make(map[string]*map[string]interface{})

	// copy inherited list of plugins
	if pluginsToInclude != nil {
		for _, config := range *pluginsToInclude {
			pluginName := (*config)["name"].(string) // safe because it was previously parsed
			configCopy := jsonbasics.DeepCopyObject(*config)

			// generate a new ID, for a new plugin, based on new basename
			if !skipID {
				configCopy["id"] = createPluginID(uuidNamespace, baseName, configCopy)
			}
			configCopy["tags"] = tags

			plugins[pluginName] = &configCopy
		}
	}

	if extensions == nil && componentExtensions == nil {
		emptyList := make([]*map[string]interface{}, 0)
		// We will return an empty list instead of nil so consumers can avoid having to do nil check.
		return &emptyList, nil
	}

	// there are extensions, go check if there are plugins
	extension := extensions.First()
	for extension != nil {
		extensionName := extension.Key()
		if strings.HasPrefix(extensionName, "x-kong-plugin-") {
			pluginName := strings.TrimPrefix(extensionName, "x-kong-plugin-")

			jsonstr, err := getXKongObject(extensions, extensionName, components)
			if err != nil {
				return nil, err
			}

			var pluginConfig map[string]interface{}
			err = json.Unmarshal(jsonstr, &pluginConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON object for '%s': %w", extensionName, err)
			}

			pluginConfig["name"] = pluginName
			if !skipID {
				pluginConfig["id"] = createPluginID(uuidNamespace, baseName, pluginConfig)
			}
			pluginConfig["tags"] = tags

			// foreign keys to service+route are not allowed (consumer is allowed)
			delete(pluginConfig, "service")
			delete(pluginConfig, "route")

			plugins[pluginName] = &pluginConfig
		}

		extension = extension.Next()
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
	// search for the request-validator plugin
	if list == nil {
		return currentConfig, list
	}

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
	docPlugins *[]*map[string]interface{}, // the current list of doc-level plugins (may be nil)
	pluginList *[]*map[string]interface{}, // the list of entity-level plugins to check for foreign keys (may be nil)
	foreignKey string, // the owner entity type: eg. "service", or "route"
	foreignValue string, // the owner entity name/id: the value (service/route name)
) (
	*[]*map[string]interface{}, // updated slice of document level plugins
	*[]*map[string]interface{}, // updated slice of entity level plugins
) {
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

// findParameterSchema returns the Schema for given parameter name.
// Path level parameters can be overridden at operation level, so we check operation parameters first
// and then fall back to path.
func findParameterSchema(
	operationParameters []*v3.Parameter,
	pathParameters []*v3.Parameter,
	paramName string,
) *openapibase.Schema {
	for _, param := range operationParameters {
		if param.Name == paramName {
			return param.Schema.Schema()
		}
	}
	for _, param := range pathParameters {
		if param.Name == paramName {
			return param.Schema.Schema()
		}
	}
	return nil
}

// Returns header parameters which can be used for routing for an operation
func findHeaderParamsForRouting(
	operationLevelParameters []*v3.Parameter,
	pathLevelParameters []*v3.Parameter,
) []*v3.Parameter {
	headerParamProcessed := make(map[string]bool)
	var result []*v3.Parameter // Store in array so output is deterministic - iterating over map is not.

	for _, param := range operationLevelParameters {
		if param.In == "header" && param.Schema != nil &&
			param.Schema.Schema() != nil && len(param.Schema.Schema().Enum) > 0 {
			headerParamProcessed[param.Name] = true
			result = append(result, param)
		}
	}

	for _, param := range pathLevelParameters {
		// Operation level params override path level params, so ignore if already present.
		if param.In == "header" && param.Schema != nil && param.Schema.Schema() != nil &&
			len(param.Schema.Schema().Enum) > 0 && !headerParamProcessed[param.Name] {
			headerParamProcessed[param.Name] = true
			result = append(result, param)
		}
	}

	return result
}

// Based on given headers and their possible values, create all possible combinations
func constructHeaderCombinationsForRouting(headers []*v3.Parameter) []map[string]any {
	headerValues := make([][]any, len(headers))
	headerNames := make([]string, len(headers))
	for i := 0; i < len(headers); i++ {
		headerNames[i] = headers[i].Name
		for _, enumMember := range headers[i].Schema.Schema().Enum {
			headerValues[i] = append(headerValues[i], enumMember.Value)
		}
	}
	headerValueCombinations := crossProduct(headerValues...)

	var result []map[string]any
	for _, combination := range headerValueCombinations {
		headerMap := make(map[string]any)
		for i := 0; i < len(headerNames); i++ {
			// Header is an array of values.
			headerMap[headerNames[i]] = []any{combination[i]}
		}
		result = append(result, headerMap)
	}

	return result
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
		err              error
		removeDocService bool // set to true if no docServers are present;
		// it's used in case services is empty
		doc            v3.Document             // the OAS3 document we're operating on
		kongComponents *map[string]interface{} // contents of OAS key `/components/x-kong/`
		kongTags       []string                // tags to attach to Kong entities
		nameConcatChar string                  // character to use for concatenating names

		docBaseName         string                     // the slugified basename for the document
		docServers          []*v3.Server               // servers block on document level
		docServiceDefaults  []byte                     // JSON string representation of service-defaults on document level
		docService          map[string]interface{}     // service entity in use on document level
		docUpstreamDefaults []byte                     // JSON string representation of upstream-defaults on document level
		docUpstream         map[string]interface{}     // upstream entity in use on document level
		docRouteDefaults    []byte                     // JSON string representation of route-defaults on document level
		docPluginList       *[]*map[string]interface{} // array of plugin configs, sorted by plugin name
		docValidatorConfig  []byte                     // JSON string representation of validator config to generate
		docOIDCdefaults     []byte                     // JSON string representation of OIDC config to generate
		foreignKeyPlugins   *[]*map[string]interface{} // top-level array of plugin configs, sorted by plugin name+id

		pathBaseName         string                     // the slugified basename for the path
		pathServers          []*v3.Server               // servers block on current path level
		pathServiceDefaults  []byte                     // JSON string representation of service-defaults on path level
		pathService          map[string]interface{}     // service entity in use on path level
		pathUpstreamDefaults []byte                     // JSON string representation of upstream-defaults on path level
		pathUpstream         map[string]interface{}     // upstream entity in use on path level
		pathRouteDefaults    []byte                     // JSON string representation of route-defaults on path level
		pathPluginList       *[]*map[string]interface{} // array of plugin configs, sorted by plugin name
		pathValidatorConfig  []byte                     // JSON string representation of validator config to generate

		operationBaseName         string                     // the slugified basename for the operation
		operationServers          []*v3.Server               // servers block on current operation level
		operationServiceDefaults  []byte                     // JSON string representation of service-defaults on ops level
		operationService          map[string]interface{}     // service entity in use on operation level
		operationUpstreamDefaults []byte                     // JSON string representation of upstream-defaults on ops level
		operationUpstream         map[string]interface{}     // upstream entity in use on operation level
		operationRouteDefaults    []byte                     // JSON string representation of route-defaults on ops level
		operationPluginList       *[]*map[string]interface{} // array of plugin configs, sorted by plugin name
		operationValidatorConfig  []byte                     // JSON string representation of validator config to generate
	)

	if opts.InsoCompat {
		nameConcatChar = "-"
	} else {
		nameConcatChar = "_"
	}

	// Load and parse the OAS file
	openapiDoc, err := libopenapi.NewDocument(content)
	if err != nil {
		return nil, fmt.Errorf("error parsing OAS3 file: [%w]", err)
	}

	// Check if circular references must be ignored
	if opts.IgnoreCircularRefs {
		docConfig := datamodel.NewDocumentConfiguration()
		docConfig.IgnoreArrayCircularReferences = true
		docConfig.IgnorePolymorphicCircularReferences = true
		openapiDoc.SetConfiguration(docConfig)
	}

	// var errors []error
	v3Model, errs := openapiDoc.BuildV3Model()

	// if anything went wrong when building the v3 model,
	// a slice of errors will be returned
	if len(errs) > 0 {
		for i := range errs {
			logbasics.Error(errs[i], "error while building v3 document model \n")
		}
		return nil, fmt.Errorf("cannot create v3 model from document: %d errors reported", len(errs))
	}

	if v3Model != nil {
		doc = v3Model.Model
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
	docServers = doc.Servers

	// determine document name, precedence: specified -> x-kong-name -> Info.Title -> random
	docBaseName = opts.DocName
	if docBaseName == "" {
		logbasics.Debug("no document name specified, trying x-kong-name")
		if docBaseName, err = getKongName(doc.Extensions); err != nil {
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
	docBaseName = Slugify(opts.InsoCompat, docBaseName)
	logbasics.Info("document name (namespace for UUID generation)", "name", docBaseName)

	if kongComponents, err = getXKongComponents(doc); err != nil {
		return nil, err
	}

	// for defaults we keep strings, so deserializing them provides a copy right away
	if docServiceDefaults, err = getServiceDefaults(doc.Extensions, kongComponents); err != nil {
		return nil, err
	}
	if docUpstreamDefaults, err = getUpstreamDefaults(doc.Extensions, kongComponents); err != nil {
		return nil, err
	}
	if docRouteDefaults, err = getRouteDefaults(doc.Extensions, kongComponents); err != nil {
		return nil, err
	}

	// create the top-level docService and (optional) docUpstream
	docService, docUpstream, err = CreateKongService(docBaseName, docServers, docServiceDefaults,
		docUpstreamDefaults, kongTags, opts.UUIDNamespace, opts.SkipID)
	if err != nil {
		return nil, fmt.Errorf("failed to create service/upstream from document root: %w", err)
	}

	services = append(services, docService)
	// if there are no document-level servers defined
	// we want to skip adding the default created docService
	// in the services slice. This is done to ensure that
	// an unintended extra service is not created.
	// However, if there are no servers defined, anywhere
	// else in the document, we still want to
	// create the default docService, so as to not return
	// an empty services array.
	// If there are other servers defined, we will
	// remove the docService from the services slice in the end.
	if len(docServers) == 0 {
		removeDocService = true
	}
	if docUpstream != nil {
		upstreams = append(upstreams, docUpstream)
	}

	// attach plugins
	var componentExtensions *orderedmap.Map[string, *yaml.Node]
	if doc.Components != nil && doc.Components.Extensions != nil {
		componentExtensions = doc.Components.Extensions
	}

	docPluginList, err = getPluginsList(doc.Extensions, componentExtensions, nil, opts.UUIDNamespace, docBaseName,
		kongComponents, kongTags, opts.SkipID)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugins list from document root: %w", err)
	}

	// get the OIDC stuff from top level, bail out if the requirements are unsupported
	if opts.OIDC {
		docOIDCdefaults, err = getOIDCdefaults(doc.Security, doc, nil, opts.IgnoreSecurityErrors)
		if err != nil {
			return nil, err
		}
		if docOIDCdefaults != nil {
			// we have OIDC defaults, so we need to add the plugin to the doc-level list
			pluginConfig, _ := filebasics.Deserialize(docOIDCdefaults)
			docPluginList = insertPlugin(docPluginList, &pluginConfig)
		}
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

	if doc.Paths == nil {
		return nil, fmt.Errorf("must have `.paths` in the root of the document. " +
			"See examples https://github.com/Kong/go-apiops/tree/main/docs")
	}

	// create a sorted array of paths, to be deterministic in our output order
	allPaths := doc.Paths.PathItems
	sortedPaths := make([]string, allPaths.Len())
	path := allPaths.First()
	i := 0
	for path != nil && i < allPaths.Len() {
		sortedPaths[i] = path.Key()
		i++
		path = path.Next()
	}
	sort.Strings(sortedPaths)

	for _, pathKey := range sortedPaths {
		pathitem, ok := allPaths.Get(pathKey)
		if !ok {
			continue
		}

		logbasics.Info("processing path", "path", pathKey)

		// determine path name, precedence: specified -> x-kong-name -> actual-path
		if pathBaseName, err = getKongName(pathitem.Extensions); err != nil {
			return nil, err
		}
		if pathBaseName == "" {
			// no given name, so use the path itself to construct the name
			if !opts.InsoCompat {
				pathBaseName = Slugify(opts.InsoCompat, pathKey)
				if strings.HasSuffix(pathKey, "/") {
					// a common case is 2 paths, one with and one without a trailing "/" so to prevent
					// duplicate names being generated, we add a "~" suffix as a special case to cater
					// for different names. Better user solution is to use operation-id's.
					pathBaseName = pathBaseName + "~"
				}
			} else {
				// we need inso compatibility
				pathBaseName = Slugify(opts.InsoCompat, pathKey)
			}
		} else {
			pathBaseName = Slugify(opts.InsoCompat, pathBaseName)
		}
		if pathBaseName != "" {
			pathBaseName = docBaseName + nameConcatChar + pathBaseName
		} else {
			pathBaseName = docBaseName
		}
		logbasics.Debug("path name (namespace for UUID generation)", "name", pathBaseName)

		// Set up the defaults on the Path level
		newPathService := false
		if pathServiceDefaults, err = getServiceDefaults(pathitem.Extensions, kongComponents); err != nil {
			return nil, err
		}
		if pathServiceDefaults == nil {
			pathServiceDefaults = docServiceDefaults
		} else {
			newPathService = true
		}

		newUpstream := false
		if pathUpstreamDefaults, err = getUpstreamDefaults(pathitem.Extensions, kongComponents); err != nil {
			return nil, err
		}
		if pathUpstreamDefaults == nil {
			pathUpstreamDefaults = docUpstreamDefaults
		} else {
			newUpstream = true
			newPathService = true
		}

		if pathRouteDefaults, err = getRouteDefaults(pathitem.Extensions, kongComponents); err != nil {
			return nil, err
		}
		if pathRouteDefaults == nil {
			pathRouteDefaults = docRouteDefaults
		}

		// if there is no path level servers block, use the document one
		pathServers = pathitem.Servers
		if len(pathServers) == 0 { // it's always set, so we ignore it if empty
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
				opts.UUIDNamespace,
				opts.SkipID)
			if err != nil {
				return nil, fmt.Errorf("failed to create service/updstream from path '%s': %w", path, err)
			}

			// collect path plugins, including the doc-level plugins since we have a new service entity
			pathPluginList, err = getPluginsList(pathitem.Extensions, nil, docPluginList,
				opts.UUIDNamespace, pathBaseName, kongComponents, kongTags, opts.SkipID)
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
			pathPluginList, err = getPluginsList(pathitem.Extensions, componentExtensions, nil,
				opts.UUIDNamespace, pathBaseName, kongComponents, kongTags, opts.SkipID)
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
		operations := pathitem.GetOperations()

		sortedMethods := make([]string, operations.Len())
		method := operations.First()
		i := 0
		for method != nil && i < operations.Len() {
			sortedMethods[i] = method.Key()
			i++
			method = method.Next()
		}
		sort.Strings(sortedMethods)

		// traverse all operations
		for _, methodKey := range sortedMethods {
			operation, ok := operations.Get(methodKey)
			if !ok {
				continue
			}

			methodKey = strings.ToUpper(methodKey)

			logbasics.Info("processing operation", "method", methodKey, "path", path, "id", operation.OperationId)

			var operationRoutes []interface{} // the routes array we need to add to

			// determine operation name, precedence: specified -> operation-ID -> method-name
			if operationBaseName, err = getKongName(operation.Extensions); err != nil {
				return nil, err
			}
			if operationBaseName != "" {
				if opts.InsoCompat {
					// an x-kong-name was provided, so build as "doc-name"
					operationBaseName = docBaseName + nameConcatChar + Slugify(opts.InsoCompat, operationBaseName)
				} else {
					// an x-kong-name was provided, so build as "doc-path-name"
					operationBaseName = pathBaseName + nameConcatChar + Slugify(opts.InsoCompat, operationBaseName)
				}
			} else {
				operationBaseName = operation.OperationId
				if operationBaseName == "" {
					// no operation ID provided, so build as "doc-path-method"
					if opts.InsoCompat {
						operationBaseName = pathBaseName + nameConcatChar + Slugify(opts.InsoCompat, strings.ToLower(methodKey))
					} else {
						operationBaseName = pathBaseName + nameConcatChar + Slugify(opts.InsoCompat, methodKey)
					}
				} else {
					// operation ID is provided, so build as "doc-operationid"
					operationBaseName = docBaseName + nameConcatChar + Slugify(opts.InsoCompat, operationBaseName)
				}
			}
			logbasics.Debug("operation base name (namespace for UUID generation)", "name", operationBaseName)

			// Set up the defaults on the Operation level
			newOperationService := false
			if operationServiceDefaults, err = getServiceDefaults(operation.Extensions, kongComponents); err != nil {
				return nil, err
			}
			if operationServiceDefaults == nil {
				operationServiceDefaults = pathServiceDefaults
			} else {
				newOperationService = true
			}

			newUpstream := false
			if operationUpstreamDefaults, err = getUpstreamDefaults(operation.Extensions, kongComponents); err != nil {
				return nil, err
			}
			if operationUpstreamDefaults == nil {
				operationUpstreamDefaults = pathUpstreamDefaults
			} else {
				newUpstream = true
				newOperationService = true
			}

			if operationRouteDefaults, err = getRouteDefaults(operation.Extensions, kongComponents); err != nil {
				return nil, err
			}
			if operationRouteDefaults == nil {
				operationRouteDefaults = pathRouteDefaults
			}

			// if there is no operation level servers block, use the path one
			operationServers = operation.Servers
			if len(operationServers) == 0 {
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
					opts.UUIDNamespace,
					opts.SkipID)
				if err != nil {
					return nil, fmt.Errorf("failed to create service/updstream from operation '%s %s': %w", pathKey, methodKey, err)
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
				operationPluginList, err = getPluginsList(operation.Extensions, nil, pathPluginList,
					opts.UUIDNamespace, operationBaseName, kongComponents, kongTags, opts.SkipID)
			} else if newOperationService {
				// we're operating on an operation-level service entity, so we need the plugins
				// from the document, path, and operation.
				operationPluginList, _ = getPluginsList(doc.Extensions, nil, nil, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags, opts.SkipID)
				operationPluginList, _ = getPluginsList(pathitem.Extensions, nil, operationPluginList, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags, opts.SkipID)
				operationPluginList, err = getPluginsList(operation.Extensions, nil, operationPluginList, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags, opts.SkipID)
			} else if newPathService {
				// we're operating on a path-level service entity, so we only need the plugins
				// from the operation.
				operationPluginList, err = getPluginsList(operation.Extensions, nil, nil, opts.UUIDNamespace,
					operationBaseName, kongComponents, kongTags, opts.SkipID)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to create plugins list from operation item: %w", err)
			}

			if opts.OIDC {
				// get the OIDC stuff from operation level, bail out if the requirements are unsupported
				operationOIDCplugin, err := getOIDCdefaults(operation.Security, doc, docOIDCdefaults, opts.IgnoreSecurityErrors)
				if err != nil {
					return nil, err
				}
				if string(operationOIDCplugin) != string(docOIDCdefaults) {
					// we have OIDC config different from the doc-level one, so we need to add the plugin to the Operation
					pluginConfig, _ := filebasics.Deserialize(operationOIDCplugin)
					operationPluginList = insertPlugin(operationPluginList, &pluginConfig)
				}
			}

			// Extract the request-validator config from the plugin list, generate it and reinsert
			operationValidatorConfig, operationPluginList = getValidatorPlugin(operationPluginList, pathValidatorConfig)
			validatorPlugin, err := generateValidatorPlugin(operationValidatorConfig, operation, pathitem, opts.UUIDNamespace,
				operationBaseName, opts.SkipID, opts.InsoCompat)
			if err != nil {
				return nil, fmt.Errorf("failed to create validator plugin: %w", err)
			}

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
			convertedPath := pathKey
			charsToEscape := []string{"(", ")", ".", "+", "?", "*", "[", "$"}
			for _, char := range charsToEscape {
				convertedPath = strings.ReplaceAll(convertedPath, char, "\\"+char)
			}

			// convert path parameters to regex captures
			re, _ := regexp.Compile("{([^}]+)}")
			regexPriority := regexPriorityPlain
			if matches := re.FindAllStringSubmatch(convertedPath, -1); matches != nil {
				regexPriority = regexPriorityWithPathParams
				for _, match := range matches {
					varName := match[1]
					// match single segment; '/', '?', and '#' can mark the end of a segment
					// see https://github.com/OAI/OpenAPI-Specification/issues/291#issuecomment-316593913
					captureName := sanitizeRegexCapture(varName, opts.InsoCompat)
					if len(captureName) >= 32 {
						return nil, fmt.Errorf("path-parameter name exceeds 32 characters: '%s' (sanitized to '%s')",
							varName, captureName)
					}
					regexMatch := "(?<" + captureName + ">[^#?/]+)"
					paramSchema := findParameterSchema(operation.Parameters, pathitem.Parameters, varName)
					// Check if the parameter has a minLength defined, if 0, allow empty string
					if paramSchema != nil && paramSchema.MinLength != nil && *paramSchema.MinLength == 0 {
						regexMatch = "(?<" + captureName + ">[^#?/]*)"
					}
					placeHolder := "{" + varName + "}"
					logbasics.Debug("replacing path parameter", "parameter", placeHolder, "regex", regexMatch)
					convertedPath = strings.Replace(convertedPath, placeHolder, regexMatch, 1)
				}
			}
			route["paths"] = []string{"~" + convertedPath + "$"}
			if !opts.SkipID {
				route["id"] = uuid.NewSHA1(opts.UUIDNamespace, []byte(operationBaseName+".route")).String()
			}
			route["name"] = operationBaseName
			route["methods"] = []string{methodKey}
			route["tags"] = kongTags
			if _, found := route["regex_priority"]; !found {
				route["regex_priority"] = regexPriority
			} else {
				// a regex_priority was provided in the defaults
				currentRegexPrio, err := jsonbasics.GetInt64Field(route, "regex_priority")
				if err != nil {
					return nil, fmt.Errorf("failed to parse 'regex_priority' from route defaults: %w", err)
				}
				// the default in x-kong-route-defaults represents the plain path, path-parameter path needs to be lower
				if regexPriority == regexPriorityWithPathParams {
					// this is a path with parameters, so we need to lower the priority
					route["regex_priority"] = currentRegexPrio - 1
				}
			}
			if _, found := route["strip_path"]; !found {
				route["strip_path"] = false // Default to false since we do not want to strip full-regex paths by default
			}

			headerParams := findHeaderParamsForRouting(operation.Parameters, pathitem.Parameters)
			if len(headerParams) > 0 {
				// This operation has header parameters, we need to create different routes based on the header values
				headerValueCombinations := constructHeaderCombinationsForRouting(headerParams)

				newRoutes := make([]interface{}, 0)
				for i, combination := range headerValueCombinations {
					clonedRoute := jsonbasics.DeepCopyObject(route)
					clonedRoute["headers"] = combination
					clonedRoute["name"] = fmt.Sprintf("%s_%v", operationBaseName, i)
					if !opts.SkipID {
						clonedRoute["id"] = uuid.NewSHA1(opts.UUIDNamespace, []byte(clonedRoute["name"].(string))).String()
					}
					newRoutes = append(newRoutes, clonedRoute)
				}
				operationRoutes = append(operationRoutes, newRoutes...)
			} else {
				operationRoutes = append(operationRoutes, route)
			}
			operationService["routes"] = operationRoutes
		}
	}

	// export arrays with services, upstreams, and plugins to the final object
	if len(services) > 1 && removeDocService {
		// we have more than one service, and the docService is not needed, so remove it
		result["services"] = services[1:]
	} else {
		result["services"] = services
	}
	result["upstreams"] = upstreams
	if len(*foreignKeyPlugins) > 0 {

		// getSortKey returns a string that can be used to sort the plugins by name, service, route, and consumer (all
		// the foreign keys that are possible).
		getSortKey := func(p *map[string]interface{}) string {
			plugin := *p
			sep := string([]byte{0})
			key := plugin["name"].(string) + sep

			if plugin["service"] != nil {
				key = key + plugin["service"].(string) + sep
			} else {
				key = key + sep
			}

			if plugin["route"] != nil {
				key = key + plugin["route"].(string) + sep
			} else {
				key = key + sep
			}

			if plugin["consumer"] != nil {
				key = key + plugin["consumer"].(string) + sep
			} else {
				key = key + sep
			}

			return key
		}

		sort.Slice(*foreignKeyPlugins,
			func(i, j int) bool {
				return getSortKey((*foreignKeyPlugins)[i]) < getSortKey((*foreignKeyPlugins)[j])
			})
		result["plugins"] = foreignKeyPlugins
	}

	// we're done!
	logbasics.Debug("finished processing document")
	return result, nil
}
