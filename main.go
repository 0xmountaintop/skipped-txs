package main

import (
	"context"

	"github.com/scroll-tech/go-ethereum/ethclient"
)

var l2GethEndpoint = "http://localhost:8545"

func main() {
	ctx := context.Background()

	l2GethClient, err := ethclient.DialContext(ctx, l2GethEndpoint)
	if err != nil {
		panic(err)
	}
	// Use gzip compression.
	l2GethClient.SetHeader("Accept-Encoding", "gzip")

	// GetNumSkippedTransactions

	// GetSkippedTransactionHashes

	// GetSkippedTransaction

	// GetTxBlockTraceOnTopOfBlock
}
