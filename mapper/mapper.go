// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package mapper

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/graphql-go/graphql/language/ast"

	"github.com/sourcenetwork/defradb/client"
	"github.com/sourcenetwork/defradb/connor"
	"github.com/sourcenetwork/defradb/core"
	"github.com/sourcenetwork/defradb/core/enumerable"
	"github.com/sourcenetwork/defradb/datastore"
	"github.com/sourcenetwork/defradb/errors"
	"github.com/sourcenetwork/defradb/query/graphql/parser"
	parserTypes "github.com/sourcenetwork/defradb/query/graphql/parser/types"
)

// ToSelect converts the given [parser.Select] into a [Select].
//
// In the process of doing so it will construct the document map required to access the data
// yielded by the [Select].
func ToSelect(ctx context.Context, txn datastore.Txn, parsed *parser.Select) (*Select, error) {
	descriptionsRepo := NewDescriptionsRepo(ctx, txn)
	// the top-level select will always have index=0, and no parent collection name
	return toSelect(descriptionsRepo, 0, parsed, "")
}

// toSelect converts the given [parser.Select] into a [Select].
//
// In the process of doing so it will construct the document map required to access the data
// yielded by the [Select].
func toSelect(
	descriptionsRepo *DescriptionsRepo,
	thisIndex int,
	parsed *parser.Select,
	parentCollectionName string,
) (*Select, error) {
	collectionName, err := getCollectionName(descriptionsRepo, parsed, parentCollectionName)
	if err != nil {
		return nil, err
	}

	mapping, desc, err := getTopLevelInfo(descriptionsRepo, parsed, collectionName)
	if err != nil {
		return nil, err
	}

	fields, aggregates, err := getRequestables(parsed, mapping, desc, descriptionsRepo)
	if err != nil {
		return nil, err
	}

	// Needs to be done before resolving aggregates, else filter conversion may fail there
	filterDependencies, err := resolveFilterDependencies(descriptionsRepo, collectionName, parsed.Filter, mapping, fields)
	if err != nil {
		return nil, err
	}
	fields = append(fields, filterDependencies...)

	// Resolve order dependencies that may have been missed due to not being rendered.
	if err := resolveOrderDependencies(descriptionsRepo, collectionName, parsed.OrderBy, mapping, &fields); err != nil {
		return nil, err
	}

	aggregates = appendUnderlyingAggregates(aggregates, mapping)
	fields, err = resolveAggregates(
		parsed,
		aggregates,
		fields,
		mapping,
		desc,
		descriptionsRepo,
	)
	if err != nil {
		return nil, err
	}

	// If there is a groupby, and no inner group has been requested, we need to map the property here
	if parsed.GroupBy != nil {
		if _, isGroupFieldMapped := mapping.IndexesByName[parserTypes.GroupFieldName]; !isGroupFieldMapped {
			index := mapping.GetNextIndex()
			mapping.Add(index, parserTypes.GroupFieldName)
		}
	}

	return &Select{
		Targetable:      toTargetable(thisIndex, parsed, mapping),
		DocumentMapping: *mapping,
		Cid:             parsed.CID,
		CollectionName:  collectionName,
		Fields:          fields,
	}, nil
}

// resolveOrderDependencies will map fields that were missed due to them not being requested.
// Modifies the consumed existingFields and mapping accordingly.
func resolveOrderDependencies(
	descriptionsRepo *DescriptionsRepo,
	descName string,
	source *parserTypes.OrderBy,
	mapping *core.DocumentMapping,
	existingFields *[]Requestable,
) error {
	if source == nil {
		return nil
	}

	// If there is orderby, and any one of the condition fields that are join fields and have not been
	// requested, we need to map them here.
	for _, condition := range source.Conditions {
		fieldNames := strings.Split(condition.Field, ".")
		if len(fieldNames) <= 1 {
			continue
		}

		joinField := fieldNames[0]

		// Check if the join field is already mapped, if not then map it.
		if isOrderJoinFieldMapped := len(mapping.IndexesByName[joinField]) != 0; !isOrderJoinFieldMapped {
			index := mapping.GetNextIndex()
			mapping.Add(index, joinField)

			// Resolve the inner child fields and get it's mapping.
			dummyJoinFieldSelect := parser.Select{Name: joinField}
			innerSelect, err := toSelect(descriptionsRepo, index, &dummyJoinFieldSelect, descName)
			if err != nil {
				return err
			}
			*existingFields = append(*existingFields, innerSelect)
			mapping.SetChildAt(index, &innerSelect.DocumentMapping)
		}
	}

	return nil
}

