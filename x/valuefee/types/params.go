package types

import (
	"fmt"
	"reflect"

	sdkmath "cosmossdk.io/math"
	core "github.com/Daviddochain/dochain-core/v4/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	KeyEnabled                  = []byte("Enabled")
	KeyRateBps                  = []byte("RateBps")
	KeyMinFee                   = []byte("MinFee")
	KeyMaxFee                   = []byte("MaxFee")
	KeyMaxFeeEnabled            = []byte("MaxFeeEnabled")
	KeyApplyToMsgSend           = []byte("ApplyToMsgSend")
	KeyFeeDenom                 = []byte("FeeDenom")
	KeyDenomValueRates          = []byte("DenomValueRates")
	KeyChargeUnknownDenomMinFee = []byte("ChargeUnknownDenomMinFee")
	KeyPolicyVersion            = []byte("PolicyVersion")
	KeyFeeExemptAddresses       = []byte("FeeExemptAddresses")
)

type DenomValueRate struct {
	Denom          string      `json:"denom" yaml:"denom"`
	UdoPerBaseUnit sdkmath.Int `json:"udo_per_base_unit" yaml:"udo_per_base_unit"`
}

type Params struct {
	Enabled                  bool             `json:"enabled" yaml:"enabled"`
	RateBps                  uint32           `json:"rate_bps" yaml:"rate_bps"`
	MinFee                   sdkmath.Int      `json:"min_fee" yaml:"min_fee"`
	MaxFee                   sdkmath.Int      `json:"max_fee" yaml:"max_fee"`
	MaxFeeEnabled            bool             `json:"max_fee_enabled" yaml:"max_fee_enabled"`
	ApplyToMsgSend           bool             `json:"apply_to_msg_send" yaml:"apply_to_msg_send"`
	FeeDenom                 string           `json:"fee_denom" yaml:"fee_denom"`
	DenomValueRates          []DenomValueRate `json:"denom_value_rates" yaml:"denom_value_rates"`
	ChargeUnknownDenomMinFee bool             `json:"charge_unknown_denom_min_fee" yaml:"charge_unknown_denom_min_fee"`
	PolicyVersion            uint32           `json:"policy_version" yaml:"policy_version"`
	FeeExemptAddresses       []string         `json:"fee_exempt_addresses" yaml:"fee_exempt_addresses"`
}

func DefaultParams() Params {
	return Params{
		Enabled:                  false,
		RateBps:                  0,
		MinFee:                   sdkmath.ZeroInt(),
		MaxFee:                   sdkmath.ZeroInt(),
		MaxFeeEnabled:            false,
		ApplyToMsgSend:           true,
		FeeDenom:                 core.MicroDoDenom,
		DenomValueRates:          []DenomValueRate{{Denom: core.MicroDoDenom, UdoPerBaseUnit: sdkmath.OneInt()}},
		ChargeUnknownDenomMinFee: false,
		PolicyVersion:            0,
		FeeExemptAddresses:       nil,
	}
}

func MainnetV21Params() Params {
	return Params{
		Enabled:                  true,
		RateBps:                  1,
		MinFee:                   sdkmath.NewInt(1_000 * core.MicroUnit),
		MaxFee:                   sdkmath.ZeroInt(),
		MaxFeeEnabled:            false,
		ApplyToMsgSend:           true,
		FeeDenom:                 core.MicroDoDenom,
		DenomValueRates:          []DenomValueRate{{Denom: core.MicroDoDenom, UdoPerBaseUnit: sdkmath.OneInt()}},
		ChargeUnknownDenomMinFee: false,
		PolicyVersion:            21,
	}
}

func MainnetV22Params() Params {
	params := MainnetV21Params()
	params.DenomValueRates = []DenomValueRate{
		{Denom: core.MicroDoDenom, UdoPerBaseUnit: sdkmath.OneInt()},
		{Denom: core.MicroDODxDenom, UdoPerBaseUnit: sdkmath.OneInt()},
	}
	params.ChargeUnknownDenomMinFee = true
	params.PolicyVersion = 22
	return params
}

