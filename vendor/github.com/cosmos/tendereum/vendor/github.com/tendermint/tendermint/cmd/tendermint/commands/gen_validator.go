package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tendermint/tendermint/types"
)

// GenValidatorCmd allows the generation of a keypair for a
// validator.
var GenValidatorCmd = &cobra.Command{
	Use:   "gen_validator",
	Short: "Generate new validator keypair",
	Run:   genValidator,
}

func genValidator(cmd *cobra.Command, args []string) {
	privValidator := types.GenPrivValidatorFS("")
	privValidatorJSONBytes, _ := json.MarshalIndent(privValidator, "", "\t")
	fmt.Printf(`%v
`, string(privValidatorJSONBytes))
}
