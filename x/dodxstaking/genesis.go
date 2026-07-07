package dodxstaking

import (
	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/keeper"
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes x/dodxstaking state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, data *types.GenesisState) {
	for _, stake := range data.Stakes {
		addr, err := sdk.AccAddressFromBech32(stake.Address)
		if err != nil {
			panic(err)
		}
		k.SetStakeAmount(ctx, addr, stake.Amount.Amount)
		k.SetTotalStakedAmount(ctx, k.GetTotalStakedAmount(ctx).Add(stake.Amount.Amount))
	}

	k.SetGovernanceEnabled(ctx, data.GovernanceEnabled)
}

// ExportGenesis exports x/dodxstaking state.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	gs := types.DefaultGenesisState()
	k.IterateStakes(ctx, func(addr sdk.AccAddress, amount sdkmath.Int) bool {
		gs.Stakes = append(gs.Stakes, types.StakeRecord{
			Address: addr.String(),
			Amount:  sdk.NewCoin(core.MicroDODxDenom, amount),
		})
		return false
	})
	gs.GovernanceEnabled = k.GovernanceEnabled(ctx)
	return gs
}
