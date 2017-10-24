package main

import (
	"os"

	"github.com/tendermint/tmlibs/cli"

	"github.com/adrianbrink/tendereum/cmd/tendereum/commands"
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

	/*
		fmt.Println("Tendereum has started.")

		addrPtr := flag.String("addr", "tcp://0.0.0.0:46658", "Listen address")
		abciPtr := flag.String("abci", "socket", "ABCI server: socket | grpc")
		flag.Parse()

		logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
		app := app.NewTendereumApplication(logger.With("module", "tendereum-application"))

		// Start the listener
		srv, err := server.NewServer(*addrPtr, *abciPtr, app)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
		srv.SetLogger(logger.With("module", "abci-server"))
		if _, err := srv.Start(); err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	*/
}
