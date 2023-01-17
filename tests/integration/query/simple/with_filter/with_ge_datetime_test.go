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

func TestQuerySimpleWithDateTimeGEFilterBlockWithEqualValue(t *testing.T) {
	test := testUtils.QueryTestCase{
		Description: "Simple query with basic ge int filter with equal value",
		Query: `query {
					users(filter: {CreatedAt: {_ge: "2017-07-23T03:46:56.647Z"}}) {
						Name
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21,
					"CreatedAt": "2017-07-23T03:46:56.647Z"
				}`,
				`{
					"Name": "Bob",
					"Age": 32,
					"CreatedAt": "2010-07-23T03:46:56.647Z"
				}`,
			},
		},
		Results: []map[string]any{
			{
				"Name": "John",
			},
		},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithDateTimeGEFilterBlockWithGreaterValue(t *testing.T) {
	test := testUtils.QueryTestCase{
		Description: "Simple query with basic ge int filter with equal value",
		Query: `query {
					users(filter: {CreatedAt: {_ge: "2017-07-22T03:46:56.647Z"}}) {
						Name
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21,
					"CreatedAt": "2017-07-23T03:46:56.647Z"
				}`,
				`{
					"Name": "Bob",
					"Age": 32,
					"CreatedAt": "2010-07-23T03:46:56.647Z"
				}`,
			},
		},
		Results: []map[string]any{
			{
				"Name": "John",
			},
		},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithDateTimeGEFilterBlockWithLesserValue(t *testing.T) {
	test := testUtils.QueryTestCase{
		Description: "Simple query with basic ge int filter with equal value",
		Query: `query {
					users(filter: {CreatedAt: {_ge: "2017-07-25T03:46:56.647Z"}}) {
						Name
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"Age": 21,
					"CreatedAt": "2017-07-23T03:46:56.647Z"
				}`,
				`{
					"Name": "Bob",
					"Age": 32,
					"CreatedAt": "2010-07-23T03:46:56.647Z"
				}`,
			},
		},
		Results: []map[string]any{},
	}

	executeTestCase(t, test)
}

func TestQuerySimpleWithDateTimeGEFilterBlockWithNilValue(t *testing.T) {
	test := testUtils.QueryTestCase{
		Description: "Simple query with basic ge nil filter",
		Query: `query {
					users(filter: {CreatedAt: {_ge: null}}) {
						Name
					}
				}`,
		Docs: map[int][]string{
			0: {
				`{
					"Name": "John",
					"CreatedAt": "2010-07-23T03:46:56.647Z"
				}`,
				`{
					"Name": "Bob"
				}`,
			},
		},
		Results: []map[string]any{
			{
				"Name": "Bob",
			},
			{
				"Name": "John",
			},
		},
	}

	executeTestCase(t, test)
}