package bindings

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type DoMsg struct {
	Swap               *Swap               `json:"swap,omitempty"`
	SwapSend           *SwapSend           `json:"swap_send,omitempty"`
	DepositDodxRewards *DepositDodxRewards `json:"deposit_dodx_rewards,omitempty"`
}

type Swap struct {
	OfferCoin sdk.Coin `json:"offer_coin"`
	AskDenom  string   `json:"ask_denom"`
}

type SwapSend struct {
	ToAddress string   `json:"to_address"`
	OfferCoin sdk.Coin `json:"offer_coin"`
	AskDenom  string   `json:"ask_denom"`
}

type DepositDodxRewards struct {
	Amount sdk.Coins `json:"amount"`
}
