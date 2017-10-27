/*
Package rpc implements an RPC server that serves the Web3 api. It connects to a local or remote  Tendermint Core
node. It is a light client to the Tendermint Core node and authenticates all data. Any client that talks Web3
gets a secure way to interact with Tendereum. For example a mobile client or desktop wallet starts the RPC
server and then connects to it over Web3.

The RPC server is exported as a C API so that it can be used from Android, iOS or any other programming language.
The library takes a URL for a Tendermint Core node and then starts the RPC server which serves a fully
authenticated Web3 api.

*/
package rpc
