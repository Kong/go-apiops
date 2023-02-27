package deckfile

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/kong/go-apiops/jsonbasics"
)

type DeckFile struct {
	Data                map[string]interface{}
	MajorVersion        int
	MinorVersion        int
	Services            []KongService
	ServicesByName      map[string]*KongService
	ServicesByID        map[string]*KongService
	Routes              []KongRoute
	RoutesByName        map[string]*KongRoute
	RoutesByID          map[string]*KongRoute
	Consumers           []KongConsumer
	ConsumersByID       map[string]*KongConsumer
	ConsumersByUsername map[string]*KongConsumer
	ConsumersByCustomID map[string]*KongConsumer
	Plugins             []KongPlugin
	PluginsByID         map[string]*KongPlugin
	Upstreams           []KongUpstream
	UpstreamsByName     map[string]*KongUpstream
	UpstreamsByID       map[string]*KongUpstream
	Targets             []KongTarget
	TargetsByID         map[string]*KongTarget
}

// NewDeckFile returns a new initialized, but empty, deckfile.
func NewDeckFile() *DeckFile {
	return &DeckFile{
		Data:                make(map[string]interface{}),
		MajorVersion:        3,
		MinorVersion:        0,
		Services:            make([]KongService, 0),
		ServicesByID:        make(map[string]*KongService, 0),
		ServicesByName:      make(map[string]*KongService, 0),
		Routes:              make([]KongRoute, 0),
		RoutesByID:          make(map[string]*KongRoute, 0),
		RoutesByName:        make(map[string]*KongRoute, 0),
		Consumers:           make([]KongConsumer, 0),
		ConsumersByID:       make(map[string]*KongConsumer, 0),
		ConsumersByUsername: make(map[string]*KongConsumer, 0),
		ConsumersByCustomID: make(map[string]*KongConsumer, 0),
		Plugins:             make([]KongPlugin, 0),
		PluginsByID:         make(map[string]*KongPlugin, 0),
		Upstreams:           make([]KongUpstream, 0),
		UpstreamsByName:     make(map[string]*KongUpstream, 0),
		UpstreamsByID:       make(map[string]*KongUpstream, 0),
		Targets:             make([]KongTarget, 0),
		TargetsByID:         make(map[string]*KongTarget, 0),
	}
}

// MustParseDeckFile takes in raw parsed json/yaml and parses it into a deckfile
// structure. Will panic if parsing fails.
func MustParseDeckFile(parsedFile interface{}) *DeckFile {
	parsed, err := ParseDeckFile(parsedFile)
	if err != nil {
		log.Fatalf("unable to parse file: %v", err)
	}
	return parsed
}

// ParseDeckFile takes in raw parsed json/yaml and parses it into a deckfile
// structure.
func ParseDeckFile(parsedFile interface{}) (*DeckFile, error) {
	data, err := jsonbasics.ToObject(parsedFile)
	if err != nil {
		return nil, fmt.Errorf("expected a JSON object at the top-level")
	}

	// get the file version and check it
	majorVersion, minorVersion, err := parseFormatVersion(data)
	if err != nil {
		return nil, err
	}
	if majorVersion != 3 {
		return nil, fmt.Errorf("bad file-format, only major version 3 is supported, got; %d.%d", majorVersion, minorVersion)
	}

	d := NewDeckFile()
	d.Data = data

	if err := parseServices(d); err != nil {
		return nil, err
	}

	if err := parseConsumers(d); err != nil {
		return nil, err
	}

	if err := parseRoutes(d); err != nil {
		return nil, err
	}

	if err := parsePlugins(d); err != nil {
		return nil, err
	}

	if err := parseUpstreams(d); err != nil {
		return nil, err
	}

	if err := parseTargets(d); err != nil {
		return nil, err
	}

	return d, nil
}