// resolveAggregates figures out which fields the given aggregates are targeting
// and converts the aggregateRequest into an Aggregate, appending it onto the given
// fields slice.
//
// If an aggregate targets a field that doesn't yet exist, it will create it and
// append the new target field as well as the aggregate.  The mapping will also be
// updated with any new fields/aggregates.
func resolveAggregates(
	parsed *parser.Select,
	aggregates []*aggregateRequest,
	inputFields []Requestable,
	mapping *core.DocumentMapping,
	desc *client.CollectionDescription,
	descriptionsRepo *DescriptionsRepo,
) ([]Requestable, error) {
	fields := inputFields
	dependenciesByParentId := map[int][]int{}

	for _, aggregate := range aggregates {
		aggregateTargets := make([]AggregateTarget, len(aggregate.targets))

		for i, target := range aggregate.targets {
			var host Requestable
			var hostTarget *Targetable
			var childTarget OptionalChildTarget

			// If the host has not been requested the child mapping may not yet exist and
			// we must create it before we can convert the filter.
			childIsMapped := len(mapping.IndexesByName[target.hostExternalName]) != 0

			var hasHost bool
			var convertedFilter *Filter
			if childIsMapped {
				fieldDesc, isField := desc.GetField(target.hostExternalName)
				if isField && !fieldDesc.IsObject() {
					var order *OrderBy
					if target.order != nil && len(target.order.Conditions) > 0 {
						// For inline arrays the order element will consist of just a direction
						order = &OrderBy{
							Conditions: []OrderCondition{
								{
									Direction: SortDirection(target.order.Conditions[0].Direction),
								},
							},
						}
					}

					// If the hostExternalName matches a non-object field
					// we don't have to search for it and can just construct the
					// targeting info here.
					hasHost = true
					host = &Targetable{
						Field: Field{
							Index: int(fieldDesc.ID),
							Name:  target.hostExternalName,
						},
						Filter:  ToFilter(target.filter, mapping),
						Limit:   target.limit,
						OrderBy: order,
					}
				} else {
					childObjectIndex := mapping.FirstIndexOfName(target.hostExternalName)
					childMapping := mapping.ChildMappings[childObjectIndex]
					convertedFilter = ToFilter(target.filter, childMapping)
					host, hasHost = tryGetTarget(
						target.hostExternalName,
						convertedFilter,
						target.limit,
						toOrderBy(target.order, childMapping),
						fields,
					)
				}
			}

			if !hasHost {
				// If a matching host is not found, we need to construct and add it.
				index := mapping.GetNextIndex()

				dummyParsed := &parser.Select{
					Root: parsed.Root,
					Name: target.hostExternalName,
				}

				childCollectionName, err := getCollectionName(descriptionsRepo, dummyParsed, desc.Name)
				if err != nil {
					return nil, err
				}

				childMapping, _, err := getTopLevelInfo(descriptionsRepo, dummyParsed, childCollectionName)
				if err != nil {
					return nil, err
				}
				childMapping = childMapping.CloneWithoutRender()
				mapping.SetChildAt(index, childMapping)

				if !childIsMapped {
					// If the child was not mapped, the filter will not have been converted yet
					// so we must do that now.
					convertedFilter = ToFilter(target.filter, mapping.ChildMappings[index])
				}

				dummyJoin := &Select{
					Targetable: Targetable{
						Field: Field{
							Index: index,
							Name:  target.hostExternalName,
						},
						Filter:  convertedFilter,
						Limit:   target.limit,
						OrderBy: toOrderBy(target.order, childMapping),
					},
					CollectionName:  childCollectionName,
					DocumentMapping: *childMapping,
				}

				fields = append(fields, dummyJoin)
				mapping.Add(index, target.hostExternalName)

				host = dummyJoin
				hostTarget = &dummyJoin.Targetable
			} else {
				var isTargetable bool
				hostTarget, isTargetable = host.AsTargetable()
				if !isTargetable {
					// If the host is not targetable, such as when it is an inline-array field,
					// we don't need to worry about preserving the targetable information and
					// can just take the field properties.
					hostTarget = &Targetable{
						Field: Field{
							Index: host.GetIndex(),
							Name:  host.GetName(),
						},
					}
				}
			}

			if target.childExternalName != "" {
				hostSelect, isHostSelectable := host.AsSelect()
				if !isHostSelectable {
					// I believe this is dead code as the gql library should always catch this error first
					return nil, errors.New(
						"aggregate target host must be selectable, but was not",
					)
				}

				if len(hostSelect.IndexesByName[target.childExternalName]) == 0 {
					// I believe this is dead code as the gql library should always catch this error first
					return nil, errors.New(fmt.Sprintf(
						"Unable to identify aggregate child: %s", target.childExternalName,
					))
				}

				childTarget = OptionalChildTarget{
					// If there are multiple children of the same name there is no way
					// for us (or the consumer) to identify which one they are hoping for
					// so we take the first.
					Index:    hostSelect.IndexesByName[target.childExternalName][0],
					Name:     target.childExternalName,
					HasValue: true,
				}
			}

			aggregateTargets[i] = AggregateTarget{
				Targetable:  *hostTarget,
				ChildTarget: childTarget,
			}
		}

		newAggregate := Aggregate{
			Field:            aggregate.field,
			DocumentMapping:  *mapping,
			AggregateTargets: aggregateTargets,
		}
		fields = append(fields, &newAggregate)
		dependenciesByParentId[aggregate.field.Index] = aggregate.dependencyIndexes
	}

	// Once aggregates have been resolved we pair up their dependencies
	for aggregateId, dependencyIds := range dependenciesByParentId {
		aggregate := fieldAt(fields, aggregateId).(*Aggregate)
		for _, dependencyId := range dependencyIds {
			aggregate.Dependencies = append(aggregate.Dependencies, fieldAt(fields, dependencyId).(*Aggregate))
		}
	}

	return fields, nil
}

