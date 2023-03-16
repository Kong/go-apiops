package deckfile

import (
	"github.com/kong/go-apiops/jsonbasics"
)

// GetUpstreamByReference returns an upstream by id/name, or nil if
// not found.
func (deckfile *DeckFile) export(flat bool) *map[string]interface{} {
	if deckfile.Data == nil {
		panic("deckfile hasn't been initialized yet, no data set")
	}
	data := *jsonbasics.DeepCopy(&deckfile.Data)

	if data["services"] != nil {
		panic("expected field 'services' to be unset")
	}
	services := make([]map[string]interface{}, 0)

	if data["routes"] != nil {
		panic("expected field 'routes' to be unset")
	}
	routes := make([]map[string]interface{}, 0)

	if data["plugins"] != nil {
		panic("expected field 'plugins' to be unset")
	}
	plugins := make([]map[string]interface{}, 0)

	if data["consumers"] != nil {
		panic("expected field 'consumers' to be unset")
	}
	consumers := make([]map[string]interface{}, 0)

	if data["upstreams"] != nil {
		panic("expected field 'upstreams' to be unset")
	}
	upstreams := make([]map[string]interface{}, 0)

	if data["targets"] != nil {
		panic("expected field 'targets' to be unset")
	}
	targets := make([]map[string]interface{}, 0)

	// Start with plugins, as they have no nested entities
	if flat {
		for
	} else {

	}

	data["services"] = services
	data["routes"] = routes
	data["plugins"] = plugins
	data["consumers"] = consumers
	data["upstreams"] = upstreams
	data["targets"] = targets
	return &data
}