// parseServices moves the 'services' JSON array in the raw file data into the
// deckfile structure as service structs.
func parseServices(deckfile *DeckFile) error {
	rawServices := deckfile.Data["services"]

	if rawServices == nil {
		return nil
	}

	servicesArray, err := jsonbasics.ToArray(rawServices)
	if err != nil {
		return errors.New("expected '.services' to be an array of objects")
	}

	for i, expectedService := range servicesArray {
		serviceObj, err := jsonbasics.ToObject(expectedService)
		if err != nil {
			return fmt.Errorf("entry '.services[%d]' is not an object", i)
		}
		var service *KongService
		if service, err = NewKongService(serviceObj, deckfile); err != nil {
			return fmt.Errorf("entry '.services[%d]' could not be added; %w", i, err)
		}

		// parse nested plugins
		if rawPlugins := serviceObj["plugins"]; rawPlugins != nil {
			pluginsArray, err := jsonbasics.ToArray(rawPlugins)
			if err != nil {
				return fmt.Errorf("expected '.services[%d].plugins' to be an array of objects", i)
			}

			if err := parsePluginArray(pluginsArray, deckfile, service.GetRef(), "", ""); err != nil {
				return err
			}

			delete(serviceObj, "plugins")
		}

		// parse nested routes
		if rawRoutes := serviceObj["routes"]; rawRoutes != nil {
			routesArray, err := jsonbasics.ToArray(rawRoutes)
			if err != nil {
				return fmt.Errorf("expected '.services[%d].routes' to be an array of objects", i)
			}

			if err := parseRouteArray(routesArray, deckfile, service.GetRef()); err != nil {
				return err
			}

			delete(serviceObj, "routes")
		}
	}

	delete(deckfile.Data, "services")
	return nil
}

// parseConsumers moves the 'consumers' JSON array in the raw file data into the
// deckfile structure as consumer structs.
func parseConsumers(deckfile *DeckFile) error {
	rawConsumers := deckfile.Data["consumers"]

	if rawConsumers == nil {
		return nil
	}

	consumersArray, err := jsonbasics.ToArray(rawConsumers)
	if err != nil {
		return errors.New("expected '.consumers' to be an array of objects")
	}

	for i, expectedConsumer := range consumersArray {
		consumerObj, err := jsonbasics.ToObject(expectedConsumer)
		if err != nil {
			return fmt.Errorf("entry '.consumers[%d]' is not an object", i)
		}
		var consumer *KongConsumer
		if consumer, err = NewKongConsumer(consumerObj, deckfile); err != nil {
			return fmt.Errorf("entry '.consumers[%d]' could not be added; %w", i, err)
		}

		// parse nested plugins
		if rawPlugins := consumerObj["plugins"]; rawPlugins != nil {
			pluginsArray, err := jsonbasics.ToArray(rawPlugins)
			if err != nil {
				return fmt.Errorf("expected '.consumers[%d].plugins' to be an array of objects", i)
			}

			if err := parsePluginArray(pluginsArray, deckfile, "", "", consumer.GetRef()); err != nil {
				return err
			}

			delete(consumerObj, "plugins")
		}
	}

	delete(deckfile.Data, "consumers")
	return nil
}

// parseRouteArray parses an array of raw data as routes. Whilst being able to
// set the parent Service object (use "" to NOT set the parent Service).
func parseRouteArray(routesArray []interface{}, deckfile *DeckFile, serviceRef string) error {
	if routesArray == nil {
		return nil
	}

	for i, expectedRoute := range routesArray {
		routeObj, err := jsonbasics.ToObject(expectedRoute)
		if err != nil {
			return fmt.Errorf("entry '.routes[%d]' is not an object", i)
		}

		if serviceRef != "" { // we have a service that is owner (this route was nested)
			routeObj["service"] = serviceRef
		}

		var route *KongRoute
		if route, err = NewKongRoute(routeObj, deckfile); err != nil {
			// TODO: add breadcrumbs to these error messages to pin-point to offending item
			return fmt.Errorf("entry '.routes[%d]' could not be added; %w", i, err)
		}

		// parse nested plugins
		if rawPlugins := routeObj["plugins"]; rawPlugins != nil {
			pluginsArray, err := jsonbasics.ToArray(rawPlugins)
			if err != nil {
				return fmt.Errorf("expected '.routes[%d].plugins' to be an array of objects", i)
			}

			if err := parsePluginArray(pluginsArray, deckfile, "", route.GetRef(), ""); err != nil {
				return err
			}

			delete(routeObj, "plugins")
		}
	}

	return nil
}

