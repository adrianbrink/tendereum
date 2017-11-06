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