func fieldAt(fields []Requestable, index int) Requestable {
	for _, f := range fields {
		if f.GetIndex() == index {
			return f
		}
	}
	return nil
}

// aggregateDependencies maps aggregate names to the names of any aggregates
// that they may be dependent on.
var aggregateDependencies = map[string][]string{
	parserTypes.AverageFieldName: {
		parserTypes.CountFieldName,
		parserTypes.SumFieldName,
	},
}

// appendUnderlyingAggregates scans the given inputAggregates for any composite aggregates
// (e.g. average), and appends any missing dependencies to the collection and mapping.
//
// It will try and make use of existing aggregates that match the targeting parameters
// before creating new ones.  It will also adjust the target filters if required (e.g.
// average skips nil items).
func appendUnderlyingAggregates(
	inputAggregates []*aggregateRequest,
	mapping *core.DocumentMapping,
) []*aggregateRequest {
	aggregates := inputAggregates

	// Loop through the aggregates slice, including items that may have been appended
	// to the slice whilst looping.
	for i := 0; i < len(aggregates); i++ {
		aggregate := aggregates[i]

		dependencies, hasDependencies := aggregateDependencies[aggregate.field.Name]
		// If the aggregate has no dependencies, then we dont need to do anything and we continue.
		if !hasDependencies {
			continue
		}

		for _, target := range aggregate.targets {
			if target.childExternalName != "" {
				if _, isAggregate := parserTypes.Aggregates[target.childExternalName]; isAggregate {
					continue
				}
			}
			// Append a not-nil filter if the target is not an aggregate.
			// If the target has no childExternalName we assume it is an inline-array (and thus not an aggregate).
			// Aggregate-targets are excluded here as they are assumed to always have a value and
			// amending the filter introduces significant complexity for both machine and developer.
			appendNotNilFilter(target, target.childExternalName)
		}

		for _, dependencyName := range dependencies {
			var newAggregate *aggregateRequest
			aggregates, newAggregate = appendIfNotExists(
				dependencyName,
				aggregate.targets,
				aggregates,
				mapping,
			)
			aggregate.dependencyIndexes = append(aggregate.dependencyIndexes, newAggregate.field.Index)
		}
	}
	return aggregates
}

// appendIfNotExists attempts to match the given name and targets against existing
// aggregates, if a match is not found, it will append a new aggregate.
func appendIfNotExists(
	name string,
	targets []*aggregateRequestTarget,
	aggregates []*aggregateRequest,
	mapping *core.DocumentMapping,
) ([]*aggregateRequest, *aggregateRequest) {
	field, exists := tryGetMatchingAggregate(name, targets, aggregates)
	if exists {
		// If a match is found, there is nothing to do so we return the aggregates slice unchanged.
		return aggregates, field
	}

	// If a match is not found, create, map and append the
	// dependency to the aggregates collection.
	index := mapping.GetNextIndex()

	field = &aggregateRequest{
		field: Field{
			Index: index,
			Name:  name,
		},
		targets: targets,
	}

	mapping.Add(index, field.field.Name)
	return append(aggregates, field), field
}

// getRequestables returns a converted slice of consumer-requested Requestables
// and aggregateRequests from the given parsed.Fields slice. It also mutates the
// consumed mapping data.
func getRequestables(
	parsed *parser.Select,
	mapping *core.DocumentMapping,
	desc *client.CollectionDescription,
	descriptionsRepo *DescriptionsRepo,
) (fields []Requestable, aggregates []*aggregateRequest, err error) {
	// If this parser.Select is itself an aggregate, we need to append the
	// relevent info here as if it was a field of its own (due to a quirk of
	// the parser package).
	if _, isAggregate := parserTypes.Aggregates[parsed.Name]; isAggregate {
		index := mapping.GetNextIndex()
		aggregateReq, err := getAggregateRequests(index, parsed)
		if err != nil {
			return nil, nil, err
		}

		mapping.RenderKeys = append(mapping.RenderKeys, core.RenderKey{
			Index: index,
			Key:   parsed.Alias,
		})
		mapping.Add(index, parsed.Name)
		aggregates = append(aggregates, &aggregateReq)
	}

	for _, field := range parsed.Fields {
		switch f := field.(type) {
		case *parser.Field:
			// We can map all fields to the first (and only index)
			// as they support no value modifiers (such as filters/limits/etc).
			// All fields should have already been mapped by getTopLevelInfo
			index := mapping.FirstIndexOfName(f.Name)

			fields = append(fields, &Field{
				Index: index,
				Name:  f.Name,
			})

			mapping.RenderKeys = append(mapping.RenderKeys, core.RenderKey{
				Index: index,
				Key:   f.Alias,
			})
		case *parser.Select:
			index := mapping.GetNextIndex()

			// Aggregate targets are not known at this point, and must be evaluated
			// after all requested fields have been evaluated - so we note which
			// aggregates have been requested and their targets here, before finalizing
			// their evaluation later.
			if _, isAggregate := parserTypes.Aggregates[f.Name]; isAggregate {
				aggregateRequest, err := getAggregateRequests(index, f)
				if err != nil {
					return nil, nil, err
				}

				aggregates = append(aggregates, &aggregateRequest)
			} else {
				innerSelect, err := toSelect(descriptionsRepo, index, f, desc.Name)
				if err != nil {
					return nil, nil, err
				}
				fields = append(fields, innerSelect)
				mapping.SetChildAt(index, &innerSelect.DocumentMapping)
			}

			mapping.RenderKeys = append(mapping.RenderKeys, core.RenderKey{
				Index: index,
				Key:   f.Alias,
			})

			mapping.Add(index, f.Name)
		default:
			return nil, nil, errors.New(fmt.Sprintf(
				"Unexpected field type: %T",
				field,
			))
		}
	}
	return
}

