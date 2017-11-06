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

var (
	// maxGas is the maximum gas per block. This shouldn't be a constant, but
	// I am not sure where to find/store the real value
	maxGas = (*core.GasPool)(big.NewInt(1000 * 1000 * 1000))
)

// TendereumApplication is an application that sits on top of Tendermint Core. It provides wraps
// an EVM and Ethereum state. It implements Query in order to service any RPC client.
// nolint: megacheck, structcheck
type TendereumApplication struct {
	types.BaseApplication

	db  ethdb.Database
	was *writeAheadState

	checkTxState    *state.StateDB
	committedHeight uint64
	committedHash   common.Hash

	chainConfig params.ChainConfig // evm tighly coupled to chainConfig
	vmConfig    vm.Config
	signer      ethTypes.Signer
	maxGas      *core.GasPool

	logger log.Logger
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

func newWriteAheadState(st *state.StateDB, gasPool *core.GasPool) *writeAheadState {
	// copy the pool... pointer to bigInt
	gp := *gasPool
	return &writeAheadState{
		state:        st,
		totalUsedGas: big.NewInt(0),
		gasPool:      &gp,
	}
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
func NewTendereumApplication(dbDir string, logger log.Logger) *TendereumApplication {

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

	eth, st, err := loadDB(dbDir)
	if err != nil {
		panic(err)
	}
	hash := st.IntermediateRoot(false)
	// TODO: height should be stored somewhere
	height := uint64(0)
	was := newWriteAheadState(st, maxGas)

	app := &TendereumApplication{
		db:              eth,
		was:             was,
		checkTxState:    st.Copy(),
		committedHash:   hash,
		committedHeight: height,

		// should set almost all options to 1 except for chain-id
		// needs to ensure that chain-id is unique and doesn't conflict with other current
		// networks.
		chainConfig: chainConfig,
		// TODO: Should be settable in order to enable EVM jit.
		vmConfig: vm.Config{Tracer: vm.NewStructLogger(nil)},
		maxGas:   maxGas,

		// EIP155 signer implements replay attack by including the chain-id in the signature
		signer: ethTypes.NewEIP155Signer(chainConfig.ChainId),
		logger: logger,
	}

	return app
}

// ------------------------------------------------------------------------------------------------
// ABCI Implementation for Tendermint Core

// ------------------------------------------------------------------------------------------------
// Info/Query connection

// Info returns some basic information about Tendereum.
// TODO: return hash and height info to do handshaking and recover after a restart
// without re-running the entire chain
func (ta *TendereumApplication) Info(req types.RequestInfo) (res types.ResponseInfo) {
	res = types.ResponseInfo{
		Data:             fmt.Sprintf("Tendereum"),
		Version:          fmt.Sprintf("0.1.0"),
		LastBlockHeight:  ta.committedHeight,
		LastBlockAppHash: ta.committedHash[:],
	}
	return res
}

// SetOption will not be used most likely. This could be useful to implement Web3 api that requires
// setting options, such as minimum gas price.
// Potentially this can be used to implement the management api.
func (ta *TendereumApplication) SetOption(key, value string) (log string) {
	// TODO: set data from the genesis file...
	// right now, only accounts

	// TODO: remove/refactor this
	addr := common.StringToAddress(key)
	tmp := big.NewInt(0)
	amount, ok := tmp.SetString(value, 0)
	if !ok {
		msg := fmt.Sprintf("Cannot parse balance: %s", value)
		panic(msg)
	}

	db := ta.was.state
	db.SetBalance(addr, amount)
	return "Success"
}

// Query handles all RPC queries that the RPC server sends to Tendermint Core.
// This is used to implement the RPC server which can be run by a light-node or full-node.
// Note: many of the web3rpc endpoints need to be mapped to tendermint rpc about
// blocks and headers and such...
//
// First for testing - query account balance
// api: internal/ethapi/api.go:PublicBlockChainAPI.GetBalance
func (ta *TendereumApplication) Query(req types.RequestQuery) (res types.ResponseQuery) {
	// must be compatible with http://godoc.org/pkg/github.com/ethereum/go-ethereum/ethclient/
	// possibly via adaptors. note all methods there...

	// see code at:
	// https://godoc.org/pkg/github.com/ethereum/go-ethereum/internal/ethapi/#PublicBlockChainAPI
	// https://godoc.org/github.com/ethereum/go-ethereum/internal/ethapi#PublicTransactionPoolAPI.SendTransaction

	// TODO: this should read from committed state (also from old block),
	// not the current delivertx state
	db := ta.was.state
	switch req.Path {
	case "/balance":
		var addr common.Address
		copy(addr[:], req.Data)
		bal := db.GetBalance(addr)
		res = types.ResponseQuery{
			Code: types.CodeType_OK,
			// TODO: encode balance as binary in data
			Log: bal.String(),
		}
	default:
		res = types.ResponseQuery{
			Code: types.CodeType_UnknownRequest,
			Log:  "Not yet implemented.",
		}
	}
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
	// https://github.com/ethereum/go-ethereum/blob/master/core/blockchain.go#L955-L961
	// some setup like here?
}

// DeliverTx delivers transactions that have been included in a finalised block.
// Updates the state
//
// This function is a slight adaptation of core.ApplyTransaction
// https://github.com/ethereum/go-ethereum/blob/master/core/state_processor.go#L86-L131
//
// api: internal/ethapi/api.go:PrivateAccountAPI.SendTransaction
// also signs the tx with local wallet
func (ta *TendereumApplication) DeliverTx(data []byte) (res types.Result) {
	// What about this logic from Process?
	// https://github.com/ethereum/go-ethereum/blob/master/core/state_processor.go#L72-L82
	// at the least, we should do:
	//   statedb.Prepare(tx.Hash(), block.Hash(), i)

	var tx ethTypes.Transaction
	if err := rlp.Decode(bytes.NewReader(data), &tx); err != nil {
		return types.ErrBaseEncodingError
	}

	msg, err := tx.AsMessage(ta.signer)
	if err != nil {
		return types.ErrInternalError
	}

	// look at https://github.com/ethereum/go-ethereum/blob/master/core/evm.go#L38-L59
	context := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Origin:      msg.From(),
		// -> set to proposer (in header?)
		// Coinbase:    beneficiary,
		// -> get blocknumber and time from header
		// BlockNumber: new(big.Int).Set(header.Number),
		// Time:        new(big.Int).Set(header.Time),
		// -> fake difficulty (or remove?)
		// Difficulty:  new(big.Int).Set(header.Difficulty),
		// -> GasLimit from a config?
		// GasLimit:    new(big.Int).Set(header.GasLimit),
		GasPrice: new(big.Int).Set(msg.GasPrice()),
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

	// store all receipts and logs...
	ta.was.receipts = append(ta.was.receipts, receipt)
	ta.was.allLogs = append(ta.was.allLogs, receipt.Logs...)

	return types.Result{Code: types.CodeType_OK}
}

// EndBlock is called to signal the end of the current block. It is called after all transactions
// have been delivered using DeliverTx.
func (ta *TendereumApplication) EndBlock(height uint64) (res types.ResponseEndBlock) {
	// https://github.com/ethereum/go-ethereum/blob/master/consensus/ethash/consensus.go#L509-L519
	// 	 state.AddBalance(header.Coinbase, reward)

	// TODO: add staking rules from precompiled staking contract and modify validators
	res = types.ResponseEndBlock{}
	return res
}

// Commit is called to obtain a unique state root for inclusion in the next Tendermint Core block.
// Writes all the changes to the state from DeliverTx to the underlying database.
func (ta *TendereumApplication) Commit() (res types.Result) {
	// https://github.com/ethereum/go-ethereum/blob/master/consensus/ethash/consensus.go#L514-L515
	//   header.Root = ta.was.state.IntermediateRoot(true)

	// Also, WriteBlockAndState
	// https://github.com/ethereum/go-ethereum/blob/master/core/blockchain.go#L965-L982
	// https://github.com/ethereum/go-ethereum/blob/master/core/blockchain.go#L806-L817

	// make sure we store the receipts and logs with the bloom filter and all,
	// so we can serve them up for web3 rpc requests

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

// TODO: actually load with proper roots and all....
func loadDB(dir string) (ethdb.Database, *state.StateDB, error) {
	eth, err := ethdb.NewLDBDatabase(dir, 16, 16)
	if err != nil {
		return nil, nil, err
	}

	// TODO: how to get the latest hash....
	// ideally we read from ethdb or such
	hash := common.StringToHash("")
	db, err := state.New(hash, state.NewDatabase(eth))
	if err != nil {
		return nil, nil, err
	}

	return eth, db, nil
}
