package ante

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	legacywasmtypes "github.com/Daviddochain/dochain-core/v4/custom/wasm/types/legacy"
	core "github.com/Daviddochain/dochain-core/v4/types"
	markettypes "github.com/Daviddochain/dochain-core/v4/x/market/types"
	mfatypes "github.com/Daviddochain/dochain-core/v4/x/mfa/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
)

type MFAKeeper interface {
	GetPolicy(ctx sdk.Context, account sdk.AccAddress) (mfatypes.Policy, bool)
	SetPolicy(ctx sdk.Context, policy mfatypes.Policy) error
	DeletePolicy(ctx sdk.Context, account sdk.AccAddress)
}

type MFARequirementDecorator struct {
	cdc           codec.Codec
	accountKeeper authante.AccountKeeper
	mfaKeeper     MFAKeeper
}

func NewMFARequirementDecorator(cdc codec.Codec, accountKeeper authante.AccountKeeper, mfaKeeper MFAKeeper) MFARequirementDecorator {
	return MFARequirementDecorator{
		cdc:           cdc,
		accountKeeper: accountKeeper,
		mfaKeeper:     mfaKeeper,
	}
}

func (d MFARequirementDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if d.mfaKeeper == nil {
		return next(ctx, tx, simulate)
	}

	mfaMemo, memoErr := parseMFAMemo(tx)
	protectedAccounts, err := d.protectedAccounts(tx.GetMsgs())
	if err != nil {
		return ctx, err
	}

	if mfaMemo != nil {
		if mfaMemo.Enable != nil && mfaMemo.Disable != nil {
			return ctx, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "memo cannot enable and disable mfa in the same transaction")
		}
		if err := d.addControlRequirements(ctx, protectedAccounts, mfaMemo); err != nil {
			return ctx, err
		}
	}

	requiredAccounts := d.accountsWithPolicy(ctx, protectedAccounts)
	if len(requiredAccounts) > 0 {
		if memoErr != nil {
			return ctx, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, memoErr.Error())
		}
		if mfaMemo == nil {
			return ctx, errorsmod.Wrap(mfatypes.ErrMFARequired, "missing mfa memo")
		}
		if err := d.verifyApprovals(ctx, tx, requiredAccounts, mfaMemo.Approvals); err != nil {
			return ctx, err
		}
	} else if memoErr != nil && containsMFAKey(getMemo(tx)) {
		return ctx, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, memoErr.Error())
	}

	if mfaMemo != nil {
		if err := d.verifyInitialEnableApproval(ctx, tx, mfaMemo); err != nil {
			return ctx, err
		}
		if err := d.applyControlActions(ctx, tx, mfaMemo); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

func (d MFARequirementDecorator) addControlRequirements(ctx sdk.Context, accounts map[string]sdk.AccAddress, mfaMemo *mfatypes.MemoMFA) error {
	if mfaMemo.Enable != nil {
		addr, err := sdk.AccAddressFromBech32(mfaMemo.Enable.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa enable account: %s", err)
		}
		if _, found := d.mfaKeeper.GetPolicy(ctx, addr); found {
			accounts[addr.String()] = addr
		}
	}
	if mfaMemo.Disable != nil {
		addr, err := sdk.AccAddressFromBech32(mfaMemo.Disable.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa disable account: %s", err)
		}
		if _, found := d.mfaKeeper.GetPolicy(ctx, addr); !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot disable mfa for an account without an active mfa policy")
		}
		accounts[addr.String()] = addr
	}
	return nil
}

func (d MFARequirementDecorator) accountsWithPolicy(ctx sdk.Context, accounts map[string]sdk.AccAddress) map[string]sdk.AccAddress {
	required := make(map[string]sdk.AccAddress)
	for key, account := range accounts {
		if _, found := d.mfaKeeper.GetPolicy(ctx, account); found {
			required[key] = account
		}
	}
	return required
}