func getAggregateRequests(index int, aggregate *parser.Select) (aggregateRequest, error) {
	aggregateTargets, err := getAggregateSources(aggregate)
	if err != nil {
		return aggregateRequest{}, err
	}

	if len(aggregateTargets) == 0 {
		return aggregateRequest{}, errors.New(
			"aggregate must be provided with a property to aggregate",
		)
	}

	return aggregateRequest{
		field: Field{
			Index: index,
			Name:  aggregate.Name,
		},
		targets: aggregateTargets,
	}, nil
}

// getCollectionName returns the name of the parsed collection.  This may be empty
// if this is a commit request.
func getCollectionName(
	descriptionsRepo *DescriptionsRepo,
	parsed *parser.Select,
	parentCollectionName string,
) (string, error) {
	if _, isAggregate := parserTypes.Aggregates[parsed.Name]; isAggregate {
		// This string is not used or referenced, its value is only there to aid debugging
		return "_topLevel", nil
	}

	if parsed.Name == parserTypes.GroupFieldName {
		return parentCollectionName, nil
	} else if parsed.Root == parserTypes.CommitSelection {
		return parentCollectionName, nil
	}

	if parentCollectionName != "" {
		parentDescription, err := descriptionsRepo.getCollectionDesc(parentCollectionName)
		if err != nil {
			return "", err
		}

		hostFieldDesc, parentHasField := parentDescription.GetField(parsed.Name)
		if parentHasField && hostFieldDesc.RelationType != 0 {
			// If this field exists on the parent, and it is a child object
			// then this collection name is the collection name of the child.
			return hostFieldDesc.Schema, nil
		}
	}

	return parsed.Name, nil
}

// getTopLevelInfo returns the collection description and maps the fields directly on the object.
func getTopLevelInfo(
	descriptionsRepo *DescriptionsRepo,
	parsed *parser.Select,
	collectionName string,
) (*core.DocumentMapping, *client.CollectionDescription, error) {
	mapping := core.NewDocumentMapping()

	if _, isAggregate := parserTypes.Aggregates[parsed.Name]; isAggregate {
		// If this is a (top-level) aggregate, then it will have no collection
		// description, and no top-level fields, so we return an empty mapping only
		return mapping, &client.CollectionDescription{}, nil
	}

	if parsed.Root != parserTypes.CommitSelection {
		mapping.Add(core.DocKeyFieldIndex, parserTypes.DocKeyFieldName)

		desc, err := descriptionsRepo.getCollectionDesc(collectionName)
		if err != nil {
			return nil, nil, err
		}

		// Map all fields from schema into the map as they are fetched automatically
		for _, f := range desc.Schema.Fields {
			if f.IsObject() {
				// Objects are skipped, as they are not fetched by default and
				// have to be requested via selects.
				continue
			}
			mapping.Add(int(f.ID), f.Name)
		}

		// Setting the type name must be done after adding the fields, as
		// the typeName index is dynamic, but the field indexes are not
		mapping.SetTypeName(collectionName)

		return mapping, &desc, nil
	}

	if parsed.Name == parserTypes.LinksFieldName {
		for i, f := range parserTypes.LinksFields {
			mapping.Add(i, f)
		}

		// Setting the type name must be done after adding the fields, as
		// the typeName index is dynamic, but the field indexes are not
		mapping.SetTypeName(parserTypes.LinksFieldName)
	} else {
		for i, f := range parserTypes.VersionFields {
			mapping.Add(i, f)
		}

		// Setting the type name must be done after adding the fields, as
		// the typeName index is dynamic, but the field indexes are not
		mapping.SetTypeName(parserTypes.CommitTypeName)
	}

	return mapping, &client.CollectionDescription{}, nil
}

func resolveFilterDependencies(
	descriptionsRepo *DescriptionsRepo,
	parentCollectionName string,
	source *parser.Filter,
	mapping *core.DocumentMapping,
	existingFields []Requestable,
) ([]Requestable, error) {
	if source == nil {
		return nil, nil
	}

	return resolveInnerFilterDependencies(
		descriptionsRepo,
		parentCollectionName,
		source.Conditions,
		mapping,
		existingFields,
	)
}

