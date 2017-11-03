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
	"github.com/ethereum/go-ethereum/crypto"
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
// nolint: megacheck, structcheck
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

// see the following for checking out how to do with precompiles:
// https://github.com/cosmos/tendereum/blob/master/vendor/github.com/ethereum/go-ethereum/core/vm/evm.go#L40-L52
// https://github.com/cosmos/tendereum/blob/master/vendor/github.com/ethereum/go-ethereum/core/vm/contracts.go#L49-L60
//
// echo is a silly start contract
type echo struct{}

var _ vm.PrecompiledContract = echo{}

func (echo) RequiredGas(input []byte) uint64 { return 100 }
func (echo) Run(input []byte) ([]byte, error) {
	// input is a multiple of 32 bytes, each one an evm argument
	// result the same

	// example of return in the same order
	// return input, nil

	// example to return the arguments in reverse order
	if len(input)%32 != 0 {
		return nil, fmt.Errorf("Input must be a multiple of 32 bytes")
	}
	output := make([]byte, 0, len(input))
	for l := len(input) - 32; l >= 0; l -= 32 {
		output = append(output, input[l:l+32]...)
	}
	return output, nil
}

func init() {
	vm.PrecompiledContractsByzantium[common.BytesToAddress([]byte{255})] = echo{}
}

// NewTendereumApplication returns a new instance of a Tendereum application.
// NOTE: Pass in a config struct which allows the specification of a chain-id. The signer needs the
// chain-id in order to provide replay protection.
// TODO: Pass in a config struct to provide the home directory.
func NewTendereumApplication(logger log.Logger) *TendereumApplication {

	chainConfig := params.ChainConfig{
		ChainId:        big.NewInt(1),
		HomesteadBlock: new(big.Int),
		DAOForkBlock:   new(big.Int),
		DAOForkSupport: true,
		EIP150Block:    new(big.Int),
		EIP150Hash:     common.Hash{},
		EIP155Block:    new(big.Int),
		EIP158Block:    new(big.Int),
		ByzantiumBlock: new(big.Int),
		Ethash:         new(params.EthashConfig),
	}

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

// nolint: megacheck, structcheck
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

// SetOption will not be used most likely. This could be useful to implement Web3 api that requires
// setting options, such as minimum gas price.
// Potentially this can be used to implement the management api.
func (ta *TendereumApplication) SetOption(key, value string) (log string) {
	log = fmt.Sprintf("Not yet implemented.")
	return log
}

// Query handles all RPC queries that the RPC server sends to Tendermint Core.
// This is used to implement the RPC server which can be run by a light-node or full-node.
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
	// NOTE: This deducts gas and credits it to the defined coinbase.
	_, gas, failed, err := core.ApplyMessage(evm, msg, ta.was.gasPool)
	if err != nil {
		return types.ErrInternalError
	}

	ta.was.state.Finalise(true)

	ta.was.totalUsedGas.Add(ta.was.totalUsedGas, gas)

	// since Byzantium the root hash of a receipt is empty
	var root []byte
	receipt := ethTypes.NewReceipt(root, failed, ta.was.totalUsedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = new(big.Int).Set(gas)
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.Context.Origin, tx.Nonce())
	}

	receipt.Logs = ta.was.state.GetLogs(tx.Hash())
	receipt.Bloom = ethTypes.CreateBloom(ethTypes.Receipts{receipt})

	return types.Result{Code: types.CodeType_OK}
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
