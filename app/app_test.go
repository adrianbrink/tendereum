package app

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

func checkQuery(t *testing.T, app *TendereumApplication, addr common.Address, expected *big.Int) {
	qres := app.Query(addrQuery(addr))
	require.True(t, qres.GetCode().IsOK())
	assert.Equal(t, expected.String(), qres.GetLog())
}

func TestGenesisQuery(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, tearDown := setupTestCase(t)
	defer tearDown(t)
	og := app.was.state.IntermediateRoot(true)

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
	checkQuery(t, app, addr1, bal1)
	checkQuery(t, app, addr2, bal2)
	checkQuery(t, app, addr3, zero)

	qbad := app.Query(types.RequestQuery{Path: "/bad path"})
	assert.False(qbad.GetCode().IsOK())

	// committing this data should update hash
	cres := app.Commit()
	require.True(cres.IsOK())
	hash := common.BytesToHash(cres.Data)
	assert.NotEqual(og, hash)

	// queries should still work
	checkQuery(t, app, addr2, bal2)

	// commit again doesn't change anything
	cres = app.Commit()
	require.True(cres.IsOK())
	hash2 := common.BytesToHash(cres.Data)
	assert.Equal(hash, hash2)
}

func TestSendTx(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	app, tearDown := setupTestCase(t)
	defer tearDown(t)

	// some constants
	recv := common.StringToAddress("receive")
	bal := big.NewInt(9988776655443322)
	amount := big.NewInt(98765432100)

	// make new key and address
	priv, err := crypto.GenerateKey()
	require.NoError(err)
	pub := priv.Public()
	ecPub := pub.(*ecdsa.PublicKey)
	sender := crypto.PubkeyToAddress(*ecPub)

	// grant initial tokens
	res := app.SetOption(string(sender[:]), bal.String())
	require.Equal("Success", res)

	// commit genesis, and get initial root hash
	cres := app.Commit()
	require.True(cres.IsOK())
	hash := common.BytesToHash(cres.Data)

	// create tx
	gasLimit := big.NewInt(1000000)
	gasPrice := big.NewInt(100)
	tx := ethTypes.NewTransaction(0, recv, amount, gasLimit, gasPrice, nil)

	// sign it
	tx, err = ethTypes.SignTx(tx, app.signer, priv)
	require.NoError(err)
	from, err := ethTypes.Sender(app.signer, tx)
	require.NoError(err)
	assert.Equal(sender, from)

	// debug
	fmt.Printf("Tx: %s\n", tx)

	// generate bytes
	txBytes, err := encodeTx(tx)
	require.NoError(err)

	// make sure checktx works
	check := app.CheckTx(txBytes)
	assert.True(check.IsOK(), check.Log)

	// TODO: more with the delivery
	del := app.DeliverTx(txBytes)
	assert.True(del.IsOK(), del.Log)

	// commit again, and very the hash changed
	cres = app.Commit()
	require.True(cres.IsOK())
	hash2 := common.BytesToHash(cres.Data)
	assert.NotEqual(hash, hash2)
}
