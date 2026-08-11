package keeper

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/gogoproto/proto"
)

type Keeper struct {
	cdc      codecBinary
	storeKey storetypes.StoreKey

	AccountKeeper types.AccountKeeper
	BankKeeper    types.BankKeeper
	StakingKeeper types.StakingKeeper
	DistrKeeper   types.DistributionKeeper
	authority     string
}

type codecBinary interface {
	MustMarshal(o proto.Message) []byte
	MustUnmarshal(bz []byte, ptr proto.Message)
}

func NewKeeper(
	cdc codecBinary,
	storeKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	distrKeeper types.DistributionKeeper,
	authority string,
) Keeper {
	if addr := accountKeeper.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		AccountKeeper: accountKeeper,
		BankKeeper:    bankKeeper,
		StakingKeeper: stakingKeeper,
		DistrKeeper:   distrKeeper,
		authority:     authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) Authority() string {
	return k.authority
}

func (k Keeper) GetNextCampaignID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.NextCampaignIDKey)
	if bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

func (k Keeper) SetNextCampaignID(ctx sdk.Context, id uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	store.Set(types.NextCampaignIDKey, bz)
}

func (k Keeper) GetRewardDenom(ctx sdk.Context, denom string) (types.RewardDenom, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(rewardDenomKey(denom))
	if bz == nil {
		return types.RewardDenom{}, false
	}
	var record types.RewardDenom
	k.cdc.MustUnmarshal(bz, &record)
	return record, true
}

func (k Keeper) SetRewardDenom(ctx sdk.Context, record types.RewardDenom) {
	store := ctx.KVStore(k.storeKey)
	store.Set(rewardDenomKey(record.Denom), k.cdc.MustMarshal(&record))
}

func (k Keeper) IterateRewardDenoms(ctx sdk.Context, handler func(types.RewardDenom) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.RewardDenomKeyPrefix})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var record types.RewardDenom
		k.cdc.MustUnmarshal(iterator.Value(), &record)
		if handler(record) {
			break
		}
	}
}

func (k Keeper) GetCampaign(ctx sdk.Context, id uint64) (types.Campaign, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(campaignKey(id))
	if bz == nil {
		return types.Campaign{}, false
	}
	var campaign types.Campaign
	k.cdc.MustUnmarshal(bz, &campaign)
	return campaign, true
}

func (k Keeper) SetCampaign(ctx sdk.Context, campaign types.Campaign) {
	store := ctx.KVStore(k.storeKey)
	store.Set(campaignKey(campaign.Id), k.cdc.MustMarshal(&campaign))
	for _, allocation := range campaign.Allocations {
		store.Set(validatorCampaignKey(allocation.ValidatorAddress, campaign.Id), []byte{0x01})
	}
}

func (k Keeper) IterateCampaigns(ctx sdk.Context, handler func(types.Campaign) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte{types.CampaignKeyPrefix})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var campaign types.Campaign
		k.cdc.MustUnmarshal(iterator.Value(), &campaign)
		if handler(campaign) {
			break
		}
	}
}

func (k Keeper) IterateValidatorCampaigns(ctx sdk.Context, validator string, handler func(types.Campaign) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, validatorCampaignPrefix(validator))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		id := binary.BigEndian.Uint64(iterator.Key()[len(iterator.Key())-8:])
		campaign, found := k.GetCampaign(ctx, id)
		if !found {
			continue
		}
		if handler(campaign) {
			break
		}
	}
}

func (k Keeper) ValidateBankDenom(ctx sdk.Context, denom string) error {
	if err := types.ValidateRewardDenom(denom); err != nil {
		return err
	}
	if !k.BankKeeper.HasDenomMetaData(ctx, denom) {
		return fmt.Errorf("%w: missing bank metadata for %s", types.ErrInvalidDenom, denom)
	}
	return nil
}

