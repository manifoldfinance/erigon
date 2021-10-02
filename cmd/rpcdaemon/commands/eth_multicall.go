package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/holiman/uint256"
	common2 "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/common/math"
	"github.com/ledgerwatch/erigon/core"
	"github.com/ledgerwatch/erigon/core/rawdb"
	"github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/vm"
	"github.com/ledgerwatch/erigon/ethdb"
	"github.com/ledgerwatch/erigon/internal/ethapi"
	"github.com/ledgerwatch/erigon/rpc"
	"github.com/ledgerwatch/erigon/turbo/rpchelper"
	"github.com/ledgerwatch/erigon/turbo/shards"
	"github.com/ledgerwatch/erigon/turbo/transactions"
	"github.com/ledgerwatch/log/v3"
)

type MulticallRunlist map[common.Address][]hexutil.Bytes

type MulticallResult map[common.Address][]*MulticallExecutionResult

type MulticallExecutionResult struct {
	UsedGas    uint64
	ReturnData hexutil.Bytes `json:",omitempty"`
	Err        string        `json:",omitempty"`
}

// Call implements eth_call. Executes a new message call immediately without creating a transaction on the block chain.
func (api *APIImpl) Multicall(ctx context.Context, commonCallArgs ethapi.CallArgs, contractsWithPayloads MulticallRunlist, blockNrOrHash *rpc.BlockNumberOrHash, overrides *map[common.Address]ethapi.Account) (MulticallResult, error) {
	startTime := time.Now()

	// result stores
	execResults := make(MulticallResult)

	dbtx, err := api.db.BeginRo(ctx)
	if err != nil {
		return nil, err
	}
	defer dbtx.Rollback()

	chainConfig, err := api.chainConfig(dbtx)
	if err != nil {
		return nil, err
	}

	var baseFee *uint256.Int
	if blockNrOrHash == nil {
		var num = rpc.LatestBlockNumber
		blockNrOrHash = &rpc.BlockNumberOrHash{BlockNumber: &num}
	}
	blockNumber, hash, err := rpchelper.GetBlockNumber(*blockNrOrHash, dbtx, api.filters)
	if err != nil {
		return nil, err
	}
	blockHeader := rawdb.ReadHeader(dbtx, hash, blockNumber)
	if blockHeader == nil {
		return nil, fmt.Errorf("block header %d(%x) not found", blockNumber, hash)
	}
	if blockHeader != nil && blockHeader.BaseFee != nil {
		var overflow bool
		baseFee, overflow = uint256.FromBig(blockHeader.BaseFee)
		if overflow {
			return nil, fmt.Errorf("header.BaseFee uint256 overflow")
		}
	}

	var stateReader state.StateReader
	if num, ok := blockNrOrHash.Number(); ok && num == rpc.LatestBlockNumber {
		stateReader = state.NewPlainStateReader(dbtx)
	} else {
		stateReader = state.NewPlainState(dbtx, blockNumber)
	}
	stateCache := shards.NewStateCache(32, 0 /* no limit */)
	cachedReader := state.NewCachedReader(stateReader, stateCache)
	noop := state.NewNoopWriter()
	cachedWriter := state.NewCachedWriter(noop, stateCache)
	vmConfig := vm.Config{}

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if callTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, callTimeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	callArgsBuf := commonCallArgs
	callArgsBuf.MaxFeePerGas = (*hexutil.Big)(blockHeader.BaseFee)

	if callArgsBuf.Gas == nil || uint64(*callArgsBuf.Gas) == 0 {
		callArgsBuf.Gas = (*hexutil.Uint64)(&api.GasCap)
	}

	ibs := state.New(cachedReader)
	var execSeq int

	contractHasTEVM := ethdb.GetHasTEVM(dbtx)

	for contractAddr, payloads := range contractsWithPayloads {
		callArgsBuf.To = &contractAddr

		execResultsForContract := make([]*MulticallExecutionResult, 0, len(payloads))

		for _, payload := range payloads {
			if err := common2.Stopped(ctx.Done()); err != nil {
				return nil, err
			}

			callArgsBuf.Data = (*hexutil.Bytes)(&payload)

			msg, err := callArgsBuf.ToMessage(api.GasCap, baseFee)
			if err != nil {
				return nil, err
			}

			blockCtx, txCtx := transactions.GetEvmContext(msg, blockHeader, blockNrOrHash.RequireCanonical, dbtx, contractHasTEVM)
			blockCtx.GasLimit = math.MaxUint64
			blockCtx.MaxGasLimit = true

			evm := vm.NewEVM(blockCtx, txCtx, ibs, chainConfig, vmConfig)
			gp := new(core.GasPool).AddGas(msg.Gas())

			// Clone the state cache before applying the changes, clone is discarded
			ibs.Prepare(common.Hash{}, blockHeader.Hash(), execSeq)

			execResult, applyMsgErr := core.ApplyMessage(evm, msg, gp, true /* refunds */, true /* gasBailout */)

			var effectiveErrDesc string
			if applyMsgErr != nil {
				effectiveErrDesc = applyMsgErr.Error()
			} else if execResult.Err != nil {
				effectiveErrDesc = ethapi.NewRevertError(execResult).Error()
			}

			if err = ibs.FinalizeTx(evm.ChainRules, noop); err != nil {
				return nil, err
			}
			if err = ibs.CommitBlock(evm.ChainRules, cachedWriter); err != nil {
				return nil, err
			}

			mcExecResult := &MulticallExecutionResult{
				UsedGas: execResult.UsedGas,
				Err:     effectiveErrDesc,
			}

			if len(execResult.ReturnData) > 0 {
				mcExecResult.ReturnData = common.CopyBytes(execResult.ReturnData)
			}

			execResultsForContract = append(execResultsForContract, mcExecResult)
			execSeq++
		}

		execResults[contractAddr] = execResultsForContract
	}

	log.Info("Executed eth_multicall", "runtime_usec", time.Since(startTime).Microseconds())

	return execResults, nil
}
