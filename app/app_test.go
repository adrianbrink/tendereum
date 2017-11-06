package app

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/tendermint/abci/types"
	"github.com/tendermint/tmlibs/log"
)

func setupTestCase(t *testing.T) (app *TendereumApplication, tearDown func(t *testing.T)) {
	t.Log("Setting up one test case.")

	logger := log.NewTMLogger(os.Stdout)
	tmpDir, err := ioutil.TempDir("", "tendereum-apptest")
	require.NoError(t, err)

	app = NewTendereumApplication(tmpDir, logger)

	tearDown = func(t *testing.T) {
		t.Log("Tearing down one test case.")
		os.RemoveAll(tmpDir)
	}

	return app, tearDown
}

func TestInfo(t *testing.T) {
	assert := assert.New(t)
	_ = require.New(t)

	app, tearDown := setupTestCase(t)
	defer tearDown(t)

	req := types.RequestInfo{Version: fmt.Sprintf("0.11.0")}
	res := app.Info(req)

	assert.Equal("Tendereum", res.Data)
	assert.Equal("0.1.0", res.Version)
	hash := common.BytesToHash(res.LastBlockAppHash)
	assert.False(common.EmptyHash(hash))
}

func setupDB() (ethdb.Database, *state.StateDB, error) {
	tmpDir, err := ioutil.TempDir("", "tendereum-apptest")
	if err != nil {
		return nil, nil, err
	}
	return loadDB(tmpDir)
}

func TestStore(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	eth, db, err := setupDB()
	require.NoError(err)
	// note that original hash is not 0 bytes (as per state.New above),
	// but some actual value
	ogHash := db.IntermediateRoot(true)

	// account is non-existent
	addr := common.StringToAddress("hello")
	assert.True(db.Empty(addr))

	// until we set a value
	val := big.NewInt(123456789)
	db.SetBalance(addr, val)
	assert.False(db.Empty(addr))
	assert.Equal(val, db.GetBalance(addr))

	// perform hash, see it change
	hash := db.IntermediateRoot(true)
	assert.NotEqual(ogHash, hash)

	// write the db, commit doesn't change hash
	commit, err := db.CommitTo(eth, true)
	require.NoError(err)
	assert.Equal(hash, commit)
}

func addrQuery(addr common.Address) types.RequestQuery {
	return types.RequestQuery{
		Path: QueryBalance,
		Data: addr.Bytes(),
	}
}

func TestGenesisQuery(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, tearDown := setupTestCase(t)
	defer tearDown(t)

	// create three addresses
	addr1 := common.StringToAddress("genesis")
	addr2 := common.StringToAddress("block")
	addr3 := common.StringToAddress("fomo")

	// two are granted balance in genesis
	bal1 := big.NewInt(1234567890)
	bal2 := big.NewInt(9988776655443322)
	zero := big.NewInt(0)

	// grant them
	res := app.SetOption(string(addr1[:]), bal1.String())
	require.Equal("Success", res)
	res = app.SetOption(string(addr2[:]), bal2.String())
	require.Equal("Success", res)

	// query values
	qres1 := app.Query(addrQuery(addr1))
	require.True(qres1.GetCode().IsOK())
	assert.Equal(bal1.String(), qres1.GetLog())

	qres2 := app.Query(addrQuery(addr2))
	require.True(qres2.GetCode().IsOK())
	assert.Equal(bal2.String(), qres2.GetLog())

	qres3 := app.Query(addrQuery(addr3))
	require.True(qres3.GetCode().IsOK())
	assert.Equal(zero.String(), qres3.GetLog())

	qbad := app.Query(types.RequestQuery{Path: "/bad path"})
	assert.False(qbad.GetCode().IsOK())
}
