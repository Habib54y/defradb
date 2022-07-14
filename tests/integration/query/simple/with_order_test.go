// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package simple

import (
	"testing"

	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

func TestQuerySimpleWithNumericOrderAscending(t *testing.T) {
	test := testUtils.QueryTestCase{
		Description: "Simple query with basic order ASC",
		Query: `query {
					users(order: {Age: ASC}) {
						Name
						Age
					}
				}`,
		Docs: map[int][]string{
			0: {
				(`{
				"Name": "John",
				"Age": 21
			}`),
				(`{
				"Name": "Bob",
				"Age": 32
			}`),
				(`{
				"Name": "Carlo",
				"Age": 55
			}`),
				(`{
				"Name": "Alice",
				"Age": 19
			}`)},
		},
		Results: []map[string]interface{}{
			{
				"Name": "Alice",
				"Age":  uint64(19),
			},
			{
				"Name": "John",
				"Age":  uint64(21),
			},
			{
				"Name": "Bob",
				"Age":  uint64(32),
			},
			{
				"Name": "Carlo",
				"Age":  uint64(55),
			},
		},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithNumericOrderDescending(t *testing.T) {
	test := testUtils.QueryTestCase{
		Description: "Simple query with basic order DESC",
		Query: `query {
					users(order: {Age: DESC}) {
						Name
						Age
					}
				}`,
		Docs: map[int][]string{
			0: {
				(`{
				"Name": "John",
				"Age": 21
			}`),
				(`{
				"Name": "Bob",
				"Age": 32
			}`),
				(`{
				"Name": "Carlo",
				"Age": 55
			}`),
				(`{
				"Name": "Alice",
				"Age": 19
			}`)},
		},
		Results: []map[string]interface{}{
			{
				"Name": "Carlo",
				"Age":  uint64(55),
			},
			{
				"Name": "Bob",
				"Age":  uint64(32),
			},
			{
				"Name": "John",
				"Age":  uint64(21),
			},
			{
				"Name": "Alice",
				"Age":  uint64(19),
			},
		},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithNumericOrderDescendingAndBooleanOrderAscending(t *testing.T) {
	test := testUtils.QueryTestCase{
		Description: "Simple query with compound order",
		Query: `query {
					users(order: {Age: DESC, Verified: ASC}) {
						Name
						Age
						Verified
					}
				}`,
		Docs: map[int][]string{
			0: {
				(`{
				"Name": "John",
				"Age": 21,
				"Verified": true
			}`),
				(`{
				"Name": "Bob",
				"Age": 21,
				"Verified": false
			}`),
				(`{
				"Name": "Carlo",
				"Age": 55,
				"Verified": true
			}`),
				(`{
				"Name": "Alice",
				"Age": 19,
				"Verified": false
			}`)},
		},
		Results: []map[string]interface{}{
			{
				"Name":     "Carlo",
				"Age":      uint64(55),
				"Verified": true,
			},
			{
				"Name":     "Bob",
				"Age":      uint64(21),
				"Verified": false,
			},
			{
				"Name":     "John",
				"Age":      uint64(21),
				"Verified": true,
			},
			{
				"Name":     "Alice",
				"Age":      uint64(19),
				"Verified": false,
			},
		},
	}

	executeTestCase(t, test)
}