func (k Keeper) ReleaseCampaign(ctx sdk.Context, campaign types.Campaign) ([]types.ValidatorAllocation, error) {
	if campaign.Cancelled {
		return nil, types.ErrCampaignInactive
	}

	released := make([]types.ValidatorAllocation, 0, len(campaign.Allocations))
	totalReleased := sdk.NewCoins()
	for i := range campaign.Allocations {
		allocation := campaign.Allocations[i]
		amount, err := vestedAmount(ctx.BlockHeight(), campaign, allocation)
		if err != nil {
			return nil, err
		}
		currentReleased, ok := sdkmath.NewIntFromString(defaultZero(allocation.Released))
		if !ok {
			return nil, fmt.Errorf("%w: invalid released amount", types.ErrInvalidCampaign)
		}
		releasable := amount.Sub(currentReleased)
		if !releasable.IsPositive() {
			continue
		}

		coin := sdk.NewCoin(campaign.Denom, releasable)
		valAddr, err := k.StakingKeeper.ValidatorAddressCodec().StringToBytes(allocation.ValidatorAddress)
		if err != nil {
			return nil, err
		}
		validator, err := k.StakingKeeper.Validator(ctx, valAddr)
		if err != nil {
			return nil, err
		}
		if validator == nil {
			return nil, distrtypes.ErrNoValidatorExists.Wrap(allocation.ValidatorAddress)
		}
		if err := k.BankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, distrtypes.ModuleName, sdk.NewCoins(coin)); err != nil {
			return nil, err
		}
		if campaign.DelegatorsOnly {
			if err := k.allocateDelegatorsOnly(ctx, sdk.ValAddress(valAddr), sdk.NewDecCoinsFromCoins(coin)); err != nil {
				return nil, err
			}
		} else if err := k.DistrKeeper.AllocateTokensToValidator(ctx, validator, sdk.NewDecCoinsFromCoins(coin)); err != nil {
			return nil, err
		}

		allocation.Released = amount.String()
		campaign.Allocations[i].Released = allocation.Released
		released = append(released, types.ValidatorAllocation{
			ValidatorAddress: allocation.ValidatorAddress,
			Amount:           coin,
			Released:         allocation.Released,
		})
		totalReleased = totalReleased.Add(coin)
	}

	if len(released) == 0 {
		return nil, types.ErrNoReleasableRewards
	}

	k.SetCampaign(ctx, campaign)
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeReleaseCampaign,
			sdk.NewAttribute(types.AttributeKeyCampaign, fmt.Sprintf("%d", campaign.Id)),
			sdk.NewAttribute(types.AttributeKeyAmount, totalReleased.String()),
		),
	)
	return released, nil
}

func (k Keeper) CancelCampaign(ctx sdk.Context, campaign types.Campaign) (sdk.Coins, error) {
	if campaign.Cancelled {
		return nil, types.ErrCampaignInactive
	}

	refund := sdk.NewCoins()
	for _, allocation := range campaign.Allocations {
		released, ok := sdkmath.NewIntFromString(defaultZero(allocation.Released))
		if !ok {
			return nil, fmt.Errorf("%w: invalid released amount", types.ErrInvalidCampaign)
		}
		remaining := allocation.Amount.Amount.Sub(released)
		if remaining.IsPositive() {
			refund = refund.Add(sdk.NewCoin(campaign.Denom, remaining))
		}
	}

	admin, err := sdk.AccAddressFromBech32(campaign.Admin)
	if err != nil {
		return nil, err
	}
	if !refund.IsZero() {
		if err := k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, admin, refund); err != nil {
			return nil, err
		}
	}
	campaign.Cancelled = true
	k.SetCampaign(ctx, campaign)
	return refund, nil
}

func (k Keeper) allocateDelegatorsOnly(ctx sdk.Context, valAddr sdk.ValAddress, tokens sdk.DecCoins) error {
	current, err := k.DistrKeeper.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return err
	}
	current.Rewards = current.Rewards.Add(tokens...)
	if err := k.DistrKeeper.SetValidatorCurrentRewards(ctx, valAddr, current); err != nil {
		return err
	}
	outstanding, err := k.DistrKeeper.GetValidatorOutstandingRewards(ctx, valAddr)
	if err != nil {
		return err
	}
	outstanding.Rewards = outstanding.Rewards.Add(tokens...)
	return k.DistrKeeper.SetValidatorOutstandingRewards(ctx, valAddr, outstanding)
}

func vestedAmount(height int64, campaign types.Campaign, allocation types.ValidatorAllocation) (sdkmath.Int, error) {
	if campaign.EndHeight == 0 || height >= campaign.EndHeight {
		return allocation.Amount.Amount, nil
	}
	if height < campaign.StartHeight {
		return sdkmath.ZeroInt(), nil
	}
	if campaign.EndHeight <= campaign.StartHeight {
		return sdkmath.ZeroInt(), fmt.Errorf("%w: invalid campaign height range", types.ErrInvalidCampaign)
	}
	elapsed := sdkmath.NewInt(height - campaign.StartHeight)
	duration := sdkmath.NewInt(campaign.EndHeight - campaign.StartHeight)
	return allocation.Amount.Amount.Mul(elapsed).Quo(duration), nil
}

func rewardDenomKey(denom string) []byte {
	return append([]byte{types.RewardDenomKeyPrefix}, []byte(denom)...)
}

func campaignKey(id uint64) []byte {
	key := make([]byte, 9)
	key[0] = types.CampaignKeyPrefix
	binary.BigEndian.PutUint64(key[1:], id)
	return key
}

func validatorCampaignPrefix(validator string) []byte {
	key := append([]byte{types.ValidatorCampaignKeyPrefix}, []byte(validator)...)
	return append(key, 0x00)
}

func validatorCampaignKey(validator string, id uint64) []byte {
	key := validatorCampaignPrefix(validator)
	idBz := make([]byte, 8)
	binary.BigEndian.PutUint64(idBz, id)
	return append(key, idBz...)
}

func defaultZero(value string) string {
	if value == "" {
		return "0"
	}
	return value
}
