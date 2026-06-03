package types

import (
	"encoding/base64"
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MemoKey         = "dochain_mfa"
	ApprovalVersion = "dochain-mfa-v1"
)

type MemoEnvelope struct {
	MFA *MemoMFA `json:"dochain_mfa,omitempty"`
}

type MemoMFA struct {
	Approvals []MemoApproval `json:"approvals,omitempty"`
	Enable    *MemoEnable    `json:"enable,omitempty"`
	Disable   *MemoDisable   `json:"disable,omitempty"`
}

type MemoEnable struct {
	Account        string `json:"account"`
	ApprovalPubKey string `json:"approval_pub_key"`
}

type MemoDisable struct {
	Account string `json:"account"`
}

type MemoApproval struct {
	Account   string `json:"account"`
	ExpiresAt int64  `json:"expires_at"`
	Signature string `json:"signature"`
}

type SignerSequence struct {
	Address  string `json:"address"`
	Sequence uint64 `json:"sequence"`
}

type ApprovalPayload struct {
	Version       string           `json:"version"`
	ChainID       string           `json:"chain_id"`
	Account       string           `json:"account"`
	ExpiresAt     int64            `json:"expires_at"`
	TimeoutHeight uint64           `json:"timeout_height"`
	MessagesHash  string           `json:"messages_hash"`
	Signers       []SignerSequence `json:"signers"`
}

func (p ApprovalPayload) SignBytes() []byte {
	bz, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return sdk.MustSortJSON(bz)
}

func EncodeApprovalSignature(signature []byte) string {
	return base64.StdEncoding.EncodeToString(signature)
}

func DecodeApprovalSignature(signature string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(signature)
}

func EncodeApprovalPubKey(pubKey []byte) string {
	return base64.StdEncoding.EncodeToString(pubKey)
}

func DecodeApprovalPubKey(pubKey string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(pubKey)
}
