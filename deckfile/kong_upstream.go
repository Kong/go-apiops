package deckfile

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
	uuid "github.com/satori/go.uuid"
)

type KongUpstream struct {
	data        map[string]interface{} // the raw JSON data
	deckfile    *DeckFile              // deckfile it belongs to
	id          string                 // ID (uuid) of the upstream (empty if not provided)
	idTemporary bool                   // if truthy, then the ID was generated as a temporary one (random)
	name        string                 // name of the upstream (empty if not provided)
}

func NewKongUpstream(data map[string]interface{}, deckfile *DeckFile) (*KongUpstream, error) {
	if deckfile == nil {
		panic("can't add an upstream without a deckfile")
	}
	if data == nil {
		panic("can't add an upstream without the upstream data")
	}

	// no error checking, we only want them as strings
	id, _ := jsonbasics.GetStringField(data, "id")
	name, _ := jsonbasics.GetStringField(data, "name")

	upstream := KongUpstream{
		data:        data,
		deckfile:    deckfile,
		id:          id,
		idTemporary: false,
		name:        name,
	}

	if err := upstream.SetID(id); err != nil {
		return nil, err
	}

	if err := upstream.SetName(name); err != nil {
		return nil, err
	}

	deckfile.Upstreams = append(deckfile.Upstreams, upstream)
	return &upstream, nil
}

// GetID returns the id of the upstream (empty if it was not provided)
func (upstream *KongUpstream) GetID() string {
	return upstream.id
}

// GetName returns the name of the upstream (empty if it was not provided)
func (upstream *KongUpstream) GetName() string {
	return upstream.name
}

// GetRef returns a reference to the upstream. If none is available it will generate
// a temporary id. Precedence; the name, then id, then new temporary-id.
func (upstream *KongUpstream) GetRef() string {
	if upstream.name != "" {
		return upstream.name
	}
	if upstream.id == "" {
		tempID := uuid.NewV4().String()
		_ = upstream.SetID(tempID)
		upstream.idTemporary = true
	}

	return upstream.id
}

// HasTemporaryID returns whether the ID is a temporary one.
func (upstream *KongUpstream) HasTemporaryID() bool {
	return upstream.idTemporary
}

// SetID sets a new ID for an upstream and will update the reverse lookup maps
// accordingly. Returns an error if it already exists.
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (upstream *KongUpstream) SetID(newID string) error {
	oldID := upstream.id

	if newID == "" {
		// delete the ID
		upstream.id = ""
		upstream.idTemporary = false
		delete(upstream.data, "id")
		delete(upstream.deckfile.UpstreamsByID, oldID)
		return nil
	}

	if newID != oldID && upstream.deckfile.UpstreamsByID[newID] != nil {
		return fmt.Errorf("an upstream with id '%s' already exists", newID)
	}

	delete(upstream.deckfile.UpstreamsByID, oldID)
	upstream.deckfile.UpstreamsByID[newID] = upstream
	upstream.data["id"] = newID
	upstream.id = newID
	upstream.idTemporary = false

	return nil
}

// SetName sets a new name for an upstream and will update the reverse lookup maps
// accordingly. Returns an error if it already exists.
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (upstream *KongUpstream) SetName(newName string) error {
	oldName := upstream.name

	if newName == "" {
		// delete the name
		upstream.name = ""
		delete(upstream.data, "name")
		delete(upstream.deckfile.UpstreamsByName, oldName)
		return nil
	}

	if newName != oldName && upstream.deckfile.UpstreamsByName[newName] != nil {
		return fmt.Errorf("an upstream with name '%s' already exists", newName)
	}

	delete(upstream.deckfile.UpstreamsByName, oldName)
	upstream.deckfile.UpstreamsByName[newName] = upstream
	upstream.data["name"] = newName
	upstream.name = newName

	return nil
}
