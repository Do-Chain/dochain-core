package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	proto "github.com/cosmos/gogoproto/proto"
)

// MsgDepositRewards deposits DEX fee rewards into x/dodxstaking for pro-rata
// distribution to native DODX stakers. DEX contracts may also bank-send fees
// directly to the module account; the keeper syncs those balances at BeginBlock.
type MsgDepositRewards struct {
	Depositor string    `protobuf:"bytes,1,opt,name=depositor,proto3" json:"depositor,omitempty"`
	Amount    sdk.Coins `protobuf:"bytes,2,rep,name=amount,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"amount"`
}

func (m *MsgDepositRewards) Reset()         { *m = MsgDepositRewards{} }
func (m *MsgDepositRewards) String() string { return proto.CompactTextString(m) }
func (*MsgDepositRewards) ProtoMessage()    {}

func (m *MsgDepositRewards) GetDepositor() string {
	if m != nil {
		return m.Depositor
	}
	return ""
}

func (m *MsgDepositRewards) GetAmount() sdk.Coins {
	if m != nil {
		return m.Amount
	}
	return nil
}

// MsgDepositRewardsResponse defines the Msg/DepositRewards response type.
type MsgDepositRewardsResponse struct{}

func (m *MsgDepositRewardsResponse) Reset()         { *m = MsgDepositRewardsResponse{} }
func (m *MsgDepositRewardsResponse) String() string { return proto.CompactTextString(m) }
func (*MsgDepositRewardsResponse) ProtoMessage()    {}

// MsgClaimRewards claims accrued DEX fee rewards for a native DODX staker.
// If Denoms is empty, all pending reward denoms are claimed.
type MsgClaimRewards struct {
	Claimer string   `protobuf:"bytes,1,opt,name=claimer,proto3" json:"claimer,omitempty"`
	Denoms  []string `protobuf:"bytes,2,rep,name=denoms,proto3" json:"denoms,omitempty"`
}

func (m *MsgClaimRewards) Reset()         { *m = MsgClaimRewards{} }
func (m *MsgClaimRewards) String() string { return proto.CompactTextString(m) }
func (*MsgClaimRewards) ProtoMessage()    {}

func (m *MsgClaimRewards) GetClaimer() string {
	if m != nil {
		return m.Claimer
	}
	return ""
}

func (m *MsgClaimRewards) GetDenoms() []string {
	if m != nil {
		return m.Denoms
	}
	return nil
}

// MsgClaimRewardsResponse returns the coins paid to the claimer.
type MsgClaimRewardsResponse struct {
	Amount sdk.Coins `protobuf:"bytes,1,rep,name=amount,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"amount"`
}

func (m *MsgClaimRewardsResponse) Reset()         { *m = MsgClaimRewardsResponse{} }
func (m *MsgClaimRewardsResponse) String() string { return proto.CompactTextString(m) }
func (*MsgClaimRewardsResponse) ProtoMessage()    {}

func (m *MsgClaimRewardsResponse) GetAmount() sdk.Coins {
	if m != nil {
		return m.Amount
	}
	return nil
}

// QueryStakesRequest queries all native DODX stakers.
type QueryStakesRequest struct {
	Pagination *query.PageRequest `protobuf:"bytes,1,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

func (m *QueryStakesRequest) Reset()         { *m = QueryStakesRequest{} }
func (m *QueryStakesRequest) String() string { return proto.CompactTextString(m) }
func (*QueryStakesRequest) ProtoMessage()    {}

// QueryStakesResponse returns native DODX stake records.
type QueryStakesResponse struct {
	Stakes     []StakeRecord       `protobuf:"bytes,1,rep,name=stakes,proto3" json:"stakes"`
	Pagination *query.PageResponse `protobuf:"bytes,2,opt,name=pagination,proto3" json:"pagination,omitempty"`
}

func (m *QueryStakesResponse) Reset()         { *m = QueryStakesResponse{} }
func (m *QueryStakesResponse) String() string { return proto.CompactTextString(m) }
func (*QueryStakesResponse) ProtoMessage()    {}

// QueryPendingRewardsRequest queries claimable DEX fee rewards for one staker.
type QueryPendingRewardsRequest struct {
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

func (m *QueryPendingRewardsRequest) Reset()         { *m = QueryPendingRewardsRequest{} }
func (m *QueryPendingRewardsRequest) String() string { return proto.CompactTextString(m) }
func (*QueryPendingRewardsRequest) ProtoMessage()    {}

// QueryPendingRewardsResponse returns claimable DEX fee rewards.
type QueryPendingRewardsResponse struct {
	Rewards sdk.Coins `protobuf:"bytes,1,rep,name=rewards,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"rewards"`
}

