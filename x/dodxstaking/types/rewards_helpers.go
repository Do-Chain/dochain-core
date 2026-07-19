package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MaxRewardDenoms bounds all consensus and user-operation work performed by
// reward accounting. New denoms must be explicitly registered by a successful
// MsgDepositRewards transaction; arbitrary bank sends are never discovered.
const MaxRewardDenoms = 32

func validateRewardDenom(denom string) error {
	if err := sdk.ValidateDenom(denom); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidRewardDenom, err)
	}
	return nil
}
