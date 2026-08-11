package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func init() {
	sdk.GetConfig().SetBech32PrefixForAccount("do", "dopub")
	sdk.GetConfig().SetBech32PrefixForValidator("dovaloper", "dovaloperpub")
}

func TestMsgCreateCampaignValidateBasic(t *testing.T) {
	admin := sdk.AccAddress([]byte("admin---------------")).String()
	val1 := sdk.ValAddress([]byte("validator-one--------")).String()
	val2 := sdk.ValAddress([]byte("validator-two--------")).String()

	tests := []struct {
		name    string
		msg     MsgCreateCampaign
		wantErr string
	}{
		{
			name: "valid tiny amount has no minimum",
			msg: MsgCreateCampaign{
				Admin: admin,
				Denom: "ul2coin",
				Allocations: []ValidatorAllocation{{
					ValidatorAddress: val1,
					Amount:           sdk.NewCoin("ul2coin", sdkmath.OneInt()),
					Released:         "0",
				}},
			},
		},
		{
			name: "duplicate validator",
			msg: MsgCreateCampaign{
				Admin: admin,
				Denom: "ul2coin",
				Allocations: []ValidatorAllocation{
					{ValidatorAddress: val1, Amount: sdk.NewCoin("ul2coin", sdkmath.OneInt())},
					{ValidatorAddress: val1, Amount: sdk.NewCoin("ul2coin", sdkmath.OneInt())},
				},
			},
			wantErr: "duplicate validator",
		},
		{
			name: "allocation denom mismatch",
			msg: MsgCreateCampaign{
				Admin: admin,
				Denom: "ul2coin",
				Allocations: []ValidatorAllocation{{
					ValidatorAddress: val2,
					Amount:           sdk.NewCoin("uother", sdkmath.OneInt()),
				}},
			},
			wantErr: "does not match campaign denom",
		},
		{
			name: "bad height range",
			msg: MsgCreateCampaign{
				Admin:       admin,
				Denom:       "ul2coin",
				StartHeight: 10,
				EndHeight:   10,
				Allocations: []ValidatorAllocation{{ValidatorAddress: val1, Amount: sdk.NewCoin("ul2coin", sdkmath.OneInt())}},
			},
			wantErr: "end_height",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
