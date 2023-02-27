package deckfile

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
	uuid "github.com/satori/go.uuid"
)

type KongService struct {
	data        map[string]interface{} // the raw JSON data
	deckfile    *DeckFile              // deckfile it belongs to
	id          string                 // ID (uuid) of the service (empty if not provided)
	idTemporary bool                   // if truthy, then the ID was generated as a temporary one (random)
	name        string                 // name of the service (empty if not provided)
}

func NewKongService(data map[string]interface{}, deckfile *DeckFile) (*KongService, error) {
	if deckfile == nil {
		panic("can't add a service without a deckfile")
	}
	if data == nil {
		panic("can't add a service without the service data")
	}

	// no error checking, we only want them as strings
	id, _ := jsonbasics.GetStringField(data, "id")
	name, _ := jsonbasics.GetStringField(data, "name")

	service := KongService{
		data:        data,
		deckfile:    deckfile,
		id:          id,
		idTemporary: false,
		name:        name,
	}

	if err := service.SetID(id); err != nil {
		return nil, err
	}

	if err := service.SetName(name); err != nil {
		return nil, err
	}

	deckfile.Services = append(deckfile.Services, service)
	return &service, nil
}

// GetID returns the id of the service (empty if it was not provided)
func (service *KongService) GetID() string {
	return service.id
}

// GetName returns the name of the service (empty if it was not provided)
func (service *KongService) GetName() string {
	return service.name
}

// GetRef returns a reference to the service. If none is available it will generate
// a temporary id. Precedence; the name, then id, then new temporary-id.
func (service *KongService) GetRef() string {
	if service.name != "" {
		return service.name
	}
	if service.id == "" {
		tempID := uuid.NewV4().String()
		_ = service.SetID(tempID)
		service.idTemporary = true
	}

	return service.id
}

// HasTemporaryID returns whether the ID is a temporary one.
func (service *KongService) HasTemporaryID() bool {
	return service.idTemporary
}

// SetID sets a new ID for a service and will update the reverse lookup maps
// accordingly. Returns an error if it already exists.
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (service *KongService) SetID(newID string) error {
	oldID := service.id

	if newID == "" {
		// delete the ID
		service.id = ""
		service.idTemporary = false
		delete(service.data, "id")
		delete(service.deckfile.ServicesByID, oldID)
		return nil
	}

	if newID != oldID && service.deckfile.ServicesByID[newID] != nil {
		return fmt.Errorf("a service with id '%s' already exists", newID)
	}

	delete(service.deckfile.ServicesByID, oldID)
	service.deckfile.ServicesByID[newID] = service
	service.data["id"] = newID
	service.id = newID
	service.idTemporary = false

	return nil
}

// SetName sets a new name for a service and will update the reverse lookup maps
// accordingly. Returns an error if it already exists.
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (service *KongService) SetName(newName string) error {
	oldName := service.name

	if newName == "" {
		// delete the name
		service.name = ""
		delete(service.data, "name")
		delete(service.deckfile.ServicesByName, oldName)
		return nil
	}

	if newName != oldName && service.deckfile.ServicesByName[newName] != nil {
		return fmt.Errorf("a service with name '%s' already exists", newName)
	}

	delete(service.deckfile.ServicesByName, oldName)
	service.deckfile.ServicesByName[newName] = service
	service.data["name"] = newName
	service.name = newName

	return nil
}
