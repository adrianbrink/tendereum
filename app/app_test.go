package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
