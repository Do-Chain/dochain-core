package ante

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
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

type MFAControlApplyDecorator struct {
	mfaKeeper MFAKeeper
}

func NewMFARequirementDecorator(cdc codec.Codec, accountKeeper authante.AccountKeeper, mfaKeeper MFAKeeper) MFARequirementDecorator {
	return MFARequirementDecorator{
		cdc:           cdc,
		accountKeeper: accountKeeper,
		mfaKeeper:     mfaKeeper,
	}
}

func NewMFAControlApplyDecorator(mfaKeeper MFAKeeper) MFAControlApplyDecorator {
	return MFAControlApplyDecorator{mfaKeeper: mfaKeeper}
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
		if err := validateControlActionCount(mfaMemo); err != nil {
			return ctx, err
		}
		if err := d.addControlRequirements(ctx, protectedAccounts, mfaMemo); err != nil {
			return ctx, err
		}
		if bypassAccount, ok, err := recoveryBypassAccount(mfaMemo); err != nil {
			return ctx, err
		} else if ok {
			if err := requireRecoveryControlCarrier(tx, bypassAccount); err != nil {
				return ctx, err
			}
			delete(protectedAccounts, bypassAccount.String())
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
		if err := d.verifyApprovals(ctx, tx, requiredAccounts, mfaMemo); err != nil {
			return ctx, err
		}
	} else if memoErr != nil && containsMFAKey(getMemo(tx)) {
		return ctx, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, memoErr.Error())
	}

	if mfaMemo != nil {
		if err := d.verifyInitialEnableApproval(ctx, tx, mfaMemo); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

func (d MFAControlApplyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if d.mfaKeeper == nil {
		return next(ctx, tx, simulate)
	}

	mfaMemo, memoErr := parseMFAMemo(tx)
	if memoErr != nil {
		if containsMFAKey(getMemo(tx)) {
			return ctx, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, memoErr.Error())
		}
		return next(ctx, tx, simulate)
	}
	if mfaMemo == nil {
		return next(ctx, tx, simulate)
	}
	if err := validateControlActionCount(mfaMemo); err != nil {
		return ctx, err
	}
	if err := d.applyControlActions(ctx, tx, mfaMemo); err != nil {
		return ctx, err
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
	if mfaMemo.SetGuardian != nil {
		addr, err := sdk.AccAddressFromBech32(mfaMemo.SetGuardian.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa guardian account: %s", err)
		}
		if _, found := d.mfaKeeper.GetPolicy(ctx, addr); !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot set a guardian for an account without an active mfa policy")
		}
		if err := validateOptionalGuardianAddress(mfaMemo.SetGuardian.GuardianAddress); err != nil {
			return err
		}
		accounts[addr.String()] = addr
	}
	if mfaMemo.RecoveryStart != nil {
		addr, err := sdk.AccAddressFromBech32(mfaMemo.RecoveryStart.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa recovery account: %s", err)
		}
		if _, found := d.mfaKeeper.GetPolicy(ctx, addr); !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot start recovery for an account without an active mfa policy")
		}
		if err := validateRecoveryRequest(mfaMemo.RecoveryStart.Action, mfaMemo.RecoveryStart.ApprovalPubKey); err != nil {
			return err
		}
	}
	if mfaMemo.RecoveryCancel != nil {
		addr, err := sdk.AccAddressFromBech32(mfaMemo.RecoveryCancel.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa recovery account: %s", err)
		}
		if _, found := d.mfaKeeper.GetPolicy(ctx, addr); !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot cancel recovery for an account without an active mfa policy")
		}
	}
	if mfaMemo.RecoveryExecute != nil {
		addr, err := sdk.AccAddressFromBech32(mfaMemo.RecoveryExecute.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa recovery account: %s", err)
		}
		policy, found := d.mfaKeeper.GetPolicy(ctx, addr)
		if !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot execute recovery for an account without an active mfa policy")
		}
		if policy.PendingRecovery == nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "no pending mfa recovery request")
		}
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
	if err := validateOptionalGuardianAddress(mfaMemo.Enable.GuardianAddress); err != nil {
		return err
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

func (d MFARequirementDecorator) verifyApprovals(ctx sdk.Context, tx sdk.Tx, required map[string]sdk.AccAddress, mfaMemo *mfatypes.MemoMFA) error {
	approvalByAccount := make(map[string]mfatypes.MemoApproval, len(mfaMemo.Approvals))
	for _, approval := range mfaMemo.Approvals {
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
		if found {
			if approval.ExpiresAt <= ctx.BlockTime().Unix() {
				return errorsmod.Wrapf(mfatypes.ErrExpiredMFAApproval, "mfa approval for %s expired", account)
			}
			signature, err := mfatypes.DecodeApprovalSignature(approval.Signature)
			if err != nil {
				return errorsmod.Wrapf(mfatypes.ErrInvalidMFAApproval, "invalid mfa signature encoding for %s", account)
			}

			payload := newApprovalPayload(ctx, account, approval.ExpiresAt, timeoutHeight, msgsHash, signers)
			pubKey := secp256k1.PubKey{Key: policy.ApprovalPubKey}
			if pubKey.VerifySignature(payload.SignBytes(), signature) {
				continue
			}
			return errorsmod.Wrapf(mfatypes.ErrInvalidMFAApproval, "mfa signature does not match policy for %s", account)
		}

		guardianOK, err := verifyGuardianApproval(ctx, account, policy, mfaMemo, timeoutHeight, msgsHash, signers)
		if err != nil {
			return err
		}
		if guardianOK {
			continue
		}

		return errorsmod.Wrapf(mfatypes.ErrMFARequired, "missing mfa approval for %s", account)
	}

	return nil
}

func verifyGuardianApproval(ctx sdk.Context, account string, policy mfatypes.Policy, mfaMemo *mfatypes.MemoMFA, timeoutHeight uint64, msgsHash string, signers []mfatypes.SignerSequence) (bool, error) {
	if mfaMemo.GuardianApproval == nil {
		return false, nil
	}
	if policy.GuardianAddress == "" {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "account has no mfa guardian")
	}
	action, approvalPubKey, ok := guardianControlAction(mfaMemo)
	if !ok {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "guardian approval can only be used for disable or rotate")
	}

	approval := mfaMemo.GuardianApproval
	if approval.Account != account {
		return false, nil
	}
	if approval.GuardianAddress != policy.GuardianAddress {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "guardian approval does not match policy guardian")
	}
	if approval.Action != action || approval.ApprovalPubKey != approvalPubKey {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "guardian approval does not match requested mfa action")
	}
	if approval.ExpiresAt <= ctx.BlockTime().Unix() {
		return false, errorsmod.Wrapf(mfatypes.ErrExpiredMFAApproval, "guardian approval for %s expired", account)
	}

	guardianPubKey, err := mfatypes.DecodeApprovalPubKey(approval.GuardianPubKey)
	if err != nil {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "invalid guardian public key encoding")
	}
	pubKey := secp256k1.PubKey{Key: guardianPubKey}
	guardianAddress := sdk.AccAddress(pubKey.Address()).String()
	if guardianAddress != policy.GuardianAddress {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "guardian public key does not match policy guardian address")
	}

	signature, err := mfatypes.DecodeApprovalSignature(approval.Signature)
	if err != nil {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "invalid guardian signature encoding")
	}
	payload := newGuardianApprovalPayload(ctx, account, policy.GuardianAddress, action, approvalPubKey, approval.ExpiresAt, timeoutHeight, msgsHash, signers)
	if !pubKey.VerifySignature(payload.SignBytes(), signature) {
		return false, errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "guardian signature does not match requested mfa action")
	}
	return true, nil
}

