package app

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/tendermint/abci/types"

	"github.com/tendermint/tmlibs/log"
)

const (
	maxTxSize = 32768
)

// TendereumApplication is an application that sits on top of Tendermint Core. It provides wraps
// an EVM and Ethereum state. It implements Query in order to service any RPC client.
type TendereumApplication struct {
	types.BaseApplication

	db  ethdb.Database
	was *writeAheadState

	checkTxState *state.StateDB

	chainConfig params.ChainConfig // evm tighly coupled to chainConfig
	vmConfig    vm.Config
	signer      ethTypes.Signer

	logger log.Logger
}

// Interface assertions
var _ types.Application = (*TendereumApplication)(nil)

// NewTendereumApplication returns a new instance of a Tendereum application.
// NOTE: Pass in a config struct which allows the specification of a chain-id. The signer needs the
// chain-id in order to provide replay protection.
func NewTendereumApplication(chainConfig params.ChainConfig, logger log.Logger) *TendereumApplication {
	return &TendereumApplication{
		// should set almost all options to 1 except for chain-id
		// needs to ensure that chain-id is unique and doesn't conflict with other current
		// networks.
		chainConfig: chainConfig,
		// TODO: Should be settable in order to enable EVM jit.
		vmConfig: vm.Config{Tracer: vm.NewStructLogger(nil)},
		// EIP155 signer implements replay attack by including the chain-id in the signature
		signer: ethTypes.NewEIP155Signer(chainConfig.ChainId),
		logger: logger,
	}
}

type writeAheadState struct {
	state        *state.StateDB
	txIndex      int
	transactions []*ethTypes.Transaction
	receipts     ethTypes.Receipts
	allLogs      []*ethTypes.Log
	totalUsedGas *big.Int
	gasPool      *core.GasPool
}

// ------------------------------------------------------------------------------------------------
// ABCI Implementation for Tendermint Core

// ------------------------------------------------------------------------------------------------
// Info/Query connection

// Info returns some basic information about Tendereum.
func (ta *TendereumApplication) Info(req types.RequestInfo) (res types.ResponseInfo) {
	res = types.ResponseInfo{Data: fmt.Sprintf("Tendereum"), Version: fmt.Sprintf("0.1.0")}
	return res
}

// SetOption will not be used most likely.
func (ta *TendereumApplication) SetOption(key, value string) (log string) {
	log = fmt.Sprintf("Not yet implemented.")
	return log
}

// Query handles all RPC queries that the RPC server sends to Tendermint Core.
// An example is getBalance().
func (ta *TendereumApplication) Query(req types.RequestQuery) (res types.ResponseQuery) {
	res = types.ResponseQuery{Code: types.CodeType_OK, Log: "Not yet implemented."}
	return res
}

// ------------------------------------------------------------------------------------------------
// Mempool connection

// CheckTx delivers transaction from the Tendermint Core mempool.
// NOTE: Ethereum currently enforces a max transaction size limit per transaction and not just per
// block. Should we also enforce a maximum size per transaction. Maybe it is in relation to the
// number of transactions per block for the last couple of blocks.
func (ta *TendereumApplication) CheckTx(data []byte) types.Result {
	tx, err := decodeTx(data)
	if err != nil {
		ta.logger.Error("Decoding transaction", "error", err)
		return types.ErrEncodingError
	}

	if tx.Size() > maxTxSize {
		return types.ErrInternalError
	}

	if tx.Value().Sign() < 0 {
		return types.ErrInternalError
	}

	from, err := ethTypes.Sender(ta.signer, tx)
	if err != nil {
		return types.ErrUnauthorized
	}

	if ta.checkTxState.GetNonce(from)+1 != tx.Nonce() {
		return types.ErrBadNonce
	}

	if ta.checkTxState.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return types.ErrInsufficientFunds
	}

	// the last parameter is whether we are on homestead. It is always true.
	intrGas := core.IntrinsicGas(tx.Data(), tx.To() == nil, true)
	if tx.Gas().Cmp(intrGas) < 0 {
		return types.ErrInternalError
	}

	// update current state to allow multiple transactions per block
	ta.checkTxState.SubBalance(from, tx.Cost())
	if to := tx.To(); to != nil {
		ta.checkTxState.AddBalance(*to, tx.Value())
	}
	ta.checkTxState.SetNonce(from, tx.Nonce()+1)

	return types.Result{Code: types.CodeType_OK}
}

// ------------------------------------------------------------------------------------------------
// Consensus connection

// InitChain is called when Tendermint Core is started.
func (ta *TendereumApplication) InitChain(req types.RequestInitChain) {
}

// BeginBlock is called before a new block is started. It is called before any transaction is
// submitted using DeliverTx.
func (ta *TendereumApplication) BeginBlock(req types.RequestBeginBlock) {
}

// DeliverTx delivers transactions that have been included in a finalised block.
// Updates the state
func (ta *TendereumApplication) DeliverTx(data []byte) (res types.Result) {
	var tx ethTypes.Transaction
	if err := rlp.Decode(bytes.NewReader(data), &tx); err != nil {
		return types.ErrBaseEncodingError
	}

	msg, err := tx.AsMessage(ta.signer)
	if err != nil {
		return types.ErrInternalError
	}

	context := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Origin:      msg.From(),
		GasPrice:    msg.GasPrice(),
	}

	evm := vm.NewEVM(context, ta.was.state, &ta.chainConfig, ta.vmConfig)
	// the third value is whether the transition failed. It is allowed to fail since we might
	// have potentially invalid messages in a block.
	// NOTE: This deducts gas and credits it to the defined coinbase.
	_, gas, _, err := core.ApplyMessage(evm, msg, ta.was.gasPool)
	if err != nil {
		return types.ErrInternalError
	}

	ta.was.totalUsedGas.Add(ta.was.totalUsedGas, gas)

	receipt := ethTypes.NewReceipt(s., failed bool, cumulativeGasUsed *big.Int)

	res = types.Result{Code: types.CodeType_OK, Log: "Not yet implemented."}
	return res
}

// EndBlock is called to signal the end of the current block. It is called after all transactions
// have been delivered using DeliverTx.
func (ta *TendereumApplication) EndBlock(height uint64) (res types.ResponseEndBlock) {
	res = types.ResponseEndBlock{}
	return res
}

// Commit is called to obtain a unique state root for inclusion in the next Tendermint Core block.
// Writes all the changes to the state from DeliverTx to the underlying database.
func (ta *TendereumApplication) Commit() (res types.Result) {
	res = types.Result{Code: types.CodeType_OK, Log: "Not yet implemented."}
	return res
}

// ------------------------------------------------------------------------------------------------
// Helpers

func decodeTx(data []byte) (*ethTypes.Transaction, error) {
	var tx ethTypes.Transaction
	if err := rlp.Decode(bytes.NewReader(data), &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}