package deckfile

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
	uuid "github.com/satori/go.uuid"
)

type KongRoute struct {
	data        map[string]interface{} // the raw JSON data
	deckfile    *DeckFile              // deckfile it belongs to
	id          string                 // ID (uuid) of the route (empty if not provided)
	idTemporary bool                   // if truthy, then the ID was generated as a temporary one (random)
	name        string                 // name of the route (empty if not provided)
	serviceRef  string                 // service it is connected to (name or id)
}

func NewKongRoute(data map[string]interface{}, deckfile *DeckFile) (*KongRoute, error) {
	if deckfile == nil {
		panic("can't add a route without a deckfile")
	}
	if data == nil {
		panic("can't add a route without the route data")
	}

	// no error checking, we only want them as strings
	id, _ := jsonbasics.GetStringField(data, "id")
	name, _ := jsonbasics.GetStringField(data, "name")
	serviceRef, _ := jsonbasics.GetStringField(data, "service")

	route := KongRoute{
		data:        data,
		deckfile:    deckfile,
		id:          id,
		idTemporary: false,
		name:        name,
		serviceRef:  serviceRef,
	}

	if err := route.SetID(id); err != nil {
		return nil, err
	}

	if err := route.SetName(name); err != nil {
		return nil, err
	}

	route.SetServiceRef(serviceRef)
	deckfile.Routes = append(deckfile.Routes, route)
	return &route, nil
}

// GetID returns the id of the route (empty if it was not provided)
func (route *KongRoute) GetID() string {
	return route.id
}

// GetName returns the name of the route (empty if it was not provided)
func (route *KongRoute) GetName() string {
	return route.name
}

// GetServiceRef returns the service reference of the route (empty if it was not provided)
func (route *KongRoute) GetServiceRef() string {
	return route.serviceRef
}

// GetService returns the owner service object (can return nil if there is none)
func (route *KongRoute) GetService() *KongService {
	return route.deckfile.GetServiceByReference(route.serviceRef)
}

// GetRef returns a reference to the route. If none is available it will generate
// a temporary id. Precedence; the username, then id, then new temporary-id.
func (route *KongRoute) GetRef() string {
	if route.name != "" {
		return route.name
	}
	if route.id == "" {
		tempID := uuid.NewV4().String()
		_ = route.SetID(tempID)
		route.idTemporary = true
	}

	return route.id
}

// HasTemporaryID returns whether the ID is a temporary one.
func (route *KongRoute) HasTemporaryID() bool {
	return route.idTemporary
}

// SetID sets a new ID for a route and will update the reverse lookup maps
// accordingly. Will return an error if the new ID already exists!
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (route *KongRoute) SetID(newID string) error {
	oldID := route.id

	if newID == "" {
		// delete the ID
		route.id = ""
		route.idTemporary = false
		delete(route.data, "id")
		delete(route.deckfile.RoutesByID, oldID)
		return nil
	}

	if newID != oldID && route.deckfile.RoutesByID[newID] != nil {
		return fmt.Errorf("a route with id '%s' already exists", newID)
	}

	delete(route.deckfile.RoutesByID, oldID)
	route.deckfile.RoutesByID[newID] = route
	route.data["id"] = newID
	route.id = newID
	route.idTemporary = false

	return nil
}

// SetName sets a new name for a route and will update the reverse lookup maps
// accordingly. Will return an error if the new name already exists!
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (route *KongRoute) SetName(newName string) error {
	oldName := route.name

	if newName == "" {
		// delete the name
		route.name = ""
		delete(route.data, "name")
		delete(route.deckfile.RoutesByName, oldName)
		return nil
	}

	if newName != oldName && route.deckfile.RoutesByName[newName] != nil {
		return fmt.Errorf("a route with name '%s' already exists", newName)
	}

	delete(route.deckfile.RoutesByName, oldName)
	route.deckfile.RoutesByName[newName] = route
	route.data["name"] = newName
	route.name = newName

	return nil
}

// SetServiceRef sets a new reference to a service this route belongs to.
// Setting an empty string will remove it.
func (route *KongRoute) SetServiceRef(serviceRef string) {
	route.serviceRef = serviceRef
	if serviceRef == "" {
		delete(route.data, "service")
	} else {
		route.data["service"] = serviceRef
	}
}
