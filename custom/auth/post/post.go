package post

import (
    dyncommkeeper "github.com/Daviddochain/dochain-core/v4/x/dyncomm/keeper"
    dyncommpost "github.com/Daviddochain/dochain-core/v4/x/dyncomm/post"
    treasurykeeper "github.com/Daviddochain/dochain-core/v4/x/treasury/keeper"
    sdk "github.com/cosmos/cosmos-sdk/types"
    accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
    bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

// HandlerOptions are the options required for constructing a default SDK PostHandler.
type HandlerOptions struct {
    DyncommKeeper dyncommkeeper.Keeper
    BankKeeper    bankkeeper.Keeper
    AccountKeeper accountkeeper.AccountKeeper
    TreasuryKeeper treasurykeeper.Keeper
}

// NewPostHandler returns a PostHandler that checks and sets target
// commission rate for msg create validator and msg edit validator.
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
    _ = options.BankKeeper
    _ = options.AccountKeeper
    _ = options.TreasuryKeeper

    return sdk.ChainPostDecorators(
        dyncommpost.NewDyncommPostDecorator(options.DyncommKeeper),
    ), nil
}
