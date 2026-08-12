package types

import (
	"fmt"
	"reflect"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyEnabled        = []byte("Enabled")
	KeyRateBps        = []byte("RateBps")
	KeyMinFee         = []byte("MinFee")
	KeyMaxFee         = []byte("MaxFee")
	KeyMaxFeeEnabled  = []byte("MaxFeeEnabled")
	KeyApplyToMsgSend = []byte("ApplyToMsgSend")
	KeyFeeDenom       = []byte("FeeDenom")
)

type Params struct {
	Enabled        bool        `json:"enabled" yaml:"enabled"`
	RateBps        uint32      `json:"rate_bps" yaml:"rate_bps"`
	MinFee         sdkmath.Int `json:"min_fee" yaml:"min_fee"`
	MaxFee         sdkmath.Int `json:"max_fee" yaml:"max_fee"`
	MaxFeeEnabled  bool        `json:"max_fee_enabled" yaml:"max_fee_enabled"`
	ApplyToMsgSend bool        `json:"apply_to_msg_send" yaml:"apply_to_msg_send"`
	FeeDenom       string      `json:"fee_denom" yaml:"fee_denom"`
}

func DefaultParams() Params {
	return Params{
		Enabled:        false,
		RateBps:        0,
		MinFee:         sdkmath.ZeroInt(),
		MaxFee:         sdkmath.ZeroInt(),
		MaxFeeEnabled:  false,
		ApplyToMsgSend: true,
		FeeDenom:       core.MicroDoDenom,
	}
}

func MainnetV21Params() Params {
	return Params{
		Enabled:        true,
		RateBps:        1,
		MinFee:         sdkmath.NewInt(1_000 * core.MicroUnit),
		MaxFee:         sdkmath.ZeroInt(),
		MaxFeeEnabled:  false,
		ApplyToMsgSend: true,
		FeeDenom:       core.MicroDoDenom,
	}
}

func ParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&Params{})
}

func (p Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyEnabled, &p.Enabled, validateBool),
		paramstypes.NewParamSetPair(KeyRateBps, &p.RateBps, validateRateBps),
		paramstypes.NewParamSetPair(KeyMinFee, &p.MinFee, validateNonNegativeInt),
		paramstypes.NewParamSetPair(KeyMaxFee, &p.MaxFee, validateNonNegativeInt),
		paramstypes.NewParamSetPair(KeyMaxFeeEnabled, &p.MaxFeeEnabled, validateBool),
		paramstypes.NewParamSetPair(KeyApplyToMsgSend, &p.ApplyToMsgSend, validateBool),
		paramstypes.NewParamSetPair(KeyFeeDenom, &p.FeeDenom, validateFeeDenom),
	}
}

func (p Params) Validate() error {
	for _, pair := range p.ParamSetPairs() {
		value := reflect.Indirect(reflect.ValueOf(pair.Value)).Interface()
		if err := pair.ValidatorFn(value); err != nil {
			return err
		}
	}
	if p.MaxFeeEnabled && p.MaxFee.LT(p.MinFee) {
		return fmt.Errorf("max fee must be greater than or equal to min fee when max fee is enabled")
	}
	return nil
}

func validateBool(i interface{}) error {
	if _, ok := i.(bool); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateRateBps(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v > 10_000 {
		return fmt.Errorf("rate bps must not exceed 10000")
	}
	return nil
}

func validateNonNegativeInt(i interface{}) error {
	v, ok := i.(sdkmath.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNegative() {
		return fmt.Errorf("fee amount must be positive or zero")
	}
	return nil
}

func validateFeeDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v != core.MicroDoDenom {
		return fmt.Errorf("fee denom must be %s", core.MicroDoDenom)
	}
	return nil
}
