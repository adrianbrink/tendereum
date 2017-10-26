package main

import (
	"os"

	"github.com/tendermint/tmlibs/cli"

	"github.com/cosmos/tendereum/cmd/tendereum/commands"
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
	cmd.Execute()
}