func guardianControlAction(mfaMemo *mfatypes.MemoMFA) (string, string, bool) {
	if mfaMemo.Disable != nil {
		return mfatypes.RecoveryActionDisable, "", true
	}
	if mfaMemo.Enable != nil {
		return mfatypes.RecoveryActionRotate, mfaMemo.Enable.ApprovalPubKey, true
	}
	return "", "", false
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

func (d MFAControlApplyDecorator) applyControlActions(ctx sdk.Context, tx sdk.Tx, mfaMemo *mfatypes.MemoMFA) error {
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
		guardianAddress := mfaMemo.Enable.GuardianAddress
		if existing, found := d.mfaKeeper.GetPolicy(ctx, account); found && guardianAddress == "" {
			guardianAddress = existing.GuardianAddress
		}
		if err := validateOptionalGuardianAddress(guardianAddress); err != nil {
			return err
		}
		policy := mfatypes.NewPolicy(account, pubKey)
		policy.GuardianAddress = guardianAddress
		if err := d.mfaKeeper.SetPolicy(ctx, policy); err != nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, err.Error())
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			mfatypes.ModuleName,
			sdk.NewAttribute("action", "enable"),
			sdk.NewAttribute("account", account.String()),
			sdk.NewAttribute("guardian", guardianAddress),
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

	if mfaMemo.SetGuardian != nil {
		account, err := sdk.AccAddressFromBech32(mfaMemo.SetGuardian.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa guardian account: %s", err)
		}
		if err := requireTxSigner(tx, account); err != nil {
			return err
		}
		guardianAddress := mfaMemo.SetGuardian.GuardianAddress
		if err := validateOptionalGuardianAddress(guardianAddress); err != nil {
			return err
		}
		policy, found := d.mfaKeeper.GetPolicy(ctx, account)
		if !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot set a guardian for an account without an active mfa policy")
		}
		policy.GuardianAddress = guardianAddress
		if err := d.mfaKeeper.SetPolicy(ctx, policy); err != nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, err.Error())
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			mfatypes.ModuleName,
			sdk.NewAttribute("action", "set_guardian"),
			sdk.NewAttribute("account", account.String()),
			sdk.NewAttribute("guardian", guardianAddress),
		))
	}

	if mfaMemo.RecoveryStart != nil {
		account, err := sdk.AccAddressFromBech32(mfaMemo.RecoveryStart.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa recovery account: %s", err)
		}
		if err := requireTxSigner(tx, account); err != nil {
			return err
		}
		policy, found := d.mfaKeeper.GetPolicy(ctx, account)
		if !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot start recovery for an account without an active mfa policy")
		}
		approvalPubKey, err := recoveryApprovalPubKey(mfaMemo.RecoveryStart.Action, mfaMemo.RecoveryStart.ApprovalPubKey)
		if err != nil {
			return err
		}
		requestedAt := ctx.BlockTime().Unix()
		policy.PendingRecovery = &mfatypes.PendingRecovery{
			Action:         mfaMemo.RecoveryStart.Action,
			ApprovalPubKey: approvalPubKey,
			RequestedAt:    requestedAt,
			ExecuteAfter:   requestedAt + mfatypes.RecoveryDelaySeconds,
		}
		if err := d.mfaKeeper.SetPolicy(ctx, policy); err != nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, err.Error())
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			mfatypes.ModuleName,
			sdk.NewAttribute("action", "recovery_start"),
			sdk.NewAttribute("account", account.String()),
			sdk.NewAttribute("recovery_action", mfaMemo.RecoveryStart.Action),
			sdk.NewAttribute("execute_after", fmt.Sprintf("%d", requestedAt+mfatypes.RecoveryDelaySeconds)),
		))
	}

	if mfaMemo.RecoveryCancel != nil {
		account, err := sdk.AccAddressFromBech32(mfaMemo.RecoveryCancel.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa recovery account: %s", err)
		}
		if err := requireTxSigner(tx, account); err != nil {
			return err
		}
		policy, found := d.mfaKeeper.GetPolicy(ctx, account)
		if !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot cancel recovery for an account without an active mfa policy")
		}
		policy.PendingRecovery = nil
		if err := d.mfaKeeper.SetPolicy(ctx, policy); err != nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, err.Error())
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			mfatypes.ModuleName,
			sdk.NewAttribute("action", "recovery_cancel"),
			sdk.NewAttribute("account", account.String()),
		))
	}

	if mfaMemo.RecoveryExecute != nil {
		account, err := sdk.AccAddressFromBech32(mfaMemo.RecoveryExecute.Account)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa recovery account: %s", err)
		}
		if err := requireTxSigner(tx, account); err != nil {
			return err
		}
		policy, found := d.mfaKeeper.GetPolicy(ctx, account)
		if !found {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "cannot execute recovery for an account without an active mfa policy")
		}
		if policy.PendingRecovery == nil {
			return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "no pending mfa recovery request")
		}
		if policy.PendingRecovery.ExecuteAfter > ctx.BlockTime().Unix() {
			return errorsmod.Wrapf(mfatypes.ErrMFARequired, "mfa recovery cannot execute before %d", policy.PendingRecovery.ExecuteAfter)
		}
		switch policy.PendingRecovery.Action {
		case mfatypes.RecoveryActionDisable:
			d.mfaKeeper.DeletePolicy(ctx, account)
		case mfatypes.RecoveryActionRotate:
			newPolicy := mfatypes.NewPolicy(account, policy.PendingRecovery.ApprovalPubKey)
			newPolicy.GuardianAddress = policy.GuardianAddress
			if err := d.mfaKeeper.SetPolicy(ctx, newPolicy); err != nil {
				return errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, err.Error())
			}
		default:
			return errorsmod.Wrapf(mfatypes.ErrInvalidMFAPolicy, "invalid mfa recovery action: %s", policy.PendingRecovery.Action)
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(
			mfatypes.ModuleName,
			sdk.NewAttribute("action", "recovery_execute"),
			sdk.NewAttribute("account", account.String()),
			sdk.NewAttribute("recovery_action", policy.PendingRecovery.Action),
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

func validateControlActionCount(mfaMemo *mfatypes.MemoMFA) error {
	actions := 0
	if mfaMemo.Enable != nil {
		actions++
	}
	if mfaMemo.Disable != nil {
		actions++
	}
	if mfaMemo.SetGuardian != nil {
		actions++
	}
	if mfaMemo.RecoveryStart != nil {
		actions++
	}
	if mfaMemo.RecoveryCancel != nil {
		actions++
	}
	if mfaMemo.RecoveryExecute != nil {
		actions++
	}
	if actions > 1 {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "memo can only contain one mfa control action")
	}
	return nil
}

