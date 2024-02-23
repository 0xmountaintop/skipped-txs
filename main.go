package main

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/eth"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

var l2GethEndpoint = "http://localhost:8545"

func init() {
	output := io.Writer(os.Stderr)
	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	ostream := log.StreamHandler(output, log.TerminalFormat(usecolor))
	glogger := log.NewGlogHandler(ostream)
	// Set log level
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)
}

func main() {
	ctx := context.Background()

	l2GethClient, err := ethclient.DialContext(ctx, l2GethEndpoint)
	if err != nil {
		panic(err)
	}
	// Use gzip compression.
	l2GethClient.SetHeader("Accept-Encoding", "gzip")

	// GetNumSkippedTransactions
	nSkipped, err := l2GethClient.GetNumSkippedTransactions(ctx)
	if err != nil {
		panic(err)
	}
	log.Info("GetNumSkippedTransactions", "nSkipped", nSkipped)

	// GetSkippedTransactionHashes
	hashList, err := l2GethClient.GetSkippedTransactionHashes(ctx, 0, nSkipped)
	if err != nil {
		panic(err)
	}

	for i, hash := range hashList {
		log.Info("handling tx", "hash", hash, "i", i, "total", nSkipped)

		// GetSkippedTransaction
		tx, err := l2GethClient.GetSkippedTransaction(ctx, hash)
		if err != nil {
			panic(err)
		}

		err = dumpTx(tx)
		if err != nil {
			panic(err)
		}
	}

	// read txs
	rpcTxs := []*eth.RPCTransaction{}
	for _, rpcTx := range rpcTxs {
		// GetTxBlockTraceOnTopOfBlock
		tx := &types.Transaction{}
		blockNumber := rpc.BlockNumber(rpcTx.SkipBlockNumber.ToInt().Int64())
		_, err = l2GethClient.GetTxBlockTraceOnTopOfBlock(ctx, tx, rpc.BlockNumberOrHash{BlockNumber: &blockNumber}, nil)
		if err != nil {
			panic(err)
		}

		// dump traces
	}
}

func dumpTx(tx *eth.RPCTransaction) error {
	b, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(tx.Hash.Hex()+".json", b, 0644)
}
