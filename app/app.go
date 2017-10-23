package app

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
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

	signer ethTypes.Signer
	logger log.Logger
}

// Interface assertions
var _ types.Application = (*TendereumApplication)(nil)

// NewTendereumApplication returns a new instance of a Tendereum application.
// NOTE: Pass in a config struct which allows the specification of a chain-id. The signer needs the
// chain-id in order to provide replay protection.
func NewTendereumApplication(logger log.Logger) *TendereumApplication {
	return &TendereumApplication{
		logger: logger,
		signer: ethTypes.NewEIP155Signer(big.NewInt(1)),
	}
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

	_, err = ethTypes.Sender(ta.signer, tx)
	if err != nil {
		return types.ErrUnauthorized
	}

	if ta.currentState.GetNonce(from)+1 != tx.Nonce() {
		return types.ErrBadNonce
	}

	if ta.currentState.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return types.ErrInsufficientFunds
	}

	// the last parameter is whether we are on homestead. It is always true.
	intrGas := core.IntrinsicGas(tx.Data(), tx.To() == nil, true)
	if tx.Gas().Cmp(intrGas) < 0 {
		return types.ErrInternalError
	}

	return types.Result{Code: types.CodeType_OK}
}

func decodeTx(data []byte) (*ethTypes.Transaction, error) {
	var tx ethTypes.Transaction
	if err := rlp.Decode(bytes.NewReader(data), &tx); err != nil {
		return nil, err
	}
	return &tx, nil
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
func (ta *TendereumApplication) DeliverTx(tx []byte) (res types.Result) {
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
func (ta *TendereumApplication) Commit() (res types.Result) {
	res = types.Result{Code: types.CodeType_OK, Log: "Not yet implemented."}
	return res
}