func (d MFARequirementDecorator) verifyInitialEnableApproval(ctx sdk.Context, tx sdk.Tx, mfaMemo *mfatypes.MemoMFA) error {
	if mfaMemo.Enable == nil {
		return nil
	}
	account, err := sdk.AccAddressFromBech32(mfaMemo.Enable.Account)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa enable account: %s", err)
	}
	if _, found := d.mfaKeeper.GetPolicy(ctx, account); found {
		return nil
	}
	if err := requireTxSigner(tx, account); err != nil {
		return err
	}
	pubKey, err := mfatypes.DecodeApprovalPubKey(mfaMemo.Enable.ApprovalPubKey)
	if err != nil {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "invalid mfa approval public key encoding")
	}
	if err := mfatypes.NewPolicy(account, pubKey).ValidateBasic(); err != nil {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, err.Error())
	}

	approval, found := approvalForAccount(mfaMemo.Approvals, account.String())
	if !found {
		return errorsmod.Wrap(mfatypes.ErrMFARequired, "initial mfa enable requires an approval from the new mfa key")
	}
	if approval.ExpiresAt <= ctx.BlockTime().Unix() {
		return errorsmod.Wrapf(mfatypes.ErrExpiredMFAApproval, "mfa enable approval for %s expired", account.String())
	}
	signature, err := mfatypes.DecodeApprovalSignature(approval.Signature)
	if err != nil {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "invalid mfa enable signature encoding")
	}
	payload, err := d.approvalPayload(ctx, tx, account.String(), approval.ExpiresAt)
	if err != nil {
		return err
	}
	if !(&secp256k1.PubKey{Key: pubKey}).VerifySignature(payload.SignBytes(), signature) {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "mfa enable signature does not match the new policy key")
	}
	return nil
}

func (d MFARequirementDecorator) verifyApprovals(ctx sdk.Context, tx sdk.Tx, required map[string]sdk.AccAddress, approvals []mfatypes.MemoApproval) error {
	if len(approvals) == 0 {
		return errorsmod.Wrap(mfatypes.ErrMFARequired, "missing mfa approvals")
	}

	approvalByAccount := make(map[string]mfatypes.MemoApproval, len(approvals))
	for _, approval := range approvals {
		approvalByAccount[approval.Account] = approval
	}

	msgsHash, err := d.messagesHash(tx.GetMsgs())
	if err != nil {
		return err
	}
	timeoutHeight := getTimeoutHeight(tx)
	signers, err := d.signerSequences(ctx, tx)
	if err != nil {
		return err
	}

	for account, addr := range required {
		policy, found := d.mfaKeeper.GetPolicy(ctx, addr)
		if !found {
			continue
		}
		approval, found := approvalByAccount[account]
		if !found {
			return errorsmod.Wrapf(mfatypes.ErrMFARequired, "missing mfa approval for %s", account)
		}
		if approval.ExpiresAt <= ctx.BlockTime().Unix() {
			return errorsmod.Wrapf(mfatypes.ErrExpiredMFAApproval, "mfa approval for %s expired", account)
		}
		signature, err := mfatypes.DecodeApprovalSignature(approval.Signature)
		if err != nil {
			return errorsmod.Wrapf(mfatypes.ErrInvalidMFAApproval, "invalid mfa signature encoding for %s", account)
		}

		payload := newApprovalPayload(ctx, account, approval.ExpiresAt, timeoutHeight, msgsHash, signers)
		pubKey := secp256k1.PubKey{Key: policy.ApprovalPubKey}
		if !pubKey.VerifySignature(payload.SignBytes(), signature) {
			return errorsmod.Wrapf(mfatypes.ErrInvalidMFAApproval, "mfa signature does not match policy for %s", account)
		}
	}

	return nil
}

func (d MFARequirementDecorator) approvalPayload(ctx sdk.Context, tx sdk.Tx, account string, expiresAt int64) (mfatypes.ApprovalPayload, error) {
	msgsHash, err := d.messagesHash(tx.GetMsgs())
	if err != nil {
		return mfatypes.ApprovalPayload{}, err
	}
	signers, err := d.signerSequences(ctx, tx)
	if err != nil {
		return mfatypes.ApprovalPayload{}, err
	}
	return newApprovalPayload(ctx, account, expiresAt, getTimeoutHeight(tx), msgsHash, signers), nil
}

