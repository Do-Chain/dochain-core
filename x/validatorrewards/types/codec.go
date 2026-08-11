package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgRegisterRewardDenom{}, "validatorrewards/MsgRegisterRewardDenom")
	legacy.RegisterAminoMsg(cdc, &MsgCreateCampaign{}, "validatorrewards/MsgCreateCampaign")
	legacy.RegisterAminoMsg(cdc, &MsgReleaseCampaign{}, "validatorrewards/MsgReleaseCampaign")
	legacy.RegisterAminoMsg(cdc, &MsgCancelCampaign{}, "validatorrewards/MsgCancelCampaign")
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterRewardDenom{},
		&MsgCreateCampaign{},
		&MsgReleaseCampaign{},
		&MsgCancelCampaign{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
