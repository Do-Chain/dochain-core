package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Policy struct {
	Account        string `json:"account"`
	ApprovalPubKey []byte `json:"approval_pub_key"`
	Enabled        bool   `json:"enabled"`
}

func NewPolicy(account sdk.AccAddress, approvalPubKey []byte) Policy {
	return Policy{
		Account:        account.String(),
		ApprovalPubKey: append([]byte(nil), approvalPubKey...),
		Enabled:        true,
	}
}

func (p Policy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(p.Account); err != nil {
		return fmt.Errorf("invalid mfa account: %w", err)
	}
	if !p.Enabled {
		return nil
	}
	if len(p.ApprovalPubKey) != secp256k1.PubKeySize {
		return fmt.Errorf("invalid mfa approval public key length: %d", len(p.ApprovalPubKey))
	}
	return nil
}

type GenesisState struct {
	Policies []Policy `json:"policies"`
}

func DefaultGenesisState() GenesisState {
	return GenesisState{Policies: []Policy{}}
}

func ValidateGenesis(gs GenesisState) error {
	seen := make(map[string]struct{}, len(gs.Policies))
	for _, policy := range gs.Policies {
		if err := policy.ValidateBasic(); err != nil {
			return err
		}
		if _, ok := seen[policy.Account]; ok {
			return fmt.Errorf("duplicate mfa policy for account %s", policy.Account)
		}
		seen[policy.Account] = struct{}{}
	}
	return nil
}