func resolveInnerFilterDependencies(
	descriptionsRepo *DescriptionsRepo,
	parentCollectionName string,
	source map[string]any,
	mapping *core.DocumentMapping,
	existingFields []Requestable,
) ([]Requestable, error) {
	newFields := []Requestable{}

	for key := range source {
		if strings.HasPrefix(key, "_") && key != parserTypes.DocKeyFieldName {
			continue
		}

		propertyMapped := len(mapping.IndexesByName[key]) != 0

		if !propertyMapped {
			index := mapping.GetNextIndex()

			dummyParsed := &parser.Select{
				Name: key,
			}

			childCollectionName, err := getCollectionName(descriptionsRepo, dummyParsed, parentCollectionName)
			if err != nil {
				return nil, err
			}

			childMapping, _, err := getTopLevelInfo(descriptionsRepo, dummyParsed, childCollectionName)
			if err != nil {
				return nil, err
			}
			childMapping = childMapping.CloneWithoutRender()
			mapping.SetChildAt(index, childMapping)

			dummyJoin := &Select{
				Targetable: Targetable{
					Field: Field{
						Index: index,
						Name:  key,
					},
				},
				CollectionName:  childCollectionName,
				DocumentMapping: *childMapping,
			}

			newFields = append(newFields, dummyJoin)
			mapping.Add(index, key)
		}

		keyIndex := mapping.FirstIndexOfName(key)

		if keyIndex >= len(mapping.ChildMappings) {
			// If the key index is outside the bounds of the child mapping array, then
			// this is not a relation/join and we can continue (no child props to process)
			continue
		}

		childMap := mapping.ChildMappings[keyIndex]
		if childMap == nil {
			// If childMap is nil, then this is not a relation/join and we can continue
			// (no child props to process)
			continue
		}

		childSource := source[key]
		childFilter, isChildFilter := childSource.(map[string]any)
		if !isChildFilter {
			// If the filter is not a child filter then the will be no inner dependencies to add and
			// we can continue.
			continue
		}

		dummyParsed := &parser.Select{
			Name: key,
		}

		childCollectionName, err := getCollectionName(descriptionsRepo, dummyParsed, parentCollectionName)
		if err != nil {
			return nil, err
		}

		allFields := enumerable.Concat(
			enumerable.New(newFields),
			enumerable.New(existingFields),
		)

		matchingFields := enumerable.Where(allFields, func(existingField Requestable) (bool, error) {
			return existingField.GetIndex() == keyIndex, nil
		})

		matchingHosts := enumerable.Select(matchingFields, func(existingField Requestable) (*Select, error) {
			host, isSelect := existingField.AsSelect()
			if !isSelect {
				// This should never be possible
				return nil, errors.New("host must be a Select, but was not")
			}
			return host, nil
		})

		host, hasHost, err := enumerable.TryGetFirst(matchingHosts)
		if err != nil {
			return nil, err
		}
		if !hasHost {
			// This should never be possible
			return nil, errors.New("failed to find host field")
		}

		childFields, err := resolveInnerFilterDependencies(
			descriptionsRepo,
			childCollectionName,
			childFilter,
			childMap,
			host.Fields,
		)
		if err != nil {
			return nil, err
		}

		host.Fields = append(host.Fields, childFields...)
	}

	return newFields, nil
}

// ToCommitSelect converts the given [parser.CommitSelect] into a [CommitSelect].
//
// In the process of doing so it will construct the document map required to access the data
// yielded by the [Select] embedded in the [CommitSelect].
func ToCommitSelect(ctx context.Context, txn datastore.Txn, parsed *parser.CommitSelect) (*CommitSelect, error) {
	underlyingSelect, err := ToSelect(ctx, txn, parsed.ToSelect())
	if err != nil {
		return nil, err
	}
	return &CommitSelect{
		Select:    *underlyingSelect,
		DocKey:    parsed.DocKey,
		FieldName: parsed.FieldName,
		Depth:     parsed.Depth,
		Cid:       parsed.Cid,
	}, nil
}

// ToMutation converts the given [parser.Mutation] into a [Mutation].
//
// In the process of doing so it will construct the document map required to access the data
// yielded by the [Select] embedded in the [Mutation].
func ToMutation(ctx context.Context, txn datastore.Txn, parsed *parser.Mutation) (*Mutation, error) {
	underlyingSelect, err := ToSelect(ctx, txn, parsed.ToSelect())
	if err != nil {
		return nil, err
	}

	return &Mutation{
		Select: *underlyingSelect,
		Type:   MutationType(parsed.Type),
		Data:   parsed.Data,
	}, nil
}

func toTargetable(index int, parsed *parser.Select, docMap *core.DocumentMapping) Targetable {
	return Targetable{
		Field:   toField(index, parsed),
		DocKeys: OptionalDocKeys(parsed.DocKeys),
		Filter:  ToFilter(parsed.Filter, docMap),
		Limit:   toLimit(parsed.Limit),
		GroupBy: toGroupBy(parsed.GroupBy, docMap),
		OrderBy: toOrderBy(parsed.OrderBy, docMap),
	}
}

func toField(index int, parsed *parser.Select) Field {
	return Field{
		Index: index,
		Name:  parsed.Name,
	}
}