func (d MFARequirementDecorator) applyControlActions(ctx sdk.Context, tx sdk.Tx, mfaMemo *mfatypes.MemoMFA) error {
	if mfaMemo.Enable != nil {
		account, err := sdk.AccAddressFromBech32(mfaMemo.Enable.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa enable account: %s", err)
		}
		if err := requireTxSigner(tx, account); err != nil {
			return err
		}
		pubKey, err := mfatypes.DecodeApprovalPubKey(mfaMemo.Enable.ApprovalPubKey)
		if err != nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "invalid mfa approval public key encoding")
		}
		if err := d.mfaKeeper.SetPolicy(ctx, mfatypes.NewPolicy(account, pubKey)); err != nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, err.Error())
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			mfatypes.ModuleName,
			sdk.NewAttribute("action", "enable"),
			sdk.NewAttribute("account", account.String()),
		))
	}

	if mfaMemo.Disable != nil {
		account, err := sdk.AccAddressFromBech32(mfaMemo.Disable.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa disable account: %s", err)
		}
		if err := requireTxSigner(tx, account); err != nil {
			return err
		}
		d.mfaKeeper.DeletePolicy(ctx, account)
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			mfatypes.ModuleName,
			sdk.NewAttribute("action", "disable"),
			sdk.NewAttribute("account", account.String()),
		))
	}

	return nil
}

func approvalForAccount(approvals []mfatypes.MemoApproval, account string) (mfatypes.MemoApproval, bool) {
	for _, approval := range approvals {
		if approval.Account == account {
			return approval, true
		}
	}
	return mfatypes.MemoApproval{}, false
}

func (d MFARequirementDecorator) protectedAccounts(msgs []sdk.Msg) (map[string]sdk.AccAddress, error) {
	accounts := make(map[string]sdk.AccAddress)
	for _, msg := range msgs {
		if err := d.addProtectedAccounts(accounts, msg); err != nil {
			return nil, err
		}
	}
	return accounts, nil
}

func (d MFARequirementDecorator) addProtectedAccounts(accounts map[string]sdk.AccAddress, msg sdk.Msg) error {
	switch msg := msg.(type) {
	case *banktypes.MsgSend:
		if containsDo(msg.Amount) {
			return addAccount(accounts, msg.FromAddress)
		}
	case *banktypes.MsgMultiSend:
		for _, input := range msg.Inputs {
			if containsDo(input.Coins) {
				if err := addAccount(accounts, input.Address); err != nil {
					return err
				}
			}
		}
	case *stakingtypes.MsgUndelegate:
		if msg.Amount.Denom == core.MicroDoDenom {
			return addAccount(accounts, msg.DelegatorAddress)
		}
	case *stakingtypes.MsgBeginRedelegate:
		if msg.Amount.Denom == core.MicroDoDenom {
			return addAccount(accounts, msg.DelegatorAddress)
		}
	case *ibctransfertypes.MsgTransfer:
		if msg.Token.Denom == core.MicroDoDenom {
			return addAccount(accounts, msg.Sender)
		}
	case *wasmtypes.MsgExecuteContract:
		if containsDo(msg.Funds) {
			return addAccount(accounts, msg.Sender)
		}
	case *legacywasmtypes.MsgExecuteContract:
		if containsDo(msg.Coins) {
			return addAccount(accounts, msg.Sender)
		}
	case *markettypes.MsgSwapSend:
		if msg.OfferCoin.Denom == core.MicroDoDenom {
			return addAccount(accounts, msg.FromAddress)
		}
	case *markettypes.MsgSwap:
		if msg.OfferCoin.Denom == core.MicroDoDenom {
			return addAccount(accounts, msg.Trader)
		}
	case *authz.MsgExec:
		innerMsgs, err := msg.GetMessages()
		if err != nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, err.Error())
		}
		for _, innerMsg := range innerMsgs {
			if err := d.addProtectedAccounts(accounts, innerMsg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d MFARequirementDecorator) messagesHash(msgs []sdk.Msg) (string, error) {
	h := sha256.New()
	for _, msg := range msgs {
		bz, err := d.cdc.MarshalInterfaceJSON(msg)
		if err != nil {
			return "", errorsmod.Wrapf(mfatypes.ErrInvalidMFAApproval, "cannot hash msg %s: %s", sdk.MsgTypeURL(msg), err)
		}
		h.Write(sdk.MustSortJSON(bz))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (d MFARequirementDecorator) signerSequences(ctx sdk.Context, tx sdk.Tx) ([]mfatypes.SignerSequence, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return nil, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "transaction does not expose signer data")
	}
	signers, err := sigTx.GetSigners()
	if err != nil {
		return nil, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, err.Error())
	}
	result := make([]mfatypes.SignerSequence, 0, len(signers))
	for _, signer := range signers {
		addr := sdk.AccAddress(signer)
		acc := d.accountKeeper.GetAccount(ctx, addr)
		sequence := uint64(0)
		if acc != nil {
			sequence = acc.GetSequence()
		}
		result = append(result, mfatypes.SignerSequence{
			Address:  addr.String(),
			Sequence: sequence,
		})
	}
	return result, nil
}

