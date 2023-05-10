package deckformat

import (
	"fmt"

	"github.com/kong/go-apiops/jsonbasics"
)

// ConvertDBless converts a DBless format to a decK type format. This updates the
// consumer-groups related entities into nested objects, since that is an operation
// that is too complex to do via a CLI.
func ConvertDBless(data map[string]interface{}) (map[string]interface{}, error) {
	// Step 1.
	// Rename "consumer_groups[*].consumer_group_plugins" to "consumer_groups[*].plugins", in case
	// nested entries already exist.
	// Only 1 should exist, and it must be an array
	consumerGroups, err := jsonbasics.GetObjectArrayField(data, "consumer_groups")
	if err != nil {
		return nil, fmt.Errorf("failed to read 'consumer_groups'; %w", err)
	}

	for i, consumerGroup := range consumerGroups {
		consumerGroupsPlugins, err := jsonbasics.GetObjectArrayField(consumerGroup, "consumer_group_plugins")
		if err != nil {
			return nil, fmt.Errorf("failed to read 'consumer_groups[%d].consumer_group_plugins'; %w", i, err)
		}

		plugins, err := jsonbasics.GetObjectArrayField(consumerGroup, "plugins")
		if err != nil {
			return nil, fmt.Errorf("failed to read 'consumer_groups[%d].plugins'; %w", i, err)
		}

		if len(consumerGroupsPlugins) > 0 && len(plugins) > 0 {
			// both given, not allowed
			return nil, fmt.Errorf("entry 'consumer_groups[%d]' contains both 'consumer_group_plugins' and 'plugins'", i)
		}

		if len(plugins) == 0 && len(consumerGroupsPlugins) == 0 {
			// neither given, delete them
			jsonbasics.SetObjectArrayField(consumerGroup, "plugins", nil)
			jsonbasics.SetObjectArrayField(consumerGroup, "consumer_group_plugins", nil)

		} else if len(plugins) > 0 {
			// only plugins given
			jsonbasics.SetObjectArrayField(consumerGroup, "plugins", plugins)
			jsonbasics.SetObjectArrayField(consumerGroup, "consumer_group_plugins", nil)

		} else {
			// only consumer_group_plugins given
			jsonbasics.SetObjectArrayField(consumerGroup, "plugins", consumerGroupsPlugins)
			jsonbasics.SetObjectArrayField(consumerGroup, "consumer_group_plugins", nil)
		}
	}

	// Step 2.
	// Read top-level 'consumer_group_plugins' and insert them into the related 'consumer_groups' entities.
	consumerGroupsPlugins, err := jsonbasics.GetObjectArrayField(data, "consumer_group_plugins")
	if err != nil {
		return nil, fmt.Errorf("failed to read 'consumer_group_plugins'; %w", err)
	}
	delete(data, "consumer_group_plugins")

	for i, plugin := range consumerGroupsPlugins {
		groupName, err := jsonbasics.GetStringField(plugin, "consumer_group")
		if err != nil {
			return nil, fmt.Errorf("failed to read 'consumer_group_plugins[%d].consumer_group'; %w", i, err)
		}

		for j, consumerGroup := range consumerGroups {
			targetGroupName, err := jsonbasics.GetStringField(consumerGroup, "name")
			if err != nil {
				return nil, fmt.Errorf("failed to read 'consumer_groups[%d].name'; %w", j, err)
			}

			if groupName == targetGroupName {
				// found it, so insert it
				delete(plugin, "consumer_group")
				plugins, _ := jsonbasics.GetObjectArrayField(consumerGroup, "plugins") // no error checking needed, done above
				jsonbasics.SetObjectArrayField(consumerGroup, "plugins", append(plugins, plugin))
				// mark as done
				plugin = nil
				break
			}
		}
		if plugin != nil {
			return nil, fmt.Errorf("consumer_group '%s' referenced by 'consumer_group_plugins[%d]' not found", groupName, i)
		}
	}

	// Step 3.
	// Integrate 'consumer_group_consumers' into 'consumers'.
	// read group memberships
	consumerGroupsConsumers, err := jsonbasics.GetObjectArrayField(data, "consumer_group_consumers")
	if err != nil {
		return nil, fmt.Errorf("failed to read 'consumer_group_consumers'; %w", err)
	}
	delete(data, "consumer_group_consumers")

	// build consumer map by name AND id
	consumers, err := jsonbasics.GetObjectArrayField(data, "consumers")
	if err != nil {
		return nil, fmt.Errorf("failed to read 'consumers'; %w", err)
	}
	consumerMap := make(map[string]map[string]interface{})
	for _, consumer := range consumers {
		name, _ := jsonbasics.GetStringField(consumer, "username")
		id, _ := jsonbasics.GetStringField(consumer, "id")
		consumerMap[name] = consumer
		consumerMap[id] = consumer
	}
	// safety, delete empty strings
	delete(consumerMap, "")

	for i, consumerGroupMap := range consumerGroupsConsumers {
		consumer, err := jsonbasics.GetStringField(consumerGroupMap, "consumer")
		if err != nil {
			return nil, fmt.Errorf("failed to read 'consumer_group_consumers[%d].consumer'; %w", i, err)
		}
		consumerGroup, err := jsonbasics.GetStringField(consumerGroupMap, "consumer_group")
		if err != nil {
			return nil, fmt.Errorf("failed to read 'consumer_group_consumers[%d].consumer_group'; %w", i, err)
		}

		consumerObj := consumerMap[consumer]
		if consumerObj == nil {
			return nil, fmt.Errorf("consumer '%s' assigned to consumer_group '%s' not found", consumer, consumerGroup)
		}

		groups, err := jsonbasics.GetObjectArrayField(consumerObj, "groups")
		if err != nil {
			return nil, fmt.Errorf("failed to read 'groups' from consumer '%s'; %w", consumer, err)
		}
		// huh??? why an object with a single key, in an array???
		entry := make(map[string]interface{})
		entry["name"] = consumerGroup
		groups = append(groups, entry)
		jsonbasics.SetObjectArrayField(consumerObj, "groups", groups)
	}

	return data, nil
}
