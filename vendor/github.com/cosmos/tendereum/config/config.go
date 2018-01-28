package config

// Config defines the top level configuration for a Tendereum node.
// TODO: Embed Tendermint config struct in order to configure the Tendermint node.
type Config struct {
	// Validator Settings
	// - gasprice - enforced during checkTx and just rejects transactions
	// GasLimit - max number of gas per block (consensus param)
}
