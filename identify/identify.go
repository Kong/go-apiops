package identify

import (
	"fmt"
	"log"
	"sort"

	"github.com/kong/go-apiops/jsonbasics"
	"sigs.k8s.io/yaml"
)

type foreignRelations struct {
	TableName        string           // the table-name or entity-array-name
	ForeignRelations []targetRelation // array of references to other entities
}

type targetRelation struct {
	ForeignTable string // the foreign table that is being referenced, eg; "services"
	LocalField   string // the local fieldname that holds the reference to the foreign entity, eg; "service"
}

// if map-key is "plugins" then entries are "services, "consumers", "routes", etc
// eg. map["plugins"]map["services"]"service" and map["plugins"]map["routes"]"route"
var fromToReferences map[string]map[string]string

// if map-key is "services" then entries are "consumers", "routes", etc
// eg. map["services"]map["plugins"]"service" and map["services"]map["routes"]"route"
var toFromReferences map[string]map[string]string

// EntityTables is a lookup map, to see if an array with entities by that name exists
var EntityTables map[string]bool

// TablesSortedByForeignRefCount is an array with tables names, sorted by least foreign
// refs ("consumers" goes first, "plugins" go last)
var TablesSortedByForeignRefCount []string

// InjectUUID will add `id` fields for all entities
func InjectUUID(dataIn interface{}) {
	data, err := jsonbasics.ToObject(dataIn)
	if err != nil {
		panic(fmt.Errorf("expected data to contain a JSON-object; %w", err))
	}

	parseEntityStructure()

	// walk all entires we know, starting with the ones having least foreign relationships
	for _, arrayName := range TablesSortedByForeignRefCount {
		breadcrumb := "." + arrayName

		list := data[arrayName]
		if list == nil {
			continue
		}
		listArr, err := jsonbasics.ToArray(list)
		if err != nil {
			log.Default().Print(fmt.Errorf("expected '"+breadcrumb+"' to be a JSON array; %w", err))
			continue
		}

		handleEntityArray(listArr, breadcrumb)
	}
}

func handleEntityArray(entityArray []interface{}, breadcrumb string) {

}

// parseEntityStructure will parse the generated file with foreign-key relationships
// and populate fromToReferences, toFromReferences, and EntityTables
func parseEntityStructure() {
	if fromToReferences != nil {
		return
	}

	var EntityStructure []foreignRelations
	err := yaml.Unmarshal([]byte(strEntityStructure), &EntityStructure)
	if err != nil {
		log.Fatal("failed deserializing data as YAML (%w)", err)
	}
	EntityTables = make(map[string]bool)
	toFromReferences = make(map[string]map[string]string)
	fromToReferences = make(map[string]map[string]string)
	for _, tableRelation := range EntityStructure {
		sourceTableName := tableRelation.TableName
		EntityTables[sourceTableName] = true

		for _, relation := range tableRelation.ForeignRelations {
			targetTableName := relation.ForeignTable
			localFieldName := relation.LocalField

			if toFromReferences[targetTableName] == nil {
				toFromReferences[targetTableName] = make(map[string]string)
			}
			toFromReferences[targetTableName][sourceTableName] = localFieldName

			if fromToReferences[sourceTableName] == nil {
				fromToReferences[sourceTableName] = make(map[string]string)
			}
			fromToReferences[sourceTableName][targetTableName] = localFieldName
		}
	}

	// create array sorted by least referenced
	type refCount struct {
		tableName string
		count     int
	}
	list := make([]refCount, len(EntityTables))
	i := 0
	for tableName := range EntityTables {
		list[i] = refCount{
			tableName: tableName,
			count:     len(toFromReferences[tableName]),
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].count < list[j].count
	})

	TablesSortedByForeignRefCount = make([]string, len(list))
	for i, entry := range list {
		TablesSortedByForeignRefCount[i] = entry.tableName
	}

}
