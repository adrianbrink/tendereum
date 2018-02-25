/*
Full Tendereum starts Tendermint Core, Tendereum and an RPC server. This is a full node that can be
a validator.
*/
package main

import (
	"os"

	"github.com/tendermint/tmlibs/cli"

	"github.com/adrianbrink/tendereum/cmd/full-tendereum/commands"
)

func main() {
	rootCmd := commands.RootCmd
	rootCmd.AddCommand(
		commands.MainnetCmd,
		commands.TestnetCmd,
		commands.DevelopmentCmd,
		commands.VersionCmd,
	)

	cmd := cli.PrepareBaseCmd(rootCmd, "TE", os.ExpandEnv("$HOME/.tendereum"))
	// nolint: errcheck
	cmd.Execute()
}
