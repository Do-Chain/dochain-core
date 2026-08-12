package wasmbinding

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/Daviddochain/dochain-core/v4/wasmbinding/bindings"
	dodxstakingkeeper "github.com/Daviddochain/dochain-core/v4/x/dodxstaking/keeper"
	dodxstakingtypes "github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	marketkeeper "github.com/Daviddochain/dochain-core/v4/x/market/keeper"
	markettypes "github.com/Daviddochain/dochain-core/v4/x/market/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CustomMessageDecorator returns decorator for custom CosmWasm bindings messages
func CustomMessageDecorator(
	market *marketkeeper.Keeper,
	dodxStaking *dodxstakingkeeper.Keeper,
) func(wasmkeeper.Messenger) wasmkeeper.Messenger {
	return func(old wasmkeeper.Messenger) wasmkeeper.Messenger {
		return &CustomMessenger{
			wrapped:           old,
			marketKeeper:      market,
			dodxStakingKeeper: dodxStaking,
		}
	}
}

type CustomMessenger struct {
	wrapped           wasmkeeper.Messenger
	marketKeeper      *marketkeeper.Keeper
	dodxStakingKeeper *dodxstakingkeeper.Keeper
}

var _ wasmkeeper.Messenger = (*CustomMessenger)(nil)

// DispatchMsg executes on the contractMsg.
func (m *CustomMessenger) DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
	if msg.Custom != nil {
		var contractMsg bindings.DoMsg
		if err := json.Unmarshal(msg.Custom, &contractMsg); err != nil {
			return nil, nil, nil, errorsmod.Wrap(err, "do msg")
		}

		switch {
		case contractMsg.Swap != nil:
			_, bz, err := m.swap(ctx, contractAddr, contractMsg.Swap)
			if err != nil {
				return nil, nil, nil, errorsmod.Wrap(err, "swap msg failed")
			}
			return nil, bz, nil, nil

		case contractMsg.SwapSend != nil:
			_, bz, err := m.swapSend(ctx, contractAddr, contractMsg.SwapSend)
			if err != nil {
				return nil, nil, nil, errorsmod.Wrap(err, "swap msg failed")
			}
			return nil, bz, nil, nil

		case contractMsg.DepositDodxRewards != nil:
			_, bz, err := m.depositDodxRewards(ctx, contractAddr, contractMsg.DepositDodxRewards)
			if err != nil {
				return nil, nil, nil, errorsmod.Wrap(err, "deposit dodx rewards msg failed")
			}
			return nil, bz, nil, nil

		default:
			return nil, nil, nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown do msg variant"}
		}
	}
	return m.wrapped.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
}

// depositDodxRewards lets a contract deposit its native DEX reward fees into
// x/dodxstaking reward accounting without an off-chain hot-wallet signer.
func (m *CustomMessenger) depositDodxRewards(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	contractMsg *bindings.DepositDodxRewards,
) ([]sdk.Event, [][]byte, error) {
	if contractMsg == nil {
		return nil, nil, wasmvmtypes.InvalidRequest{Err: "deposit dodx rewards msg was null"}
	}
	if m.dodxStakingKeeper == nil {
		return nil, nil, wasmvmtypes.UnsupportedRequest{Kind: "dodx staking keeper unavailable"}
	}

	msg := dodxstakingtypes.NewMsgDepositRewards(contractAddr, contractMsg.Amount)
	if err := msg.ValidateBasic(); err != nil {
		return nil, nil, errorsmod.Wrap(err, "failed validating MsgDepositRewards")
	}

	msgSvr := dodxstakingkeeper.NewMsgServerImpl(*m.dodxStakingKeeper)
	res, err := msgSvr.DepositRewards(sdk.WrapSDKContext(ctx), msg)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "depositing dodx rewards")
	}

	bz, err := json.Marshal(res)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "error marshal deposit dodx rewards response")
	}

	return nil, [][]byte{bz}, nil
}

// swap wraps around performing market swap
func (m *CustomMessenger) swap(ctx sdk.Context, contractAddr sdk.AccAddress, contractMsg *bindings.Swap) ([]sdk.Event, [][]byte, error) {
	res, err := PerformSwap(m.marketKeeper, ctx, contractAddr, contractMsg)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "perform swap")
	}

	bz, err := json.Marshal(res)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "error marshal swap response")
	}

	return nil, [][]byte{bz}, nil
}

// PerformSwap performs market swap
func PerformSwap(f *marketkeeper.Keeper, ctx sdk.Context, contractAddr sdk.AccAddress, contractMsg *bindings.Swap) (*markettypes.MsgSwapResponse, error) {
	if contractMsg == nil {
		return nil, wasmvmtypes.InvalidRequest{Err: "market swap msg was null"}
	}

	marketMsgSvr := marketkeeper.NewMsgServerImpl(*f)

	msgSwap := markettypes.NewMsgSwap(contractAddr, contractMsg.OfferCoin, contractMsg.AskDenom)

	if err := msgSwap.ValidateBasic(); err != nil {
		return nil, errorsmod.Wrap(err, "failed validating MsgSwap")
	}

	// swap
	res, err := marketMsgSvr.Swap(
		sdk.WrapSDKContext(ctx),
		msgSwap,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "swapping")
	}
	return res, nil
}

// swap wraps around performing market swap
func (m *CustomMessenger) swapSend(ctx sdk.Context, contractAddr sdk.AccAddress, contractMsg *bindings.SwapSend) ([]sdk.Event, [][]byte, error) {
	res, err := PerformSwapSend(m.marketKeeper, ctx, contractAddr, contractMsg)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "perform swap send")
	}

	bz, err := json.Marshal(res)
	if err != nil {
		return nil, nil, errorsmod.Wrap(err, "error marshal swap send response")
	}

	return nil, [][]byte{bz}, nil
}

// PerformSwapSend performs market swap
func PerformSwapSend(f *marketkeeper.Keeper, ctx sdk.Context, contractAddr sdk.AccAddress, contractMsg *bindings.SwapSend) (*markettypes.MsgSwapSendResponse, error) {
	if contractMsg == nil {
		return nil, wasmvmtypes.InvalidRequest{Err: "market swap send msg was null"}
	}

	marketMsgSvr := marketkeeper.NewMsgServerImpl(*f)

	toAddr, err := sdk.AccAddressFromBech32(contractMsg.ToAddress)
	if err != nil {
		return nil, err
	}

	msgSwapSend := markettypes.NewMsgSwapSend(contractAddr, toAddr, contractMsg.OfferCoin, contractMsg.AskDenom)

	if err := msgSwapSend.ValidateBasic(); err != nil {
		return nil, errorsmod.Wrap(err, "failed validating MsgSwapSend")
	}

	// swap
	res, err := marketMsgSvr.SwapSend(
		sdk.WrapSDKContext(ctx),
		msgSwapSend,
	)
	if err != nil {
		return nil, errorsmod.Wrap(err, "swapping and sending")
	}
	return res, nil
}
