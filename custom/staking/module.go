package staking

import (
	"context"
	"encoding/json"
	"fmt"

	customtypes "github.com/Daviddochain/dochain-core/v4/custom/staking/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const EqualValidatorConsensusPower int64 = 1

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// AppModuleBasic defines the basic application module used by the staking module.
type AppModuleBasic struct {
	staking.AppModuleBasic
}

// RegisterLegacyAminoCodec registers the staking module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	customtypes.RegisterLegacyAminoCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the gov
// module.
func (am AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	// customize to set default genesis state deposit denom to udo
	defaultGenesisState := stakingtypes.DefaultGenesisState()
	defaultGenesisState.Params.BondDenom = core.MicroDoDenom

	return cdc.MustMarshalJSON(defaultGenesisState)
}

// AppModule implements an application module for the staking module.
type AppModule struct {
	staking.AppModule

	keeper       *keeper.Keeper
	paramsKeeper paramskeeper.Keeper
	ss           paramtypes.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec,
	keeper *keeper.Keeper,
	ak stakingtypes.AccountKeeper,
	bk stakingtypes.BankKeeper,
	pk paramskeeper.Keeper,
	ss paramtypes.Subspace,
) AppModule {
	return AppModule{
		AppModule:    staking.NewAppModule(cdc, keeper, ak, bk, ss),
		keeper:       keeper,
		paramsKeeper: pk,
		ss:           ss,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	stakingtypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))

	querier := keeper.Querier{Keeper: am.keeper}
	stakingtypes.RegisterQueryServer(
		cfg.QueryServer(),
		NewLegacyQueryServer(querier, am.ss, am.keeper),
	)

	m := keeper.NewMigrator(am.keeper, am.ss)
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 2 to 3: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 3 to 4: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 4, m.Migrate4to5); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 4 to 5: %v", stakingtypes.ModuleName, err))
	}
}

// EndBlock preserves normal staking state transitions, but keeps the CometBFT
// validator set at one-validator-one-power.
//
// The stored ValidatorUpdates value is intentionally kept compatible with the
// previous binary: zero-power removals followed by every active validator at
// equal power. That keeps historical replay and app hashes aligned. The return
// value is compacted from the underlying staking updates, so CometBFT is not
// asked to re-apply the unchanged active validator set on every block.
func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	standardUpdates, err := am.keeper.EndBlocker(ctx)
	if err != nil {
		return nil, err
	}

	storedUpdates, err := am.fullEqualValidatorPowerUpdates(ctx, standardUpdates)
	if err != nil {
		return nil, err
	}
	if replayWriteOptimizationActive(ctx) && len(standardUpdates) == 0 {
		existingUpdates, err := am.keeper.GetValidatorUpdates(ctx)
		if err != nil {
			return nil, err
		}
		if len(existingUpdates) > 0 {
			return nil, nil
		}
	}
	if err := am.keeper.SetValidatorUpdates(ctx, storedUpdates); err != nil {
		return nil, err
	}

	return compactEqualValidatorPowerUpdates(standardUpdates), nil
}

func replayWriteOptimizationActive(ctx context.Context) bool {
	return sdk.UnwrapSDKContext(ctx).BlockHeight() >= core.ReplayWriteOptimizationHeight
}

func (am AppModule) fullEqualValidatorPowerUpdates(ctx context.Context, standardUpdates []abci.ValidatorUpdate) ([]abci.ValidatorUpdate, error) {
	activeValidators, err := am.keeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return nil, err
	}

	updates := make([]abci.ValidatorUpdate, 0, len(standardUpdates)+len(activeValidators))
	for _, update := range standardUpdates {
		if update.Power == 0 {
			updates = append(updates, update)
		}
	}

	for _, validator := range activeValidators {
		update, err := equalPowerValidatorUpdate(validator)
		if err != nil {
			return nil, err
		}
		updates = append(updates, update)
	}

	return updates, nil
}

func compactEqualValidatorPowerUpdates(standardUpdates []abci.ValidatorUpdate) []abci.ValidatorUpdate {
	updates := make([]abci.ValidatorUpdate, 0, len(standardUpdates))
	for _, update := range standardUpdates {
		if update.Power > 0 {
			update.Power = EqualValidatorConsensusPower
		}
		updates = append(updates, update)
	}
	return updates
}

func equalPowerValidatorUpdate(validator stakingtypes.Validator) (abci.ValidatorUpdate, error) {
	pubKey, err := validator.TmConsPublicKey()
	if err != nil {
		return abci.ValidatorUpdate{}, err
	}

	return abci.ValidatorUpdate{
		PubKey: pubKey,
		Power:  EqualValidatorConsensusPower,
	}, nil
}
