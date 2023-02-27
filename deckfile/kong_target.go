package deckfile

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
)

type KongTarget struct {
	data        map[string]interface{} // the raw JSON data
	deckfile    *DeckFile              // deckfile it belongs to
	id          string                 // ID (uuid) of the target (empty if not provided)
	idTemporary bool                   // if truthy, then the ID was generated as a temporary one (random)
	upstreamRef string                 // upstream it is connected to (name or id)
}

func NewKongTarget(data map[string]interface{}, deckfile *DeckFile) (*KongTarget, error) {
	if deckfile == nil {
		panic("can't add a target without a deckfile")
	}
	if data == nil {
		panic("can't add a target without the target data")
	}

	// no error checking, we only want them as strings
	id, _ := jsonbasics.GetStringField(data, "id")
	upstreamRef, _ := jsonbasics.GetStringField(data, "upstream")

	target := KongTarget{
		data:        data,
		deckfile:    deckfile,
		id:          id,
		idTemporary: false,
		upstreamRef: upstreamRef,
	}

	if err := target.SetID(id); err != nil {
		return nil, err
	}

	target.SetUpstreamRef(upstreamRef)
	deckfile.Targets = append(deckfile.Targets, target)
	return &target, nil
}

// GetID returns the id of the target (empty if it was not provided)
func (target *KongTarget) GetID() string {
	return target.id
}

// GetUpstreamRef returns the upstream reference of the target (empty if it was not provided)
func (target *KongTarget) GetUpstreamRef() string {
	return target.upstreamRef
}

// GetUpstream returns the owner upstream object (can return nil if there is none)
func (target *KongTarget) GetUpstream() *KongUpstream {
	return target.deckfile.GetUpstreamByReference(target.upstreamRef)
}

// HasTemporaryID returns whether the ID is a temporary one.
func (target *KongTarget) HasTemporaryID() bool {
	return target.idTemporary
}

// SetID sets a new ID for a target and will update the reverse lookup maps
// accordingly. Will return an error if the new ID already exists!
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (target *KongTarget) SetID(newID string) error {
	oldID := target.id

	if newID == "" {
		// delete the ID
		target.id = ""
		target.idTemporary = false
		delete(target.data, "id")
		delete(target.deckfile.TargetsByID, oldID)
		return nil
	}

	if newID != oldID && target.deckfile.TargetsByID[newID] != nil {
		return fmt.Errorf("a target with id '%s' already exists", newID)
	}

	delete(target.deckfile.TargetsByID, oldID)
	target.deckfile.TargetsByID[newID] = target
	target.data["id"] = newID
	target.id = newID
	target.idTemporary = false

	return nil
}

// SetUpstreamRef sets a new reference to an upstream this target belongs to.
// Setting an empty string will remove it.
func (target *KongTarget) SetUpstreamRef(upstreamRef string) {
	target.upstreamRef = upstreamRef
	if upstreamRef == "" {
		delete(target.data, "upstream")
	} else {
		target.data["upstream"] = upstreamRef
	}
}
