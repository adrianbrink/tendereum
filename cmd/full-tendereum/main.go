/*
Full Tendereum starts Tendermint Core, Tendereum and an RPC server. This is a full node that can be
a validator.
*/
package main

import (
	//	"os"
	"fmt"
	"math/big"

	"github.com/ethereumproject/sputnikvm-ffi/go/sputnikvm"
	//	"github.com/tendermint/tmlibs/cli"

	"github.com/cosmos/tendereum/cmd/full-tendereum/commands"
)

func main() {
	fmt.Println("Tendereum")

	account := sputnikvm.AccountChangeStorageItem{
		Key:   big.NewInt(100),
		Value: big.NewInt(19),
	}

	fmt.Println(account.Key)

	rootCmd := commands.RootCmd
	rootCmd.AddCommand(
		commands.MainnetCmd,
		commands.TestnetCmd,
		commands.DevelopmentCmd,
		commands.VersionCmd,
	)

	//cmd := cli.PrepareBaseCmd(rootCmd, "TE", os.ExpandEnv("$HOME/.tendereum"))
	// nolint: errcheck
	//cmd.Execute()
}