func recoveryBypassAccount(mfaMemo *mfatypes.MemoMFA) (sdk.AccAddress, bool, error) {
	var account string
	switch {
	case mfaMemo.RecoveryStart != nil:
		account = mfaMemo.RecoveryStart.Account
	case mfaMemo.RecoveryCancel != nil:
		account = mfaMemo.RecoveryCancel.Account
	case mfaMemo.RecoveryExecute != nil:
		account = mfaMemo.RecoveryExecute.Account
	default:
		return nil, false, nil
	}

	addr, err := sdk.AccAddressFromBech32(account)
	if err != nil {
		return nil, false, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa recovery account: %s", err)
	}
	return addr, true, nil
}

func requireRecoveryControlCarrier(tx sdk.Tx, account sdk.AccAddress) error {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return nil
	}
	if len(msgs) != 1 {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "mfa recovery transaction cannot include other messages")
	}

	msg, ok := msgs[0].(*banktypes.MsgSend)
	if !ok {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "mfa recovery transaction must use the wallet self-send control message")
	}
	if msg.FromAddress != account.String() || msg.ToAddress != account.String() {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "mfa recovery control transaction must be a self-send")
	}
	if !msg.Amount.AmountOf(core.MicroDoDenom).Equal(sdkmath.OneInt()) || !msg.Amount.Equal(sdk.NewCoins(sdk.NewCoin(core.MicroDoDenom, sdkmath.OneInt()))) {
		return errorsmod.Wrap(mfatypes.ErrInvalidMFAApproval, "mfa recovery control transaction must send exactly 1 udo to self")
	}
	return nil
}

