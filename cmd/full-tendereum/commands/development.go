package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tendermint/tmlibs/common"

	"github.com/adrianbrink/tendereum/app"
)

// DevelopmentCmd initialises all files and connects to a local development net.
var DevelopmentCmd = &cobra.Command{
	Use:   "development",
	Short: "Initialises and starts a Tendereum node that connects to a local development net.",
	Run:   connectDevelopment,
}

func connectDevelopment(cmd *cobra.Command, args []string) {
	fmt.Println(`Should initialise all files for Tendermint and Tendereum and start all
necessary processes.`)

	app := app.NewTendereumApplication(logger)

	srv, err := startTendereum(app)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	n, err := startTendermint()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		srv.Stop()
		n.Stop()
	})
}
