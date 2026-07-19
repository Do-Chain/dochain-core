package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func validateRewardDenom(denom string) error {
	if err := sdk.ValidateDenom(denom); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidRewardDenom, err)
	}
	return nil
}
