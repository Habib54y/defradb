// Copyright 2024 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package test_acp_add_policy

import (
	"testing"

	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

func TestACP_AddPolicy_NoRelations_Error(t *testing.T) {
	test := testUtils.TestCase{

		Description: "Test acp, add policy, no relations, should return error",

		Actions: []any{
			testUtils.AddPolicy{
				Identity: actor1Identity,

				Policy: `
                    description: a policy

                    actor:
                      name: actor

                    resources:
                      users:
                        permissions:
                          write:
                            expr: owner
                          read:
                            expr: owner + reader

                        relations:
                `,

				ExpectedError: "resource users: resource missing owner relation: invalid policy",
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}

func TestACP_AddPolicy_NoRelationsLabel_Error(t *testing.T) {
	test := testUtils.TestCase{

		Description: "Test acp, add policy, no relations label, should return error",

		Actions: []any{
			testUtils.AddPolicy{
				Identity: actor1Identity,

				Policy: `
                    description: a policy

                    actor:
                      name: actor

                    resources:
                      users:
                        permissions:
                          write:
                            expr: owner
                          read:
                            expr: owner + reader
                `,

				ExpectedError: "resource users: resource missing owner relation: invalid policy",
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}
