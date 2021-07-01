// Copyright 2020 Source Inc.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.
package planner

import (
	"github.com/sourcenetwork/defradb/core"
)

// @todo: Rebuild render system.
// @body: Current render system embeds render meta-data
// into EVERY SINGLE returned object map. This can be drastically
// reduced. Related: Replace Values() result with a typed object
// instead of a raw map[string]interface{}

// the final field select and render
type renderNode struct { // selectNode??
	p    *Planner
	plan planNode

	// fields []*base.FieldDescription
	// aliases []string
}

func (p *Planner) render() *renderNode {
	return &renderNode{p: p}
}

func (n *renderNode) Init() error            { return n.plan.Init() }
func (n *renderNode) Start() error           { return n.plan.Start() }
func (n *renderNode) Next() (bool, error)    { return n.plan.Next() }
func (n *renderNode) Spans(spans core.Spans) { n.plan.Spans(spans) }
func (n *renderNode) Close()                 { n.plan.Close() }
func (n *renderNode) Source() planNode       { return n.plan }

// we only need to implement the Values() func of the planNode
// interface since the embedded baseNode implements the rest
func (r *renderNode) Values() map[string]interface{} {
	doc := r.plan.Values()
	if doc == nil {
		return doc
	}
	return r.render(doc)
}

// render uses the __render map within the return doc via Values().
// it extracts the associated render meta-data, and returns a newly
// rendered map.
// The render rules are as follows:
// The doc returned by the plan has the following values:
// {
//	... document fields returned by scanPlan
// 	__render: {
// 		numRender: ... 	=> the number of fields in the actual selectionset
// 		fields: ... 	=> array of fields extracted from the raw query (Includes selection set + filter dependencies)
// 		aliases: ...	=> array of aliases, index matched to fields array.
// 	}
// }
func (r *renderNode) render(src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if renderMap, ok := src["__render"].(map[string]interface{}); ok {
		numRenderFields := renderMap["numResults"].(int)
		fields := renderMap["fields"].([]string)
		aliases := renderMap["aliases"].([]string)

		for i := 0; i < numRenderFields; i++ {
			field := fields[i]
			var dst string
			name := field
			dst = name
			alias := aliases[i]
			if alias != "" {
				dst = alias
			}

			if val, ok := src[name]; ok {
				switch v := val.(type) {
				case map[string]interface{}:
					result[dst] = r.render(v)
				case []map[string]interface{}:
					subdocs := make([]map[string]interface{}, 0)
					for _, subv := range v {
						subdocs = append(subdocs, r.render(subv))
					}
					result[dst] = subdocs
				default:
					result[dst] = v
				}
			} else {
				result[dst] = nil
			}
		}
	} else {
		return src
	}
	return result
}