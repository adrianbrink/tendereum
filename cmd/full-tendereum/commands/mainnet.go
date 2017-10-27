package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// MainnetCmd initialises all files and connects to the mainnet.
var MainnetCmd = &cobra.Command{
	Use:   "mainnet",
	Short: "Initialises and starts a Tendereum node that connects to the mainnet.",
	Run:   connectMainnet,
}

func connectMainnet(cmd *cobra.Command, args []string) {
	fmt.Println(`Should initialise all files for Tendermint and Tendereum and start all 
necessary processes.`)
}