// ToFilter converts the given `source` parser filter to a Filter using the given mapping.
//
// Any requestables identified by name will be converted to being identified by index instead.
func ToFilter(source *parser.Filter, mapping *core.DocumentMapping) *Filter {
	if source == nil {
		return nil
	}
	conditions := make(map[connor.FilterKey]any, len(source.Conditions))

	for sourceKey, sourceClause := range source.Conditions {
		key, clause := toFilterMap(sourceKey, sourceClause, mapping)
		conditions[key] = clause
	}

	return &Filter{
		Conditions:         conditions,
		ExternalConditions: source.Conditions,
	}
}

// toFilterMap converts a consumer-defined filter key-value into a filter clause
// keyed by field index.
//
// Return key will either be an int (field index), or a string (operator).
func toFilterMap(
	sourceKey string,
	sourceClause any,
	mapping *core.DocumentMapping,
) (connor.FilterKey, any) {
	if strings.HasPrefix(sourceKey, "_") && sourceKey != parserTypes.DocKeyFieldName {
		key := &Operator{
			Operation: sourceKey,
		}
		switch typedClause := sourceClause.(type) {
		case []any:
			// If the clause is an array then we need to convert any inner maps.
			returnClauses := []any{}
			for _, innerSourceClause := range typedClause {
				var returnClause any
				switch typedInnerSourceClause := innerSourceClause.(type) {
				case map[string]any:
					innerMapClause := map[connor.FilterKey]any{}
					for innerSourceKey, innerSourceValue := range typedInnerSourceClause {
						rKey, rValue := toFilterMap(innerSourceKey, innerSourceValue, mapping)
						innerMapClause[rKey] = rValue
					}
					returnClause = innerMapClause
				default:
					returnClause = innerSourceClause
				}
				returnClauses = append(returnClauses, returnClause)
			}
			return key, returnClauses
		default:
			return key, typedClause
		}
	} else {
		// If there are multiple properties of the same name we can just take the first as
		// we have no other reasonable way of identifying which property they mean if multiple
		// consumer specified requestables are available.  Aggregate dependencies should not
		// impact this as they are added after selects.
		index := mapping.FirstIndexOfName(sourceKey)
		key := &PropertyIndex{
			Index: index,
		}
		switch typedClause := sourceClause.(type) {
		case map[string]any:
			returnClause := map[connor.FilterKey]any{}
			for innerSourceKey, innerSourceValue := range typedClause {
				var innerMapping *core.DocumentMapping
				switch innerSourceValue.(type) {
				case map[string]any:
					// If the innerSourceValue is also a map, then we should parse the nested clause
					// using the child mapping, as this key must refer to a host property in a join
					// and deeper keys must refer to properties on the child items.
					innerMapping = mapping.ChildMappings[index]
				default:
					innerMapping = mapping
				}
				rKey, rValue := toFilterMap(innerSourceKey, innerSourceValue, innerMapping)
				returnClause[rKey] = rValue
			}
			return key, returnClause
		default:
			return key, sourceClause
		}
	}
}

func toLimit(source *parserTypes.Limit) *Limit {
	if source == nil {
		return nil
	}

	return &Limit{
		Limit:  source.Limit,
		Offset: source.Offset,
	}
}

func toGroupBy(source *parserTypes.GroupBy, mapping *core.DocumentMapping) *GroupBy {
	if source == nil {
		return nil
	}

	fields := make([]Field, len(source.Fields))
	for i, fieldName := range source.Fields {
		// If there are multiple properties of the same name we can just take the first as
		// we have no other reasonable way of identifying which property they mean if multiple
		// consumer specified requestables are available.  Aggregate dependencies should not
		// impact this as they are added after selects.
		key := mapping.FirstIndexOfName(fieldName)

		fields[i] = Field{
			Index: key,
			Name:  fieldName,
		}
	}

	return &GroupBy{
		Fields: fields,
	}
}

func toOrderBy(source *parserTypes.OrderBy, mapping *core.DocumentMapping) *OrderBy {
	if source == nil {
		return nil
	}

	conditions := make([]OrderCondition, len(source.Conditions))
	for conditionIndex, condition := range source.Conditions {
		fields := strings.Split(condition.Field, ".")
		fieldIndexes := make([]int, len(fields))
		currentMapping := mapping
		for fieldIndex, field := range fields {
			// If there are multiple properties of the same name we can just take the first as
			// we have no other reasonable way of identifying which property they mean if multiple
			// consumer specified requestables are available.  Aggregate dependencies should not
			// impact this as they are added after selects.
			firstFieldIndex := currentMapping.FirstIndexOfName(field)
			fieldIndexes[fieldIndex] = firstFieldIndex
			if fieldIndex != len(fields)-1 {
				// no need to do this for the last (and will panic)
				currentMapping = currentMapping.ChildMappings[firstFieldIndex]
			}
		}

		conditions[conditionIndex] = OrderCondition{
			FieldIndexes: fieldIndexes,
			Direction:    SortDirection(condition.Direction),
		}
	}

	return &OrderBy{
		Conditions: conditions,
	}
}

// RunFilter runs the given filter expression
// using the document, and evaluates.
func RunFilter(doc any, filter *Filter) (bool, error) {
	if filter == nil {
		return true, nil
	}

	return connor.Match(filter.Conditions, doc)
}

