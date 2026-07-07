package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestValidateGenesis(t *testing.T) {
	addr := sdk.AccAddress([]byte("addr1_______________"))

	require.NoError(t, ValidateGenesis(DefaultGenesisState()))

	valid := &GenesisState{
		Stakes: []StakeRecord{
			{
				Address: addr.String(),
				Amount:  sdk.NewCoin(core.MicroDODxDenom, sdkmath.OneInt()),
			},
		},
		GovernanceEnabled: true,
	}
	require.NoError(t, ValidateGenesis(valid))

	duplicate := &GenesisState{
		Stakes: []StakeRecord{
			{
				Address: addr.String(),
				Amount:  sdk.NewCoin(core.MicroDODxDenom, sdkmath.OneInt()),
			},
			{
				Address: addr.String(),
				Amount:  sdk.NewCoin(core.MicroDODxDenom, sdkmath.OneInt()),
			},
		},
	}
	require.ErrorIs(t, ValidateGenesis(duplicate), ErrDuplicateStake)

	wrongDenom := &GenesisState{
		Stakes: []StakeRecord{
			{
				Address: addr.String(),
				Amount:  sdk.NewCoin(core.MicroDoDenom, sdkmath.OneInt()),
			},
		},
	}
	require.ErrorIs(t, ValidateGenesis(wrongDenom), ErrInvalidStakeDenom)
}
