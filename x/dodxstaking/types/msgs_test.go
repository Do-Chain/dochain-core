package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
