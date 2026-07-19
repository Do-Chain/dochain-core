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
		RewardAccumulators: []RewardAmountRecord{
			{Denom: core.MicroDoDenom, Amount: "1000000000000000000"},
		},
		RewardPools: []RewardAmountRecord{
			{Denom: core.MicroDoDenom, Amount: "1000"},
		},
		RewardDebts: []AccountRewardAmountRecord{
			{Address: addr.String(), Denom: core.MicroDoDenom, Amount: "1"},
		},
		PendingRewards: []AccountRewardAmountRecord{
			{Address: addr.String(), Denom: core.MicroDoDenom, Amount: "10"},
		},
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

	duplicateRewardDenom := &GenesisState{
		RewardPools: []RewardAmountRecord{
			{Denom: core.MicroDoDenom, Amount: "1"},
			{Denom: core.MicroDoDenom, Amount: "2"},
		},
	}
	require.Error(t, ValidateGenesis(duplicateRewardDenom))

	invalidRewardAmount := &GenesisState{
		RewardPools: []RewardAmountRecord{
			{Denom: core.MicroDoDenom, Amount: "0"},
		},
	}
	require.Error(t, ValidateGenesis(invalidRewardAmount))

	duplicateAccountReward := &GenesisState{
		PendingRewards: []AccountRewardAmountRecord{
			{Address: addr.String(), Denom: core.MicroDoDenom, Amount: "1"},
			{Address: addr.String(), Denom: core.MicroDoDenom, Amount: "2"},
		},
	}
	require.Error(t, ValidateGenesis(duplicateAccountReward))
}