// parseRoutes moves the 'routes' JSON array in the raw file data into the
// deckfile structure as route structs.
func parseRoutes(deckfile *DeckFile) error {
	rawRoutes := deckfile.Data["routes"]

	if rawRoutes == nil {
		return nil
	}

	routesArray, err := jsonbasics.ToArray(rawRoutes)
	if err != nil {
		return errors.New("expected '.routes' to be an array of objects")
	}

	if err := parseRouteArray(routesArray, deckfile, ""); err != nil {
		return err
	}

	delete(deckfile.Data, "routes")
	return nil
}

// parsePluginArray parses an array of raw data as plugins. Whilst being able to
// set the parent Service/Route/Consumer object (use "" to NOT set a parent).
func parsePluginArray(
	pluginsArray []interface{}, deckfile *DeckFile,
	serviceRef string, routeRef string, consumerRef string,
) error {
	if pluginsArray == nil {
		return nil
	}

	for i, expectedPlugin := range pluginsArray {
		pluginObj, err := jsonbasics.ToObject(expectedPlugin)
		if err != nil {
			return fmt.Errorf("entry '.plugins[%d]' is not an object", i)
		}

		// if we have 1 reference, we add them all
		if serviceRef != "" || routeRef != "" || consumerRef != "" {
			pluginObj["service"] = serviceRef
			pluginObj["route"] = routeRef
			pluginObj["consumer"] = consumerRef
		}

		if _, err := NewKongPlugin(pluginObj, deckfile); err != nil {
			// TODO: add breadcrumbs to these error messages to pin-point to offending item
			return fmt.Errorf("entry '.plugins[%d]' could not be added; %w", i, err)
		}
	}

	return nil
}

// parsePlugins moves the 'plugins' JSON array in the raw file data into the
// deckfile structure as plugin structs.
func parsePlugins(deckfile *DeckFile) error {
	rawPlugins := deckfile.Data["plugins"]

	if rawPlugins == nil {
		return nil
	}

	pluginsArray, err := jsonbasics.ToArray(rawPlugins)
	if err != nil {
		return errors.New("expected '.plugins' to be an array of objects")
	}

	if err := parsePluginArray(pluginsArray, deckfile, "", "", ""); err != nil {
		return err
	}

	delete(deckfile.Data, "plugins")
	return nil
}

// parseUpstreams moves the 'upstreams' JSON array in the raw file data into the
// deckfile structure as upstream structs.
func parseUpstreams(deckfile *DeckFile) error {
	rawUpstreams := deckfile.Data["upstreams"]

	if rawUpstreams == nil {
		return nil
	}

	upstreamsArray, err := jsonbasics.ToArray(rawUpstreams)
	if err != nil {
		return errors.New("expected '.upstreams' to be an array of objects")
	}

	for i, expectedUpstream := range upstreamsArray {
		upstreamObj, err := jsonbasics.ToObject(expectedUpstream)
		if err != nil {
			return fmt.Errorf("entry '.upstreams[%d]' is not an object", i)
		}
		var upstream *KongUpstream
		if upstream, err = NewKongUpstream(upstreamObj, deckfile); err != nil {
			return fmt.Errorf("entry '.upstreams[%d]' could not be added; %w", i, err)
		}

		// parse nested targets
		if rawTargets := upstreamObj["routes"]; rawTargets != nil {
			targetsArray, err := jsonbasics.ToArray(rawTargets)
			if err != nil {
				return fmt.Errorf("expected '.upstreams[%d].targets' to be an array of objects", i)
			}

			if err := parseTargetArray(targetsArray, deckfile, upstream.GetRef()); err != nil {
				return err
			}

			delete(upstreamObj, "targets")
		}
	}

	delete(deckfile.Data, "upstreams")
	return nil
}

