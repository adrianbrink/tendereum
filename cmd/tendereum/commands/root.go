package commands

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tendermint/tmlibs/cli"
	"github.com/tendermint/tmlibs/log"
)

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "main")
)

// RootCmd is the root command for Tendereum.
var RootCmd = &cobra.Command{
	Use:   "tendereum",
	Short: "Tendereum (Ethereum on Tendermint Core)in Go",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if viper.GetBool(cli.TraceFlag) {
			logger = log.NewTracingLogger(logger)
		}
		return nil
	},
}
