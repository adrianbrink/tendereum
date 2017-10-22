package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tendermint/abci/server"

	"github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/log"

	"github.com/adrianbrink/tendereum/app"
)

func main() {
	fmt.Println("Tendereum has started.")

	addrPtr := flag.String("addr", "tcp://0.0.0.0:46658", "Listen address")
	abciPtr := flag.String("abci", "socket", "ABCI server: socket | grpc")
	flag.Parse()

	app := app.NewTendereumApplication()
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

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

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		srv.Stop()
	})
}
