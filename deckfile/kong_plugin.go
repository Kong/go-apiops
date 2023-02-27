package deckfile

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
)

type KongPlugin struct {
	data        map[string]interface{} // the raw JSON data
	deckfile    *DeckFile              // deckfile it belongs to
	id          string                 // ID (uuid) of the route (empty if not provided)
	idTemporary bool                   // if truthy, then the ID was generated as a temporary one (random)
	serviceRef  string                 // service it is connected to (name or id)
	routeRef    string                 // route it is connected to (name or id)
	consumerRef string                 // consumer it is connected to (username or id)
}

func NewKongPlugin(data map[string]interface{}, deckfile *DeckFile) (*KongPlugin, error) {
	if deckfile == nil {
		panic("can't add a consumer without a deckfile")
	}
	if data == nil {
		panic("can't add a plugin without the plugin data")
	}

	// no error checking, we only want them as strings
	id, _ := jsonbasics.GetStringField(data, "id")
	serviceRef, _ := jsonbasics.GetStringField(data, "service")
	routeRef, _ := jsonbasics.GetStringField(data, "route")
	consumerRef, _ := jsonbasics.GetStringField(data, "consumer")

	plugin := KongPlugin{
		data:        data,
		deckfile:    deckfile,
		id:          id,
		idTemporary: false,
		serviceRef:  serviceRef,
		routeRef:    routeRef,
		consumerRef: consumerRef,
	}

	if err := plugin.SetID(id); err != nil {
		return nil, err
	}

	plugin.SetServiceRef(serviceRef)
	plugin.SetRouteRef(routeRef)
	plugin.SetConsumerRef(consumerRef)
	deckfile.Plugins = append(deckfile.Plugins, plugin)
	return &plugin, nil
}

// GetID returns the id of the plugin (empty if it was not provided)
func (plugin *KongPlugin) GetID() string {
	return plugin.id
}

// GetServiceRef returns the service reference of the plugin (empty if it was not provided)
func (plugin *KongPlugin) GetServiceRef() string {
	return plugin.serviceRef
}

// GetService returns the owner service object (can return nil if there is none)
func (plugin *KongPlugin) GetService() *KongService {
	return plugin.deckfile.GetServiceByReference(plugin.serviceRef)
}

// GetRouteRef returns the route reference of the plugin (empty if it was not provided)
func (plugin *KongPlugin) GetRouteRef() string {
	return plugin.routeRef
}

// GetRoute returns the owner route object (can return nil if there is none)
func (plugin *KongPlugin) GetRoute() *KongRoute {
	return plugin.deckfile.GetRouteByReference(plugin.routeRef)
}

// GetConsumerRef returns the consumer reference of the plugin (empty if it was not provided)
func (plugin *KongPlugin) GetConsumerRef() string {
	return plugin.consumerRef
}

// GetConsumer returns the owner consumer object (can return nil if there is none)
func (plugin *KongPlugin) GetConsumer() *KongConsumer {
	return plugin.deckfile.GetConsumerByReference(plugin.consumerRef)
}

// SetID sets a new ID for a plugin and will update the reverse lookup maps
// accordingly. Will return an error if the new ID already exists!
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (plugin *KongPlugin) SetID(newID string) error {
	oldID := plugin.id

	if newID == "" {
		// delete the ID
		plugin.id = ""
		plugin.idTemporary = false
		delete(plugin.data, "id")
		delete(plugin.deckfile.PluginsByID, oldID)
		return nil
	}

	if newID != oldID && plugin.deckfile.PluginsByID[newID] != nil {
		return fmt.Errorf("a plugin with id '%s' already exists", newID)
	}

	delete(plugin.deckfile.PluginsByID, oldID)
	plugin.deckfile.PluginsByID[newID] = plugin
	plugin.data["id"] = newID
	plugin.id = newID
	plugin.idTemporary = false

	return nil
}

// SetServiceRef sets a new reference to a service this plugin belongs to.
// Setting an empty string will remove it.
func (plugin *KongPlugin) SetServiceRef(serviceRef string) {
	plugin.serviceRef = serviceRef
	if serviceRef == "" {
		delete(plugin.data, "service")
	} else {
		plugin.data["service"] = serviceRef
	}
}

// SetRouteRef sets a new reference to a route this plugin belongs to.
// Setting an empty string will remove it.
func (plugin *KongPlugin) SetRouteRef(routeRef string) {
	plugin.routeRef = routeRef
	if routeRef == "" {
		delete(plugin.data, "route")
	} else {
		plugin.data["route"] = routeRef
	}
}

// SetConsumerRef sets a new reference to a consumer this plugin belongs to.
// Setting an empty string will remove it.
func (plugin *KongPlugin) SetConsumerRef(consumerRef string) {
	plugin.consumerRef = consumerRef
	if consumerRef == "" {
		delete(plugin.data, "consumer")
	} else {
		plugin.data["consumer"] = consumerRef
	}
}
