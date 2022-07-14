// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package cli

import (
	"context"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/sourcenetwork/defradb/logging"
	netclient "github.com/sourcenetwork/defradb/net/api/client"
)

// queryCmd represents the query command
var addReplicatorCmd = &cobra.Command{
	Use:   "addreplicator",
	Short: "Add a new replicator <collection> <peer>",
	Long: `Use this command if you wish to add a new target replicator
for the p2p data sync system.
		`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// get args
		collection := args[0]
		peerAddr, err := ma.NewMultiaddr(args[1])
		if err != nil {
			log.FatalE(ctx, "Invalid peer address", err)
		}

		log.Info(
			ctx,
			"Adding replicator for collection",
			logging.NewKV("PeerAddress", peerAddr),
			logging.NewKV("Collection", collection),
			logging.NewKV("RPCAddress", cfg.Net.RPCAddress))

		cred := insecure.NewCredentials()
		client, err := netclient.NewClient(cfg.Net.RPCAddress, grpc.WithTransportCredentials(cred))
		if err != nil {
			log.FatalE(ctx, "Couldn't create RPC client", err)
		}

		rpcTimeoutDuration, err := cfg.Net.RPCTimeoutDuration()
		if err != nil {
			log.FatalE(ctx, "Failed to parse RPC timeout duration", err)
		}
		ctx, cancel := context.WithTimeout(ctx, rpcTimeoutDuration)
		defer cancel()

		pid, err := client.AddReplicator(ctx, collection, peerAddr)
		if err != nil {
			log.FatalE(ctx, "Request failed", err)
		}
		log.Info(ctx, "Successfully added replicator", logging.NewKV("PID", pid))
	},
}

func init() {
	rpcCmd.AddCommand(addReplicatorCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// queryCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// queryCmd.Flags().StringVar(&queryStr, "query", "", "Query to run on the database")
}