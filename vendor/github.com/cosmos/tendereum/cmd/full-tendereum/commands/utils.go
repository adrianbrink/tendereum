package commands

import (
	"github.com/spf13/viper"

	"github.com/tendermint/abci/server"
	"github.com/tendermint/abci/types"

	"github.com/tendermint/tmlibs/common"

	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/proxy"
	tmTypes "github.com/tendermint/tendermint/types"
)

func startTendereum(app types.Application) (common.Service, error) {
	addr := viper.GetString(flagAddress)
	srv, err := server.NewServer(addr, "socket", app)
	if err != nil {
		return nil, err
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if _, err := srv.Start(); err != nil {
		return nil, err
	}

	return srv, nil
}

func startTendermint() (*node.Node, error) {
	cfg, err := tcmd.ParseConfig()
	if err != nil {
		return nil, err
	}

	n, err := node.NewNode(
		config.DefaultConfig(),
		tmTypes.LoadOrGenPrivValidatorFS(cfg.PrivValidatorFile()),
		proxy.DefaultClientCreator(cfg.ProxyApp, cfg.ABCI, cfg.DBDir()),
		node.DefaultGenesisDocProviderFunc(cfg),
		node.DefaultDBProvider,
		logger.With("module", "tendermint-core"))
	if err != nil {
		return nil, err
	}

	_, err = n.Start()
	if err != nil {
		return nil, err
	}

	return n, nil
}
