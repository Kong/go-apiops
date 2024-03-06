package namespace

import (
	"errors"
	"fmt"

	"github.com/kong/go-apiops/logbasics"
	"github.com/kong/go-apiops/yamlbasics"
	"gopkg.in/yaml.v3"
)

// ApplyNamespaceHost applies the namespace to the hosts field of the selected routes
// by adding the listed hosts if they ar not in the list already.
func ApplyNamespaceHost(
	deckfile *yaml.Node, // the deckFile to operate on
	selectors yamlbasics.SelectorSet, // the selectors to use to select the routes
	hosts []string, // the hosts to add to the routes
	clear bool, // if true, clear the hosts field before adding the hosts
	allowEmptySelection bool, // if true, do not return an error if no routes are selected
) error {
	if deckfile == nil {
		panic("expected 'deckfile' to be non-nil")
	}

	allRoutes := getAllRoutes(deckfile)
	var targetRoutes yamlbasics.NodeSet
	var err error
	if selectors.IsEmpty() {
		// no selectors, apply to all routes
		targetRoutes = make(yamlbasics.NodeSet, len(allRoutes))
		copy(targetRoutes, allRoutes)
	} else {
		targetRoutes, err = selectors.Find(deckfile)
		if err != nil {
			return err
		}
	}

	var remainder yamlbasics.NodeSet
	targetRoutes, remainder = allRoutes.Intersection(targetRoutes) // check for non-routes
	if len(remainder) != 0 {
		return fmt.Errorf("the selectors returned non-route entities; %d", len(remainder))
	}
	if len(targetRoutes) == 0 {
		if allowEmptySelection {
			logbasics.Info("no routes matched the selectors, nothing to do")
			return nil
		}
		return errors.New("no routes matched the selectors")
	}

	return updateRouteHosts(targetRoutes, hosts, clear)
}

// updateRouteHosts updates the hosts field of the provided routes.
// If clear is true, the hosts field is cleared before adding the hosts.
func updateRouteHosts(routes yamlbasics.NodeSet, hosts []string, clear bool) error {
	for _, route := range routes {
		if err := yamlbasics.CheckType(route, yamlbasics.TypeObject); err != nil {
			logbasics.Info("ignoring route: " + err.Error())
			continue
		}

		hostsValueNode := yamlbasics.GetFieldValue(route, "hosts")
		if hostsValueNode == nil {
			// the 'hosts' array doesn't exist
			if len(hosts) == 0 {
				// nothing to do since we're not adding anything
				continue
			}
			// create an empty 'hosts' array, so we can add to it
			hostsValueNode = yamlbasics.NewArray()
			yamlbasics.SetFieldValue(route, "hosts", hostsValueNode)
		} else {
			// the 'hosts' array exists, check the type
			if err := yamlbasics.CheckType(hostsValueNode, yamlbasics.TypeArray); err != nil {
				logbasics.Info("ignoring route.hosts property: " + err.Error())
				continue
			}
		}

		if clear && len(hostsValueNode.Content) > 0 {
			hostsValueNode.Content = make([]*yaml.Node, 0)
		}

		if len(hosts) > 0 {
			appendHosts(hostsValueNode, hosts)
		}
	}

	return nil
}

// appendHosts appends the provided hosts to the hosts array, without duplicates.
func appendHosts(hostsValueNode *yaml.Node, hosts []string) {
	if hostsValueNode == nil || hostsValueNode.Kind != yaml.SequenceNode {
		panic("expected 'hostsValueNode' to be a sequence node")
	}
	if len(hosts) == 0 {
		panic("expected 'hosts' to be non-nil and non-empty")
	}

	for _, hostname := range hosts {
		exists := false
		for _, hostNameNode := range hostsValueNode.Content {
			if hostNameNode.Value == hostname {
				// already exists, skip
				exists = true
				break
			}
		}
		if !exists {
			// add the hostname to the array
			hostsValueNode.Content = append(hostsValueNode.Content, yamlbasics.NewString(hostname))
		}
	}
}