func (m *QueryPendingRewardsResponse) Reset()         { *m = QueryPendingRewardsResponse{} }
func (m *QueryPendingRewardsResponse) String() string { return proto.CompactTextString(m) }
func (*QueryPendingRewardsResponse) ProtoMessage()    {}

// QueryRewardPoolRequest queries the module's accounted reward pools.
type QueryRewardPoolRequest struct{}

func (m *QueryRewardPoolRequest) Reset()         { *m = QueryRewardPoolRequest{} }
func (m *QueryRewardPoolRequest) String() string { return proto.CompactTextString(m) }
func (*QueryRewardPoolRequest) ProtoMessage()    {}

// QueryRewardPoolResponse returns total unclaimed/accounted DEX fee rewards.
type QueryRewardPoolResponse struct {
	Rewards sdk.Coins `protobuf:"bytes,1,rep,name=rewards,proto3,castrepeated=github.com/cosmos/cosmos-sdk/types.Coins" json:"rewards"`
}

func (m *QueryRewardPoolResponse) Reset()         { *m = QueryRewardPoolResponse{} }
func (m *QueryRewardPoolResponse) String() string { return proto.CompactTextString(m) }
func (*QueryRewardPoolResponse) ProtoMessage()    {}

// RewardAmountRecord stores module-level reward accounting values in genesis.
// Amount is an integer string so reward accumulators and pools can be exported
// without implying that accumulator units are bank coins.
type RewardAmountRecord struct {
	Denom  string `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty"`
	Amount string `protobuf:"bytes,2,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (m *RewardAmountRecord) Reset()         { *m = RewardAmountRecord{} }
func (m *RewardAmountRecord) String() string { return proto.CompactTextString(m) }
func (*RewardAmountRecord) ProtoMessage()    {}

// AccountRewardAmountRecord stores per-account reward debt or pending rewards
// in genesis.
type AccountRewardAmountRecord struct {
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Denom   string `protobuf:"bytes,2,opt,name=denom,proto3" json:"denom,omitempty"`
	Amount  string `protobuf:"bytes,3,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (m *AccountRewardAmountRecord) Reset()         { *m = AccountRewardAmountRecord{} }
func (m *AccountRewardAmountRecord) String() string { return proto.CompactTextString(m) }
func (*AccountRewardAmountRecord) ProtoMessage()    {}

func init() {
	proto.RegisterType((*MsgDepositRewards)(nil), "do.dodxstaking.v1beta1.MsgDepositRewards")
	proto.RegisterType((*MsgDepositRewardsResponse)(nil), "do.dodxstaking.v1beta1.MsgDepositRewardsResponse")
	proto.RegisterType((*MsgClaimRewards)(nil), "do.dodxstaking.v1beta1.MsgClaimRewards")
	proto.RegisterType((*MsgClaimRewardsResponse)(nil), "do.dodxstaking.v1beta1.MsgClaimRewardsResponse")
	proto.RegisterType((*QueryStakesRequest)(nil), "do.dodxstaking.v1beta1.QueryStakesRequest")
	proto.RegisterType((*QueryStakesResponse)(nil), "do.dodxstaking.v1beta1.QueryStakesResponse")
	proto.RegisterType((*QueryPendingRewardsRequest)(nil), "do.dodxstaking.v1beta1.QueryPendingRewardsRequest")
	proto.RegisterType((*QueryPendingRewardsResponse)(nil), "do.dodxstaking.v1beta1.QueryPendingRewardsResponse")
	proto.RegisterType((*QueryRewardPoolRequest)(nil), "do.dodxstaking.v1beta1.QueryRewardPoolRequest")
	proto.RegisterType((*QueryRewardPoolResponse)(nil), "do.dodxstaking.v1beta1.QueryRewardPoolResponse")
	proto.RegisterType((*RewardAmountRecord)(nil), "do.dodxstaking.v1beta1.RewardAmountRecord")
	proto.RegisterType((*AccountRewardAmountRecord)(nil), "do.dodxstaking.v1beta1.AccountRewardAmountRecord")
}

func validateRewardDenom(denom string) error {
	if err := sdk.ValidateDenom(denom); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidRewardDenom, err)
	}
	return nil
}
