package post

import (
	dyncommkeeper "github.com/Daviddochain/dochain-core/v4/x/dyncomm/keeper"
	dyncommpost "github.com/Daviddochain/dochain-core/v4/x/dyncomm/post"
	treasurykeeper "github.com/Daviddochain/dochain-core/v4/x/treasury/keeper"
	valuefeekeeper "github.com/Daviddochain/dochain-core/v4/x/valuefee/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// HandlerOptions are the options required for constructing a default SDK PostHandler.
type HandlerOptions struct {
	DyncommKeeper  dyncommkeeper.Keeper
	BankKeeper     bankkeeper.Keeper
	AccountKeeper  accountkeeper.AccountKeeper
	TreasuryKeeper treasurykeeper.Keeper
	ValueFeeKeeper valuefeekeeper.Keeper
}

// NewPostHandler returns a PostHandler that checks and sets target
// commission rate for msg create validator and msg edit validator.
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
	_ = options.TreasuryKeeper

	return sdk.ChainPostDecorators(
		NewRefundUnusedGasDecorator(options.AccountKeeper, options.BankKeeper, options.ValueFeeKeeper),
		dyncommpost.NewDyncommPostDecorator(options.DyncommKeeper),
	), nil
}
