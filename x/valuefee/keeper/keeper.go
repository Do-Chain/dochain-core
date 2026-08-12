package keeper

import (
	"github.com/Daviddochain/dochain-core/v4/x/valuefee/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type Keeper struct {
	paramSpace paramstypes.Subspace
}

func NewKeeper(paramSpace paramstypes.Subspace) Keeper {
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}
	return Keeper{paramSpace: paramSpace}
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	params := types.DefaultParams()
	k.paramSpace.GetParamSetIfExists(ctx, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