// equal compares the given Targetables and returns true if they can be considered equal.
func (s Targetable) equal(other Targetable) bool {
	if s.Index != other.Index &&
		s.Name != other.Name {
		return false
	}

	if !s.Filter.equal(other.Filter) {
		return false
	}

	if !s.Limit.equal(other.Limit) {
		return false
	}

	if !s.OrderBy.equal(other.OrderBy) {
		return false
	}

	return true
}

func (l *Limit) equal(other *Limit) bool {
	if l == nil {
		return other == nil
	}

	if other == nil {
		return l == nil
	}

	return l.Limit == other.Limit && l.Offset == other.Offset
}

func (f *Filter) equal(other *Filter) bool {
	if f == nil {
		return other == nil
	}

	if other == nil {
		return f == nil
	}

	return deepEqualConditions(f.Conditions, other.Conditions)
}

// deepEqualConditions performs a deep equality of two conditions.
// Handles: map[0xc00069cfd0:map[0xc005eda8c0:<nil>]] -> map[{5}:map[{_ne}:<nil>]]
func deepEqualConditions(x map[connor.FilterKey]any, y map[connor.FilterKey]any) bool {
	if len(x) != len(y) {
		return false
	}

	for xKey, xValue := range x {
		var isFoundInY bool

		// Ensure a matching key exists in the other map.
		for yKey, yValue := range y {
			if !xKey.Equal(yKey) {
				continue
			}

			xValueConditions, xOk := xValue.(map[connor.FilterKey]any)
			yValueConditions, yOk := yValue.(map[connor.FilterKey]any)
			if xOk && yOk {
				if deepEqualConditions(xValueConditions, yValueConditions) {
					isFoundInY = true
					break
				}
			} else if !xOk && !yOk {
				// Both are some basic values.
				if reflect.DeepEqual(xValue, yValue) {
					isFoundInY = true
					break
				}
			}
		}

		// No matching key (including matching data, of the pointer keys) found, so exit early.
		if !isFoundInY {
			return false
		}
	}

	return true
}

func (o *OrderBy) equal(other *OrderBy) bool {
	if o == nil {
		return other == nil
	}

	if other == nil {
		return o == nil
	}

	if len(o.Conditions) != len(other.Conditions) {
		return false
	}

	for i, conditionA := range o.Conditions {
		conditionB := other.Conditions[i]
		if conditionA.Direction != conditionB.Direction {
			return false
		}

		if len(conditionA.FieldIndexes) != len(conditionB.FieldIndexes) {
			return false
		}

		for j, fieldIndexA := range conditionA.FieldIndexes {
			fieldIndexB := conditionB.FieldIndexes[j]
			if fieldIndexA != fieldIndexB {
				return false
			}
		}
	}

	return true
}

// aggregateRequest is an intermediary struct defining a consumer-requested
// aggregate. These are defined before it can be determined as to which exact
// fields they target and so only specify the names of the target properties
// as they are know to the consumer.
type aggregateRequest struct {
	// This field.
	//
	// The Index and Name of *this* aggregate are known, and are specified here.
	field Field

	// The targets of this aggregate, as defined by the consumer.
	targets           []*aggregateRequestTarget
	dependencyIndexes []int
}

// aggregateRequestTarget contains the user defined information for an aggregate
// target before the actual underlying target is identified and/or created.
type aggregateRequestTarget struct {
	// The name of the host target as known by the consumer.
	//
	// This name may match zero to many field names requested by the consumer.
	hostExternalName string

	// The name of the child target as known by the consumer. This property is
	// optional and may be default depending on aggregate type and the type of
	// the host property.
	//
	// This name may match zero to many field names requested by the consumer.
	childExternalName string

	// The aggregate filter specified by the consumer for this target. Optional.
	filter *parser.Filter

	// The aggregate limit-offset specified by the consumer for this target. Optional.
	limit *Limit

	// The order in which items should be aggregated. Affects results when used with
	// limit. Optional.
	order *parserTypes.OrderBy
}

