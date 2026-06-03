package types

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const RecoveryDelaySeconds int64 = 72 * 60 * 60

type Policy struct {
	Account         string           `json:"account"`
	ApprovalPubKey  []byte           `json:"approval_pub_key"`
	Enabled         bool             `json:"enabled"`
	GuardianAddress string           `json:"guardian_address,omitempty"`
	PendingRecovery *PendingRecovery `json:"pending_recovery,omitempty"`
}

type PendingRecovery struct {
	Action         string `json:"action"`
	ApprovalPubKey []byte `json:"approval_pub_key,omitempty"`
	RequestedAt    int64  `json:"requested_at"`
	ExecuteAfter   int64  `json:"execute_after"`
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
	if p.GuardianAddress != "" {
		if _, err := sdk.AccAddressFromBech32(p.GuardianAddress); err != nil {
			return fmt.Errorf("invalid mfa guardian address: %w", err)
		}
	}
	if p.PendingRecovery != nil {
		if err := p.PendingRecovery.ValidateBasic(); err != nil {
			return err
		}
	}
	return nil
}

func (r PendingRecovery) ValidateBasic() error {
	if !IsRecoveryAction(r.Action) {
		return fmt.Errorf("invalid mfa recovery action: %s", r.Action)
	}
	if r.Action == RecoveryActionRotate && len(r.ApprovalPubKey) != secp256k1.PubKeySize {
		return fmt.Errorf("invalid mfa recovery approval public key length: %d", len(r.ApprovalPubKey))
	}
	if r.Action == RecoveryActionDisable && len(r.ApprovalPubKey) != 0 {
		return fmt.Errorf("disable recovery must not include an approval public key")
	}
	if r.RequestedAt <= 0 {
		return fmt.Errorf("invalid mfa recovery request time: %d", r.RequestedAt)
	}
	if r.ExecuteAfter < r.RequestedAt+RecoveryDelaySeconds {
		return fmt.Errorf("mfa recovery delay is too short")
	}
	return nil
}

func IsRecoveryAction(action string) bool {
	return action == RecoveryActionDisable || action == RecoveryActionRotate
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
