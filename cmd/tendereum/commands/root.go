package commands

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tendermint/tmlibs/cli"
	"github.com/tendermint/tmlibs/cli/flags"
	"github.com/tendermint/tmlibs/log"
)

const (
	defaultLogLevel = "debug"
	flagLogLevel    = "log_level"
	flagAddress     = "addr"
)

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "main")
)

func init() {
	RootCmd.PersistentFlags().String(flagLogLevel, defaultLogLevel, "Log level")
	RootCmd.PersistentFlags().String(flagAddress, "tcp://0.0.0.0:46658", `The address that Tendereum
listens on for Tendermint Core`)
}

// RootCmd is the root command for Tendereum.
var RootCmd = &cobra.Command{
	Use:   "tendereum",
	Short: "Tendereum (Ethereum on Tendermint Core)in Go",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		level := viper.GetString(flagLogLevel)
		logger, err = flags.ParseLogLevel(level, logger, defaultLogLevel)
		if err != nil {
			return err
		}

		if viper.GetBool(cli.TraceFlag) {
			logger = log.NewTracingLogger(logger)
		}
		return nil
	},
}