// Returns the source of the aggregate as requested by the consumer
func getAggregateSources(field *parser.Select) ([]*aggregateRequestTarget, error) {
	targets := make([]*aggregateRequestTarget, len(field.Statement.Arguments))

	for i, argument := range field.Statement.Arguments {
		switch argumentValue := argument.Value.GetValue().(type) {
		case string:
			targets[i] = &aggregateRequestTarget{
				hostExternalName: argumentValue,
			}
		case []*ast.ObjectField:
			hostExternalName := argument.Name.Value
			var childExternalName string
			var filter *parser.Filter
			var limit *Limit
			var order *parserTypes.OrderBy

			fieldArg, hasFieldArg := tryGet(argumentValue, parserTypes.Field)
			if hasFieldArg {
				if innerPathStringValue, isString := fieldArg.Value.GetValue().(string); isString {
					childExternalName = innerPathStringValue
				}
			}

			filterArg, hasFilterArg := tryGet(argumentValue, parserTypes.FilterClause)
			if hasFilterArg {
				var err error
				filter, err = parser.NewFilter(filterArg.Value.(*ast.ObjectValue))
				if err != nil {
					return nil, err
				}
			}

			limitArg, hasLimitArg := tryGet(argumentValue, parserTypes.LimitClause)
			offsetArg, hasOffsetArg := tryGet(argumentValue, parserTypes.OffsetClause)
			var limitValue int64
			var offsetValue int64
			if hasLimitArg {
				var err error
				limitValue, err = strconv.ParseInt(limitArg.Value.(*ast.IntValue).Value, 10, 64)
				if err != nil {
					return nil, err
				}
			}

			if hasOffsetArg {
				var err error
				offsetValue, err = strconv.ParseInt(offsetArg.Value.(*ast.IntValue).Value, 10, 64)
				if err != nil {
					return nil, err
				}
			}

			if hasLimitArg || hasOffsetArg {
				limit = &Limit{
					Limit:  limitValue,
					Offset: offsetValue,
				}
			}

			orderArg, hasOrderArg := tryGet(argumentValue, parserTypes.OrderClause)
			if hasOrderArg {
				switch orderArgValue := orderArg.Value.(type) {
				case *ast.EnumValue:
					// For inline arrays the order arg will be a simple enum declaring the order direction
					orderDirectionString := orderArgValue.Value
					orderDirection := parserTypes.OrderDirection(orderDirectionString)

					order = &parserTypes.OrderBy{
						Conditions: []parserTypes.OrderCondition{
							{
								Direction: orderDirection,
							},
						},
					}

				case *ast.ObjectValue:
					// For relations the order arg will be the complex order object as used by the host object
					// for non-aggregate ordering

					// We use the parser package parsing for convienience here
					orderConditions, err := parser.ParseConditionsInOrder(orderArgValue)
					if err != nil {
						return nil, err
					}

					order = &parserTypes.OrderBy{
						Conditions: orderConditions,
						Statement:  orderArgValue,
					}
				}
			}

			targets[i] = &aggregateRequestTarget{
				hostExternalName:  hostExternalName,
				childExternalName: childExternalName,
				filter:            filter,
				limit:             limit,
				order:             order,
			}
		}
	}

	return targets, nil
}

func tryGet(fields []*ast.ObjectField, name string) (*ast.ObjectField, bool) {
	for _, field := range fields {
		if field.Name.Value == name {
			return field, true
		}
	}
	return nil, false
}

// tryGetMatchingAggregate scans the given collection for aggregates with the given name and targets.
//
// Will return the matching target and true if one is found, otherwise will return false.
func tryGetMatchingAggregate(
	name string,
	targets []*aggregateRequestTarget,
	collection []*aggregateRequest,
) (*aggregateRequest, bool) {
collectionLoop:
	for _, aggregate := range collection {
		if aggregate.field.Name != name {
			continue
		}
		if len(aggregate.targets) != len(targets) {
			continue
		}

		for i, target := range targets {
			potentialMatchingTarget := aggregate.targets[i]

			if target.hostExternalName != potentialMatchingTarget.hostExternalName {
				continue collectionLoop
			}

			if target.childExternalName != potentialMatchingTarget.childExternalName {
				continue collectionLoop
			}

			if target.filter == nil && potentialMatchingTarget.filter != nil {
				continue collectionLoop
			}

			if potentialMatchingTarget.filter == nil && target.filter != nil {
				continue collectionLoop
			}

			if target.filter == nil && potentialMatchingTarget.filter == nil {
				// target matches, so continue the `target` loop and check the remaining.
				continue
			}

			if !reflect.DeepEqual(target.filter.Conditions, potentialMatchingTarget.filter.Conditions) {
				continue collectionLoop
			}
		}

		return aggregate, true
	}
	return nil, false
}

// tryGetTarget scans the given collection of Requestables for an item that matches the given
// name and filter.
//
// If a match is found the matching field will be returned along with true. If a match is not
// found, nil and false will be returned.
func tryGetTarget(
	name string,
	filter *Filter,
	limit *Limit,
	order *OrderBy,
	collection []Requestable,
) (Requestable, bool) {
	dummyTarget := Targetable{
		Field: Field{
			Name: name,
		},
		Filter:  filter,
		Limit:   limit,
		OrderBy: order,
	}

	for _, field := range collection {
		if field == nil {
			continue
		}
		targetable, isTargetable := field.AsTargetable()
		if isTargetable && targetable.equal(dummyTarget) {
			// Return the original field in order to preserve type specific info
			return field, true
		}
	}
	return nil, false
}

// appendNotNilFilter appends a not nil filter for the given child field
// to the given Select.
func appendNotNilFilter(field *aggregateRequestTarget, childField string) {
	if field.filter == nil {
		field.filter = &parser.Filter{}
	}

	if field.filter.Conditions == nil {
		field.filter.Conditions = map[string]any{}
	}

	var childBlock any
	var hasChildBlock bool
	if childField == "" {
		childBlock = field.filter.Conditions
	} else {
		childBlock, hasChildBlock = field.filter.Conditions[childField]
		if !hasChildBlock {
			childBlock = map[string]any{}
			field.filter.Conditions[childField] = childBlock
		}
	}

	typedChildBlock := childBlock.(map[string]any)
	typedChildBlock["_ne"] = nil
}