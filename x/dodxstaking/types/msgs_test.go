package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgStakeValidateBasic(t *testing.T) {
	addr := sdk.AccAddress([]byte("addr1_______________"))

	tests := []struct {
		name        string
		msg         *MsgStake
		expectedErr string
	}{
		{
			name: "valid",
			msg:  NewMsgStake(addr, sdk.NewCoin(core.MicroDODxDenom, sdkmath.OneInt())),
		},
		{
			name:        "invalid address",
			msg:         NewMsgStake(sdk.AccAddress{}, sdk.NewCoin(core.MicroDODxDenom, sdkmath.OneInt())),
			expectedErr: "invalid staker address (empty address string is not allowed): invalid address",
		},
		{
			name:        "wrong denom",
			msg:         NewMsgStake(addr, sdk.NewCoin(core.MicroDoDenom, sdkmath.OneInt())),
			expectedErr: "got udo: DODx staking only accepts udodx",
		},
		{
			name:        "zero amount",
			msg:         NewMsgStake(addr, sdk.NewCoin(core.MicroDODxDenom, sdkmath.ZeroInt())),
			expectedErr: "0udodx: invalid DODx staking amount",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectedErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestMsgUnstakeValidateBasic(t *testing.T) {
	addr := sdk.AccAddress([]byte("addr1_______________"))

	tests := []struct {
		name        string
		msg         *MsgUnstake
		expectedErr string
	}{
		{
			name: "valid",
			msg:  NewMsgUnstake(addr, sdk.NewCoin(core.MicroDODxDenom, sdkmath.OneInt())),
		},
		{
			name:        "wrong denom",
			msg:         NewMsgUnstake(addr, sdk.NewCoin(core.MicroDoDenom, sdkmath.OneInt())),
			expectedErr: "got udo: DODx staking only accepts udodx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.expectedErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestMsgDepositRewardsValidateBasic(t *testing.T) {
	addr := sdk.AccAddress([]byte("addr1_______________"))

	tests := []struct {
		name string
		msg  *MsgDepositRewards
		err  error
	}{
		{
			name: "valid",
			msg:  NewMsgDepositRewards(addr, sdk.NewCoins(sdk.NewCoin(core.MicroDoDenom, sdkmath.OneInt()))),
		},
		{
			name: "invalid address",
			msg:  NewMsgDepositRewards(sdk.AccAddress{}, sdk.NewCoins(sdk.NewCoin(core.MicroDoDenom, sdkmath.OneInt()))),
			err:  sdkerrors.ErrInvalidAddress,
		},
		{
			name: "zero amount",
			msg:  NewMsgDepositRewards(addr, sdk.NewCoins(sdk.NewCoin(core.MicroDoDenom, sdkmath.ZeroInt()))),
			err:  ErrInvalidRewardAmount,
		},
		{
			name: "bad denom",
			msg: &MsgDepositRewards{
				Depositor: addr.String(),
				Amount:    sdk.Coins{sdk.Coin{Denom: "bad denom", Amount: sdkmath.OneInt()}},
			},
			err: ErrInvalidRewardAmount,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestMsgClaimRewardsValidateBasic(t *testing.T) {
	addr := sdk.AccAddress([]byte("addr1_______________"))

	tests := []struct {
		name string
		msg  *MsgClaimRewards
		err  error
	}{
		{
			name: "valid all denoms",
			msg:  NewMsgClaimRewards(addr, nil),
		},
		{
			name: "valid selected denoms",
			msg:  NewMsgClaimRewards(addr, []string{core.MicroDoDenom, core.MicroDODxDenom}),
		},
		{
			name: "invalid address",
			msg:  NewMsgClaimRewards(sdk.AccAddress{}, nil),
			err:  sdkerrors.ErrInvalidAddress,
		},
		{
			name: "duplicate denom",
			msg:  NewMsgClaimRewards(addr, []string{core.MicroDoDenom, core.MicroDoDenom}),
			err:  ErrInvalidRewardDenom,
		},
		{
			name: "bad denom",
			msg:  NewMsgClaimRewards(addr, []string{"bad denom"}),
			err:  ErrInvalidRewardDenom,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.err)
		})
	}
}