// parseTargetArray parses an array of raw data as targets. Whilst being able to
// set the parent Upstream object (use "" to NOT set a parent).
func parseTargetArray(targetsArray []interface{}, deckfile *DeckFile, upstreamRef string) error {
	if targetsArray == nil {
		return nil
	}

	for i, expectedTarget := range targetsArray {
		targetObj, err := jsonbasics.ToObject(expectedTarget)
		if err != nil {
			return fmt.Errorf("entry '.targets[%d]' is not an object", i)
		}

		if upstreamRef != "" { // if we have a parent, then set it
			targetObj["upstream"] = upstreamRef
		}

		if _, err := NewKongTarget(targetObj, deckfile); err != nil {
			// TODO: add breadcrumbs to these error messages to pin-point to offending item
			return fmt.Errorf("entry '.targets[%d]' could not be added; %w", i, err)
		}
	}

	return nil
}

// parseTargets moves the 'targets' JSON array in the raw file data into the
// deckfile structure as target structs.
func parseTargets(deckfile *DeckFile) error {
	rawTargets := deckfile.Data["targets"]

	if rawTargets == nil {
		return nil
	}

	targetsArray, err := jsonbasics.ToArray(rawTargets)
	if err != nil {
		return errors.New("expected '.targets' to be an array of objects")
	}

	if err := parseTargetArray(targetsArray, deckfile, ""); err != nil {
		return err
	}

	delete(deckfile.Data, "targets")
	return nil
}

//
//
//  Generic file properties
//
//

// parseFormatVersion parses field `_format_version` and returns major+minor.
func parseFormatVersion(data map[string]interface{}) (int, int, error) {
	// get the file version and check it
	v, err := jsonbasics.GetStringField(data, "_format_version")
	if err != nil {
		return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
	}
	elem := strings.Split(v, ".")
	if len(elem) > 2 {
		return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
	}

	majorVersion, err := strconv.Atoi(elem[1])
	if err != nil {
		return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
	}

	minorVersion := 0
	if len(elem) > 1 {
		minorVersion, err = strconv.Atoi(elem[2])
		if err != nil {
			return 0, 0, errors.New("expected field '._format_version' to be a string in 'x.y' format")
		}
	}

	return majorVersion, minorVersion, nil
}

//
//
//  Retrieve entities by reference
//
//

// GetConsumerByReference returns a consumer by id/username/custom_id, or nil if
// not found.
func (deckfile *DeckFile) GetConsumerByReference(ref string) *KongConsumer {
	// TODO: if the deck file has a plugin with field `consumer` set, what is the lookup-order/logic
	// that Kong uses? just an order? or will it test for uuid format?
	// For now; ID -> username -> customid
	if ref == "" {
		return nil
	}

	consumer := deckfile.ConsumersByID[ref]
	if consumer != nil {
		return consumer
	}

	consumer = deckfile.ConsumersByUsername[ref]
	if consumer != nil {
		return consumer
	}

	return deckfile.ConsumersByCustomID[ref]
}

// GetServiceByReference returns a service by id/name, or nil if
// not found.
func (deckfile *DeckFile) GetServiceByReference(ref string) *KongService {
	// TODO: if the deck file has a route/consumer with field `service` set, what is the lookup-order/logic
	// that Kong uses? just an order? or will it test for uuid format?
	// For now; ID -> name
	if ref == "" {
		return nil
	}

	service := deckfile.ServicesByID[ref]
	if service != nil {
		return service
	}

	return deckfile.ServicesByName[ref]
}

// GetRouteByReference returns a route by id/name, or nil if not found.
func (deckfile *DeckFile) GetRouteByReference(ref string) *KongRoute {
	// TODO: if the deck file has a plugin with field `route` set, what is the lookup-order/logic
	// that Kong uses? just an order? or will it test for uuid format?
	// For now; ID -> name
	if ref == "" {
		return nil
	}

	route := deckfile.RoutesByID[ref]
	if route != nil {
		return route
	}

	return deckfile.RoutesByName[ref]
}

// GetUpstreamByReference returns an upstream by id/name, or nil if
// not found.
func (deckfile *DeckFile) GetUpstreamByReference(ref string) *KongUpstream {
	// TODO: if the deck file has a route/consumer with field `service` set, what is the lookup-order/logic
	// that Kong uses? just an order? or will it test for uuid format?
	// For now; ID -> name
	if ref == "" {
		return nil
	}

	upstream := deckfile.UpstreamsByID[ref]
	if upstream != nil {
		return upstream
	}

	return deckfile.UpstreamsByName[ref]
}