func parseMFAMemo(tx sdk.Tx) (*mfatypes.MemoMFA, error) {
	memo := getMemo(tx)
	if memo == "" {
		return nil, nil
	}
	var envelope mfatypes.MemoEnvelope
	if err := json.Unmarshal([]byte(memo), &envelope); err != nil {
		return nil, err
	}
	return envelope.MFA, nil
}

func getMemo(tx sdk.Tx) string {
	txWithMemo, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return ""
	}
	return txWithMemo.GetMemo()
}

func getTimeoutHeight(tx sdk.Tx) uint64 {
	txWithTimeout, ok := tx.(sdk.TxWithTimeoutHeight)
	if !ok {
		return 0
	}
	return txWithTimeout.GetTimeoutHeight()
}

func containsMFAKey(memo string) bool {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(memo), &envelope); err != nil {
		return false
	}
	_, ok := envelope[mfatypes.MemoKey]
	return ok
}

func containsDo(coins sdk.Coins) bool {
	return coins.AmountOf(core.MicroDoDenom).IsPositive()
}

func addAccount(accounts map[string]sdk.AccAddress, address string) error {
	account, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid protected account address %s: %s", address, err)
	}
	accounts[account.String()] = account
	return nil
}

func requireTxSigner(tx sdk.Tx, account sdk.AccAddress) error {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "transaction does not expose signer data")
	}
	signers, err := sigTx.GetSigners()
	if err != nil {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, err.Error())
	}
	for _, signer := range signers {
		if account.Equals(sdk.AccAddress(signer)) {
			return nil
		}
	}
	return errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "mfa control account %s must sign the transaction", account.String())
}

func BuildApprovalPayload(ctx sdk.Context, tx sdk.Tx, cdc codec.Codec, account sdk.AccAddress, expiresAt int64, signers []mfatypes.SignerSequence) (mfatypes.ApprovalPayload, error) {
	hash, err := messagesHash(cdc, tx.GetMsgs())
	if err != nil {
		return mfatypes.ApprovalPayload{}, err
	}
	return newApprovalPayload(ctx, account.String(), expiresAt, getTimeoutHeight(tx), hash, signers), nil
}

func newApprovalPayload(ctx sdk.Context, account string, expiresAt int64, timeoutHeight uint64, messagesHash string, signers []mfatypes.SignerSequence) mfatypes.ApprovalPayload {
	return mfatypes.ApprovalPayload{
		Version:       mfatypes.ApprovalVersion,
		ChainID:       ctx.ChainID(),
		Account:       account,
		ExpiresAt:     expiresAt,
		TimeoutHeight: timeoutHeight,
		MessagesHash:  messagesHash,
		Signers:       signers,
	}
}

func messagesHash(cdc codec.Codec, msgs []sdk.Msg) (string, error) {
	h := sha256.New()
	for _, msg := range msgs {
		bz, err := cdc.MarshalInterfaceJSON(msg)
		if err != nil {
			return "", fmt.Errorf("cannot hash msg %s: %w", sdk.MsgTypeURL(msg), err)
		}
		h.Write(sdk.MustSortJSON(bz))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
