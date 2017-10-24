package commands

import (
	"os"

	"github.com/spf13/viper"

	"github.com/tendermint/abci/server"
	"github.com/tendermint/abci/types"

	"github.com/tendermint/tmlibs/common"

	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/node"
)

func startTendereum(app types.Application) {
	addr := viper.GetString(flagAddress)
	srv, err := server.NewServer(addr, "socket", app)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if _, err := srv.Start(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
}

func startTendermint(app types.Application) {
	cfg := config.DefaultConfig()

	n, err := node.DefaultNewNode(cfg, logger.With("module", "tendermint-core"))
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	_, err = n.Start()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Trap signal, run forever.
	n.RunForever()
}
