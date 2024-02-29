package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	// "path/filepath"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/eth"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

var (
	l2GethEndpoint  = "http://localhost:8545"
	network         = "mainnet" // mainnet or sepolia
	dumpTxsDir      = fmt.Sprintf("txs/%s/", network)
	dedupTxsFromDir = fmt.Sprintf("txs/%s/", network)
	dedupTxsToDir   = fmt.Sprintf("txs/%s/dedupped/", network)
	readTxsDir      = fmt.Sprintf("txs/%s/dedupped/", network)
	dumpTracesDir   = fmt.Sprintf("traces/%s/", network)
)

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
	log.Info("connected", "l2GethEndpoint", l2GethEndpoint)
	// Use gzip compression.
	l2GethClient.SetHeader("Accept-Encoding", "gzip")

	// dumpTxs(ctx, l2GethClient)

	// dedupTxs()

	getTraces(ctx, l2GethClient)
}

func getTraces(ctx context.Context, l2GethClient *ethclient.Client) {
	// read txs
	files, err := ioutil.ReadDir(dedupTxsToDir)
	if err != nil {
		log.Error("ioutil.ReadDir", "err", err)
		return
	}

	var rpcTxs []eth.RPCTransaction
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := ioutil.ReadFile(dedupTxsToDir + file.Name())
		if err != nil {
			log.Error("ioutil.ReadFile", "err", err)
			return
		}

		var tx eth.RPCTransaction
		if err = json.Unmarshal(data, &tx); err != nil {
			log.Error("json.Unmarshal", "err", err)
			return
		}

		rpcTxs = append(rpcTxs, tx)
	}

	// GetTxBlockTraceOnTopOfBlock
	for i, rpcTx := range rpcTxs {
		log.Info("processing", "i", i, "total", len(rpcTxs), "txHash", rpcTx.Hash.Hex())

		b, err := json.Marshal(rpcTx)
		if err != nil {
			panic(err)
		}
		// fmt.Println(string(b))

		tx := &types.Transaction{}
		if err := tx.UnmarshalJSON(b); err != nil {
			panic(err)
		}

		// b, err = tx.MarshalJSON()
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Println(string(b))

		blockNumber := rpc.BlockNumber(rpcTx.SkipBlockNumber.ToInt().Int64())
		traces, err := l2GethClient.GetTxBlockTraceOnTopOfBlock(ctx, tx, rpc.BlockNumberOrHash{BlockNumber: &blockNumber}, nil)
		if err != nil {
			log.Error("l2GethClient.GetTxBlockTraceOnTopOfBlock", "txHash", tx.Hash().Hex(), "err", err)
			continue
		}

		// dump traces
		b, err = json.Marshal(traces)
		if err != nil {
			panic(err)
		}

		if err := ioutil.WriteFile(fmt.Sprintf("%s%s.json", dumpTracesDir, tx.Hash().Hex()), b, 0644); err != nil {
			panic(err)
		}
	}
}

func dedupTxs() {
	files, err := ioutil.ReadDir(dedupTxsFromDir)
	if err != nil {
		log.Error("ioutil.ReadDir", "err", err)
		return
	}

	txDataMap := make(map[string]eth.RPCTransaction)
	var txs []eth.RPCTransaction
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := ioutil.ReadFile(dedupTxsFromDir + file.Name())
		if err != nil {
			log.Error("ioutil.ReadFile", "err", err)
			return
		}

		var tx eth.RPCTransaction
		if err = json.Unmarshal(data, &tx); err != nil {
			log.Error("json.Unmarshal", "err", err)
			return
		}

		if prev, ok := txDataMap[hex.EncodeToString(tx.Input)]; ok {
			if tx.SkipReason != prev.SkipReason {
				log.Error(
					"tx.SkipReason != prev.SkipReason",
					"hash", tx.Hash.Hex(),
					"prev.SkipReason", prev.SkipReason,
					"tx.SkipReason", tx.SkipReason,
				)
			}
			if tx.Accesses != prev.Accesses {
				log.Error("tx.Accesses != prev.Accesses", "hash", tx.Hash.Hex())
			}
			// if tx.From != prev.From {
			// 	log.Error("tx.From != prev.From", "hash", tx.Hash.Hex())
			// }
		} else {
			txDataMap[hex.EncodeToString(tx.Input)] = tx
			txs = append(txs, tx)
		}
	}

	for _, tx := range txs {
		b, err := json.Marshal(tx)
		if err != nil {
			log.Error("json.Marshal", "err", err)
			continue
		}

		if err := ioutil.WriteFile(fmt.Sprintf("%s%s.json", dedupTxsToDir, tx.Hash.Hex()), b, 0644); err != nil {
			log.Error("ioutil.WriteFile", "err", err)
			continue
		}
	}
}

func dumpTxs(ctx context.Context, l2GethClient *ethclient.Client) {
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
}

func dumpTx(tx *eth.RPCTransaction) error {
	b, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fmt.Sprintf("%s%s.json", dumpTxsDir, tx.Hash.Hex()), b, 0644)
}
