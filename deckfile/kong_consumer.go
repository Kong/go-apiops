package deckfile

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
	uuid "github.com/satori/go.uuid"
)

type KongConsumer struct {
	data        map[string]interface{} // the raw JSON data
	deckfile    *DeckFile              // deckfile it belongs to
	id          string                 // ID (uuid) of the consumer (empty if not provided)
	idTemporary bool                   // if truthy, then the ID was generated as a temporary one (random)
	username    string                 // username of the consumer (empty if not provided)
	customID    string                 // custom_id of the consumer (empty if not provided)
}

func NewKongConsumer(data map[string]interface{}, deckfile *DeckFile) (*KongConsumer, error) {
	if deckfile == nil {
		panic("can't add a consumer without a deckfile")
	}
	if data == nil {
		panic("can't add a consumer without the consumer data")
	}

	// no error checking, we only want them as strings
	id, _ := jsonbasics.GetStringField(data, "id")
	username, _ := jsonbasics.GetStringField(data, "username")
	customid, _ := jsonbasics.GetStringField(data, "custom_id")

	consumer := KongConsumer{
		data:        data,
		deckfile:    deckfile,
		id:          id,
		idTemporary: false,
		username:    username,
		customID:    customid,
	}

	if err := consumer.SetID(id); err != nil {
		return nil, err
	}

	if err := consumer.SetUsername(username); err != nil {
		return nil, err
	}

	if err := consumer.SetCustomID(customid); err != nil {
		return nil, err
	}

	deckfile.Consumers = append(deckfile.Consumers, consumer)
	return &consumer, nil
}

// GetID returns the id of the consumer (empty if it was not provided)
func (consumer *KongConsumer) GetID() string {
	return consumer.id
}

// GetUsername returns the name of the consumer (empty if it was not provided)
func (consumer *KongConsumer) GetUsername() string {
	return consumer.username
}

// GetCustomID returns the custom-id of the consumer (empty if it was not provided)
func (consumer *KongConsumer) GetCustomID() string {
	return consumer.customID
}

// GetRef returns a reference to the consumer. If none is available it will generate
// a temporary id. Precedence; the username, then id, then new temporary-id.
func (consumer *KongConsumer) GetRef() string {
	if consumer.username != "" {
		return consumer.username
	}
	if consumer.id == "" {
		tempID := uuid.NewV4().String()
		_ = consumer.SetID(tempID)
		consumer.idTemporary = true
	}

	return consumer.id
}

// HasTemporaryID returns whether the ID is a temporary one.
func (consumer *KongConsumer) HasTemporaryID() bool {
	return consumer.idTemporary
}

// SetID sets a new ID for a consumer and will update the reverse lookup maps
// accordingly. Will return an error if the new ID already exists!
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (consumer *KongConsumer) SetID(newID string) error {
	oldID := consumer.id

	if newID == "" {
		// delete the ID
		consumer.id = ""
		consumer.idTemporary = false
		delete(consumer.data, "id")
		delete(consumer.deckfile.ConsumersByID, oldID)
		return nil
	}

	if newID != oldID && consumer.deckfile.ConsumersByID[newID] != nil {
		return fmt.Errorf("a consumer with id '%s' already exists", newID)
	}

	delete(consumer.deckfile.ConsumersByID, oldID)
	consumer.deckfile.ConsumersByID[newID] = consumer
	consumer.data["id"] = newID
	consumer.id = newID
	consumer.idTemporary = false

	return nil
}

// SetUsername sets a new ID for a consumer and will update the reverse lookup maps
// accordingly. Will return an error if the new username already exists!
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (consumer *KongConsumer) SetUsername(newUsername string) error {
	oldUsername := consumer.username

	if newUsername == "" {
		// delete the username
		consumer.username = ""
		delete(consumer.data, "username")
		delete(consumer.deckfile.ConsumersByUsername, oldUsername)
		return nil
	}

	if newUsername != oldUsername && consumer.deckfile.ConsumersByUsername[newUsername] != nil {
		return fmt.Errorf("a consumer with username '%s' already exists", newUsername)
	}

	delete(consumer.deckfile.ConsumersByUsername, oldUsername)
	consumer.deckfile.ConsumersByUsername[newUsername] = consumer
	consumer.data["username"] = newUsername
	consumer.username = newUsername

	return nil
}

// SetCustomID sets a new ID for a consumer and will update the reverse lookup maps
// accordingly. Will return an error if the new custom-id already exists!
// Setting an empty string will remove it from the lookup maps, but not from the file.
func (consumer *KongConsumer) SetCustomID(newCustomID string) error {
	oldCustomID := consumer.customID

	if newCustomID == "" {
		// delete the custom-id
		consumer.customID = ""
		delete(consumer.data, "custom_id")
		delete(consumer.deckfile.ConsumersByCustomID, oldCustomID)
		return nil
	}

	if newCustomID != oldCustomID && consumer.deckfile.ConsumersByCustomID[newCustomID] != nil {
		return fmt.Errorf("a consumer with custom-id '%s' already exists", newCustomID)
	}

	delete(consumer.deckfile.ConsumersByCustomID, oldCustomID)
	consumer.deckfile.ConsumersByCustomID[newCustomID] = consumer
	consumer.data["custom_id"] = newCustomID
	consumer.customID = newCustomID

	return nil
}
