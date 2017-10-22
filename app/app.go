package app

import (
	"fmt"

	"github.com/tendermint/abci/types"
)

// TendereumApplication is an application that sits on top of Tendermint Core. It provides wraps
// an EVM and Ethereum state. It implements Query in order to service any RPC client.
type TendereumApplication struct {
	types.BaseApplication
}

// Interface assertions
var _ types.Application = (*TendereumApplication)(nil)

// NewTendereumApplication returns a new instance of a Tendereum application.
func NewTendereumApplication() *TendereumApplication {
	return &TendereumApplication{}
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
func (ta *TendereumApplication) CheckTx(tx []byte) (res types.Result) {
	res = types.Result{Code: types.CodeType_OK, Log: "Not yet implemented."}
	return res
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
