package app

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"

	"github.com/tendermint/abci/types"
)

func setupTestCase(t *testing.T) (app *TendereumApplication, tearDown func(t *testing.T)) {
	t.Log("Setting up one test case.")
	app = &TendereumApplication{}

	tearDown = func(t *testing.T) {
		t.Log("Tearing down one test case.")
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

	assert.Equal(res, types.ResponseInfo{Data: fmt.Sprintf("Tendereum"),
		Version: fmt.Sprintf("0.1.0")})
}

func setupDB() (ethdb.Database, *state.StateDB, error) {
	tmpDir, err := ioutil.TempDir("", "tendereum-apptest")
	if err != nil {
		return nil, nil, err
	}

	eth, err := ethdb.NewLDBDatabase(tmpDir, 16, 16)
	if err != nil {
		return nil, nil, err
	}

	db, err := state.New(common.StringToHash(""),
		state.NewDatabase(eth))
	if err != nil {
		return nil, nil, err
	}

	return eth, db, nil
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