func MainnetV23Params() Params {
	params := MainnetV22Params()
	params.PolicyVersion = 23
	return params
}

func MainnetV24Params() Params {
	params := MainnetV23Params()
	params.PolicyVersion = 24
	params.FeeExemptAddresses = []string{
		"do1mjgpy7wjhelpxssm8gl63fz5crl4tydvc2g5pj",
		"do1xas7z5ldgd296gj37ux2cwc37lt87h008v090d",
		"do1ftxpqe0dj6rcayz3fhw3xvesq47a26es8enyuu",
		"do12emm6cvg4sqh9cxsd6ys42kq67gyfjnw3jhhjr",
		"do1fulr5u0saspce2zuh2quqppesfx65cg5cu9eaz",
	}
	return params
}

func ParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&Params{})
}

func (p *Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyEnabled, &p.Enabled, validateBool),
		paramstypes.NewParamSetPair(KeyRateBps, &p.RateBps, validateRateBps),
		paramstypes.NewParamSetPair(KeyMinFee, &p.MinFee, validateNonNegativeInt),
		paramstypes.NewParamSetPair(KeyMaxFee, &p.MaxFee, validateNonNegativeInt),
		paramstypes.NewParamSetPair(KeyMaxFeeEnabled, &p.MaxFeeEnabled, validateBool),
		paramstypes.NewParamSetPair(KeyApplyToMsgSend, &p.ApplyToMsgSend, validateBool),
		paramstypes.NewParamSetPair(KeyFeeDenom, &p.FeeDenom, validateFeeDenom),
		paramstypes.NewParamSetPair(KeyDenomValueRates, &p.DenomValueRates, validateDenomValueRates),
		paramstypes.NewParamSetPair(KeyChargeUnknownDenomMinFee, &p.ChargeUnknownDenomMinFee, validateBool),
		paramstypes.NewParamSetPair(KeyPolicyVersion, &p.PolicyVersion, validatePolicyVersion),
		paramstypes.NewParamSetPair(KeyFeeExemptAddresses, &p.FeeExemptAddresses, validateFeeExemptAddresses),
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
	if !hasDenomValueRate(p.DenomValueRates, p.FeeDenom) {
		return fmt.Errorf("denom value rates must include fee denom %s", p.FeeDenom)
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

func validatePolicyVersion(i interface{}) error {
	_, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
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

func validateDenomValueRates(i interface{}) error {
	v, ok := i.([]DenomValueRate)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	seen := map[string]bool{}
	for _, rate := range v {
		if err := sdk.ValidateDenom(rate.Denom); err != nil {
			return fmt.Errorf("invalid denom value rate denom %q: %w", rate.Denom, err)
		}
		if seen[rate.Denom] {
			return fmt.Errorf("duplicate denom value rate for %s", rate.Denom)
		}
		if !rate.UdoPerBaseUnit.IsPositive() {
			return fmt.Errorf("udo per base unit must be positive for %s", rate.Denom)
		}
		seen[rate.Denom] = true
	}
	return nil
}

func hasDenomValueRate(rates []DenomValueRate, denom string) bool {
	for _, rate := range rates {
		if rate.Denom == denom {
			return true
		}
	}
	return false
}

func validateFeeExemptAddresses(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	seen := map[string]bool{}
	for _, address := range v {
		if _, bz, err := bech32.DecodeAndConvert(address); err != nil {
			return fmt.Errorf("invalid fee exempt address %q: %w", address, err)
		} else if len(bz) == 0 {
			return fmt.Errorf("invalid fee exempt address %q: empty address", address)
		}
		if seen[address] {
			return fmt.Errorf("duplicate fee exempt address %s", address)
		}
		seen[address] = true
	}
	return nil
}

func (p Params) IsFeeExemptAddress(addr sdk.AccAddress) bool {
	if len(addr) == 0 {
		return false
	}
	address := addr.String()
	for _, exempt := range p.FeeExemptAddresses {
		if exempt == address {
			return true
		}
	}
	return false
}
