package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// TestnetCmd initialises all files and connects to the testnet.
var TestnetCmd = &cobra.Command{
	Use:   "testnet",
	Short: "Initialises and starts a Tendereum node that connects to the testnet.",
	Run:   connectTestnet,
}

func connectTestnet(cmd *cobra.Command, args []string) {
	fmt.Println(`Should initialise all files for Tendermint and Tendereum and start all
                     necessary processes.`)
}
