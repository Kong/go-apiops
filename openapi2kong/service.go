package openapi2kong

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
)

const (
	httpScheme  = "http"
	httpsScheme = "https"
)

// parseServerUris parses the server uri's after rendering the template variables.
// result will always have at least 1 entry, but not necessarily a hostname/port/scheme
func parseServerUris(servers *openapi3.Servers) ([]*url.URL, error) {
	var targets []*url.URL

	if servers == nil || len(*servers) == 0 {
		uriObject, _ := url.ParseRequestURI("/") // path '/' is the default for empty server blocks
		targets = make([]*url.URL, 1)
		targets[0] = uriObject

	} else {
		targets = make([]*url.URL, len(*servers))

		for i, server := range *servers {
			uriString := server.URL
			for name, svar := range server.Variables {
				uriString = strings.ReplaceAll(uriString, "{"+name+"}", svar.Default)
			}

			uriObject, err := url.ParseRequestURI(uriString)
			if err != nil {
				return targets, fmt.Errorf("failed to parse uri '%s'; %w", uriString, err)
			}

			if uriObject.Path == "" {
				uriObject.Path = "/" // path '/' is the default
			}

			targets[i] = uriObject
		}
	}

	return targets, nil
}

// setServerDefaults sets the scheme and port if missing and inferable.
// It's set based on; scheme given, port (80/443), default-scheme. In that order.
func setServerDefaults(targets []*url.URL, schemeDefault string) {
	for _, target := range targets {
		// set the hostname if unset
		if target.Host == "" {
			target.Host = "localhost"
		}

		// set the scheme if unset
		if target.Scheme == "" {
			// detect scheme from the port
			switch target.Port() {
			case "80":
				target.Scheme = httpScheme

			case "443":
				target.Scheme = httpsScheme

			default:
				target.Scheme = schemeDefault
			}
		}

		// set the port if unset (but a host is given)
		if target.Host != "" && target.Port() == "" {
			if target.Scheme == httpScheme {
				target.Host = target.Host + ":80"
			}
			if target.Scheme == httpsScheme {
				target.Host = target.Host + ":443"
			}
		}
	}
}

func parseDefaultTargets(targets interface{}, tags []string) ([]map[string]interface{}, error) {
	// validate that its an array
	var targetArray []interface{}
	switch t := targets.(type) {
	case []interface{}:
		targetArray = t
	default:
		return nil, fmt.Errorf("expected 'targets' to be an array")
	}

	resultTargets := make([]map[string]interface{}, len(targetArray))
	for i, targetInterface := range targetArray {
		// validate entry to be a string map
		var target map[string]interface{}
		switch m := targetInterface.(type) {
		case map[string]interface{}:
			target = m
		default:
			return nil, fmt.Errorf("expected entries in 'targets' to be objects")
		}

		// just add/overwrite tags, nothing more to do
		target["tags"] = tags
		resultTargets[i] = target
	}
	return resultTargets, nil
}

// createKongUpstream create a new upstream entity.
func createKongUpstream(
	baseName string, // slugified name of the upstream, and uuid input
	servers *openapi3.Servers, // the OAS3 server block to use for generation
	upstreamDefaults []byte, // defaults to use (JSON string) or empty if no defaults
	tags []string, // tags to attach to the new upstream
	uuidNamespace uuid.UUID,
	skipID bool,
) (map[string]interface{}, error) {
	var upstream map[string]interface{}

	// have to create an upstream with targets
	if upstreamDefaults != nil {
		// got defaults, so apply them
		_ = json.Unmarshal(upstreamDefaults, &upstream)
	} else {
		upstream = make(map[string]interface{})
	}

	upstreamName := baseName + ".upstream"
	if !skipID {
		upstream["id"] = uuid.NewSHA1(uuidNamespace, []byte(upstreamName)).String()
	}
	upstream["name"] = upstreamName
	upstream["tags"] = tags

	if upstream["targets"] != nil {
		// if targets provided in the defaults, so use those
		targets, err := parseDefaultTargets(upstream["targets"], tags)
		if err != nil {
			return nil, err
		}
		upstream["targets"] = targets
		return upstream, nil
	}

	// no target array provided, so take from servers

	// the server urls, will have minimum 1 entry on success
	targets, err := parseServerUris(servers)
	if err != nil {
		return nil, fmt.Errorf("failed to generate upstream: %w", err)
	}

	setServerDefaults(targets, httpsScheme)

	// now add the targets to the upstream
	upstreamTargets := make([]map[string]interface{}, len(targets))
	for i, target := range targets {
		t := make(map[string]interface{})
		t["target"] = target.Host
		t["tags"] = tags
		upstreamTargets[i] = t
	}
	upstream["targets"] = upstreamTargets

	return upstream, nil
}

// CreateKongService creates a new Kong service entity, and optional upstream.
// `baseName` will be used as the name of the service (slugified), and as input
// for the UUIDv5 generation.
func CreateKongService(
	baseName string, // slugified name of the service, and uuid input
	servers *openapi3.Servers,
	serviceDefaults []byte,
	upstreamDefaults []byte,
	tags []string,
	uuidNamespace uuid.UUID,
	skipID bool,
) (map[string]interface{}, map[string]interface{}, error) {
	var (
		service  map[string]interface{}
		upstream map[string]interface{}
	)

	// setup the defaults
	if serviceDefaults != nil {
		_ = json.Unmarshal(serviceDefaults, &service)
	} else {
		service = make(map[string]interface{})
	}

	// add id, name and tags to the service
	if !skipID {
		service["id"] = uuid.NewSHA1(uuidNamespace, []byte(baseName+".service")).String()
	}
	service["name"] = baseName
	service["tags"] = tags
	service["plugins"] = make([]interface{}, 0)
	service["routes"] = make([]interface{}, 0)

	// the server urls, will have minimum 1 entry on success
	targets, err := parseServerUris(servers)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create service: %w", err)
	}

	// fill in the scheme of the url if missing. Use service-defaults for the default scheme
	scheme := httpsScheme
	if service["protocol"] != nil {
		scheme = service["protocol"].(string)
	}
	setServerDefaults(targets, scheme)

	if service["protocol"] == nil {
		scheme = targets[0].Scheme
		service["protocol"] = scheme
	}
	if service["path"] == nil {
		service["path"] = targets[0].Path
	}
	if service["port"] == nil {
		if targets[0].Port() != "" {
			// port is provided, so parse it
			parsedPort, err := strconv.ParseUint(targets[0].Port(), 10, 16)
			if err != nil {
				return nil, nil, err
			}
			service["port"] = parsedPort
		} else {
			// no port provided, so set it based on scheme, where https/443 is the default
			if scheme != httpScheme {
				service["port"] = 443
			} else {
				service["port"] = 80
			}
		}
	}

	// we need an upstream if;
	// a) upstream defaults are provided, or
	// b) there is more than one entry in the servers block
	// c) the service doesn't have a default host name
	if service["host"] == nil {
		if len(targets) == 1 && upstreamDefaults == nil {
			// have to create a simple service, no upstream, so just set the hostname
			service["host"] = targets[0].Hostname()
		} else {
			// have to create an upstream with targets
			upstream, err = createKongUpstream(baseName, servers, upstreamDefaults, tags, uuidNamespace, skipID)
			if err != nil {
				return nil, nil, err
			}
			service["host"] = upstream["name"]
		}
	}

	return service, upstream, nil
}