func validateOptionalGuardianAddress(address string) error {
	if address == "" {
		return nil
	}
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid mfa guardian address: %s", err)
	}
	return nil
}

func validateRecoveryRequest(action, approvalPubKey string) error {
	if !mfatypes.IsRecoveryAction(action) {
		return errorsmod.Wrapf(mfatypes.ErrInvalidMFAPolicy, "invalid mfa recovery action: %s", action)
	}
	if _, err := recoveryApprovalPubKey(action, approvalPubKey); err != nil {
		return err
	}
	return nil
}

func recoveryApprovalPubKey(action, encodedPubKey string) ([]byte, error) {
	if action == mfatypes.RecoveryActionDisable {
		if encodedPubKey != "" {
			return nil, errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "disable recovery must not include an approval public key")
		}
		return nil, nil
	}
	pubKey, err := mfatypes.DecodeApprovalPubKey(encodedPubKey)
	if err != nil {
		return nil, errorsmod.Wrap(mfatypes.ErrInvalidMFAPolicy, "invalid recovery approval public key encoding")
	}
	if len(pubKey) != secp256k1.PubKeySize {
		return nil, errorsmod.Wrapf(mfatypes.ErrInvalidMFAPolicy, "invalid recovery approval public key length: %d", len(pubKey))
	}
	return pubKey, nil
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

func BuildGuardianApprovalPayload(ctx sdk.Context, tx sdk.Tx, cdc codec.Codec, account sdk.AccAddress, guardianAddress, action, approvalPubKey string, expiresAt int64, signers []mfatypes.SignerSequence) (mfatypes.GuardianApprovalPayload, error) {
	hash, err := messagesHash(cdc, tx.GetMsgs())
	if err != nil {
		return mfatypes.GuardianApprovalPayload{}, err
	}
	return newGuardianApprovalPayload(ctx, account.String(), guardianAddress, action, approvalPubKey, expiresAt, getTimeoutHeight(tx), hash, signers), nil
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

func newGuardianApprovalPayload(ctx sdk.Context, account, guardianAddress, action, approvalPubKey string, expiresAt int64, timeoutHeight uint64, messagesHash string, signers []mfatypes.SignerSequence) mfatypes.GuardianApprovalPayload {
	return mfatypes.GuardianApprovalPayload{
		Version:         mfatypes.GuardianApprovalVersion,
		ChainID:         ctx.ChainID(),
		Account:         account,
		GuardianAddress: guardianAddress,
		Action:          action,
		ApprovalPubKey:  approvalPubKey,
		ExpiresAt:       expiresAt,
		TimeoutHeight:   timeoutHeight,
		MessagesHash:    messagesHash,
		Signers:         signers,
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
