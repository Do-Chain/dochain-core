package ante_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	coreaddress "cosmossdk.io/core/address"
	sdklog "cosmossdk.io/log"
	store "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	core "github.com/Daviddochain/dochain-core/v4/types"
	mfaante "github.com/Daviddochain/dochain-core/v4/x/mfa/ante"
	mfakeeper "github.com/Daviddochain/dochain-core/v4/x/mfa/keeper"
	mfatypes "github.com/Daviddochain/dochain-core/v4/x/mfa/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdksigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"
)

type fixture struct {
	ctx      sdk.Context
	cdc      codec.Codec
	keeper   mfakeeper.Keeper
	accounts *mockAccountKeeper
	handler  sdk.AnteHandler
}

func newFixture(t *testing.T) fixture {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(mfatypes.StoreKey)
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, sdklog.NewNopLogger(), storemetrics.NewNoOpMetrics())
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, cms.LoadLatestVersion())

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	ctx := sdk.NewContext(
		cms,
		tmproto.Header{ChainID: "dochain-1", Time: time.Unix(1_700_000_000, 0)},
		false,
		sdklog.NewNopLogger(),
	)
	keeper := mfakeeper.NewKeeper(cdc, storeKey)
	accounts := newMockAccountKeeper()

	return fixture{
		ctx:      ctx,
		cdc:      cdc,
		keeper:   keeper,
		accounts: accounts,
		handler: sdk.ChainAnteDecorators(
			mfaante.NewMFARequirementDecorator(cdc, accounts, keeper),
			mfaante.NewMFAControlApplyDecorator(keeper),
		),
	}
}

func TestProtectedSendRequiresMFA(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 7)
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

	tx := newTestTx([]sdk.Msg{banktypes.NewMsgSend(account, sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1)))}, "", []sdk.AccAddress{account})

	_, err := fx.handler(fx.ctx, tx, false)
	require.ErrorIs(t, err, mfatypes.ErrMFARequired)
}

func TestCorruptPolicyFailsClosed(t *testing.T) {
	fx := newFixture(t)
	account, _, _ := fx.addAccount(t, 7)
	fx.ctx.KVStore(fx.keeper.StoreKey()).Set(mfatypes.PolicyKey(account.String()), []byte("{not-json"))

	_, found := fx.keeper.GetPolicy(fx.ctx, account)
	require.True(t, found)

	tx := newTestTx([]sdk.Msg{banktypes.NewMsgSend(account, sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1)))}, "", []sdk.AccAddress{account})

	_, err := fx.handler(fx.ctx, tx, false)
	require.ErrorIs(t, err, mfatypes.ErrInvalidMFAPolicy)
}

func TestProtectedSendAllowsValidMFA(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 7)
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

	tx := newTestTx([]sdk.Msg{banktypes.NewMsgSend(account, sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1)))}, "", []sdk.AccAddress{account})
	tx.memo = signApprovalMemo(t, fx.ctx, fx.cdc, tx, account, mfaPriv, 1_700_000_300, 7)

	_, err := fx.handler(fx.ctx, tx, false)
	require.NoError(t, err)
}

func TestProtectedSendRejectsApprovalForDifferentMessage(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 7)
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

	signedTx := newTestTx([]sdk.Msg{banktypes.NewMsgSend(account, sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1)))}, "", []sdk.AccAddress{account})
	memo := signApprovalMemo(t, fx.ctx, fx.cdc, signedTx, account, mfaPriv, 1_700_000_300, 7)

	changedTx := newTestTx([]sdk.Msg{banktypes.NewMsgSend(account, sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 2)))}, memo, []sdk.AccAddress{account})
	_, err := fx.handler(fx.ctx, changedTx, false)
	require.ErrorIs(t, err, mfatypes.ErrInvalidMFAApproval)
}

func TestEnableAndDisableControlMemo(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 3)

	enableTx := newTestTx(nil, "", []sdk.AccAddress{account})
	enableTx.memo = controlMemo(t, &mfatypes.MemoMFA{
		Approvals: []mfatypes.MemoApproval{approval(t, fx.ctx, fx.cdc, enableTx, account, mfaPriv, 1_700_000_300, 3)},
		Enable: &mfatypes.MemoEnable{
			Account:        account.String(),
			ApprovalPubKey: mfatypes.EncodeApprovalPubKey(mfaPriv.PubKey().Bytes()),
		},
	})
	_, err := fx.handler(fx.ctx, enableTx, false)
	require.NoError(t, err)
	require.True(t, fx.keeper.HasPolicy(fx.ctx, account))

	disableTx := newTestTx(nil, "", []sdk.AccAddress{account})
	disableTx.memo = controlMemo(t, &mfatypes.MemoMFA{
		Approvals: []mfatypes.MemoApproval{approval(t, fx.ctx, fx.cdc, disableTx, account, mfaPriv, 1_700_000_300, 3)},
		Disable:   &mfatypes.MemoDisable{Account: account.String()},
	})
	_, err = fx.handler(fx.ctx, disableTx, false)
	require.NoError(t, err)
	require.False(t, fx.keeper.HasPolicy(fx.ctx, account))
}

func TestEnableRequiresInitialMFAApproval(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 3)

	enableTx := newTestTx(nil, "", []sdk.AccAddress{account})
	enableTx.memo = controlMemo(t, &mfatypes.MemoMFA{
		Enable: &mfatypes.MemoEnable{
			Account:        account.String(),
			ApprovalPubKey: mfatypes.EncodeApprovalPubKey(mfaPriv.PubKey().Bytes()),
		},
	})

	_, err := fx.handler(fx.ctx, enableTx, false)
	require.ErrorIs(t, err, mfatypes.ErrMFARequired)
	require.False(t, fx.keeper.HasPolicy(fx.ctx, account))
}

func TestSetGuardianRequiresMFAApproval(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 4)
	guardian, _, _ := fx.addAccount(t, 1)
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

	tx := newTestTx(nil, "", []sdk.AccAddress{account})
	tx.memo = controlMemo(t, &mfatypes.MemoMFA{
		Approvals: []mfatypes.MemoApproval{approval(t, fx.ctx, fx.cdc, tx, account, mfaPriv, 1_700_000_300, 4)},
		SetGuardian: &mfatypes.MemoSetGuardian{
			Account:         account.String(),
			GuardianAddress: guardian.String(),
		},
	})

	_, err := fx.handler(fx.ctx, tx, false)
	require.NoError(t, err)
	policy, found := fx.keeper.GetPolicy(fx.ctx, account)
	require.True(t, found)
	require.Equal(t, guardian.String(), policy.GuardianAddress)
}

func TestGuardianApprovalCanDisableMFA(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 4)
	guardian, guardianPriv, _ := fx.addAccount(t, 1)
	policy := mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())
	policy.GuardianAddress = guardian.String()
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, policy))

	tx := newTestTx(nil, "", []sdk.AccAddress{account})
	tx.memo = controlMemo(t, &mfatypes.MemoMFA{
		Disable:          &mfatypes.MemoDisable{Account: account.String()},
		GuardianApproval: guardianApproval(t, fx.ctx, fx.cdc, tx, account, guardian, guardianPriv, mfatypes.RecoveryActionDisable, "", 1_700_000_300, 4),
	})

	_, err := fx.handler(fx.ctx, tx, false)
	require.NoError(t, err)
	require.False(t, fx.keeper.HasPolicy(fx.ctx, account))
}

func TestGuardianApprovalCannotApproveProtectedSend(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 4)
	guardian, guardianPriv, _ := fx.addAccount(t, 1)
	policy := mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())
	policy.GuardianAddress = guardian.String()
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, policy))

	tx := newTestTx([]sdk.Msg{
		banktypes.NewMsgSend(account, sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1))),
	}, "", []sdk.AccAddress{account})
	tx.memo = controlMemo(t, &mfatypes.MemoMFA{
		GuardianApproval: guardianApproval(t, fx.ctx, fx.cdc, tx, account, guardian, guardianPriv, mfatypes.RecoveryActionDisable, "", 1_700_000_300, 4),
	})

	_, err := fx.handler(fx.ctx, tx, false)
	require.ErrorIs(t, err, mfatypes.ErrInvalidMFAApproval)
}

func TestRejectsMultipleMFAControlActions(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 4)
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

	tx := newTestTx(nil, "", []sdk.AccAddress{account})
	tx.memo = controlMemo(t, &mfatypes.MemoMFA{
		Approvals: []mfatypes.MemoApproval{approval(t, fx.ctx, fx.cdc, tx, account, mfaPriv, 1_700_000_300, 4)},
		Disable:   &mfatypes.MemoDisable{Account: account.String()},
		SetGuardian: &mfatypes.MemoSetGuardian{
			Account: account.String(),
		},
	})

	_, err := fx.handler(fx.ctx, tx, false)
	require.ErrorIs(t, err, mfatypes.ErrInvalidMFAApproval)
	require.True(t, fx.keeper.HasPolicy(fx.ctx, account))
}

func TestDelayedRecoveryStartAndExecuteRotatesMFA(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 4)
	newMfaPriv := secp256k1.GenPrivKey()
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

	startTx := newTestTx([]sdk.Msg{selfSend(account)}, "", []sdk.AccAddress{account})
	startTx.memo = controlMemo(t, &mfatypes.MemoMFA{
		RecoveryStart: &mfatypes.MemoRecoveryStart{
			Account:        account.String(),
			Action:         mfatypes.RecoveryActionRotate,
			ApprovalPubKey: mfatypes.EncodeApprovalPubKey(newMfaPriv.PubKey().Bytes()),
		},
	})
	_, err := fx.handler(fx.ctx, startTx, false)
	require.NoError(t, err)
	policy, found := fx.keeper.GetPolicy(fx.ctx, account)
	require.True(t, found)
	require.NotNil(t, policy.PendingRecovery)

	executeTx := newTestTx([]sdk.Msg{selfSend(account)}, "", []sdk.AccAddress{account})
	executeTx.memo = controlMemo(t, &mfatypes.MemoMFA{
		RecoveryExecute: &mfatypes.MemoRecoveryExecute{Account: account.String()},
	})
	_, err = fx.handler(fx.ctx, executeTx, false)
	require.ErrorIs(t, err, mfatypes.ErrMFARequired)

	fx.ctx = fx.ctx.WithBlockTime(fx.ctx.BlockTime().Add(time.Duration(mfatypes.RecoveryDelaySeconds) * time.Second))
	_, err = fx.handler(fx.ctx, executeTx, false)
	require.NoError(t, err)
	policy, found = fx.keeper.GetPolicy(fx.ctx, account)
	require.True(t, found)
	require.Equal(t, newMfaPriv.PubKey().Bytes(), policy.ApprovalPubKey)
	require.Nil(t, policy.PendingRecovery)
}

func TestDelayedRecoveryCannotBypassOutgoingSend(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 4)
	recipient := sdk.AccAddress("recipient____________")
	require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

	tx := newTestTx([]sdk.Msg{banktypes.NewMsgSend(account, recipient, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 2)))}, "", []sdk.AccAddress{account})
	tx.memo = controlMemo(t, &mfatypes.MemoMFA{
		RecoveryStart: &mfatypes.MemoRecoveryStart{
			Account: account.String(),
			Action:  mfatypes.RecoveryActionDisable,
		},
	})

	_, err := fx.handler(fx.ctx, tx, false)
	require.ErrorIs(t, err, mfatypes.ErrInvalidMFAApproval)
}

func TestDelayedRecoveryRequiresExactSelfSendCarrier(t *testing.T) {
	tests := []struct {
		name string
		msgs []sdk.Msg
	}{
		{
			name: "non self send",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(sdk.AccAddress("sender_______________"), sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1))),
			},
		},
		{
			name: "wrong amount",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(sdk.AccAddress("sender_______________"), sdk.AccAddress("sender_______________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 2))),
			},
		},
		{
			name: "extra message",
			msgs: []sdk.Msg{
				banktypes.NewMsgSend(sdk.AccAddress("sender_______________"), sdk.AccAddress("sender_______________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1))),
				banktypes.NewMsgSend(sdk.AccAddress("sender_______________"), sdk.AccAddress("recipient____________"), sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1))),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fx := newFixture(t)
			account, _, mfaPriv := fx.addAccount(t, 4)
			require.NoError(t, fx.keeper.SetPolicy(fx.ctx, mfatypes.NewPolicy(account, mfaPriv.PubKey().Bytes())))

			msgs := make([]sdk.Msg, len(tc.msgs))
			for index, msg := range tc.msgs {
				if send, ok := msg.(*banktypes.MsgSend); ok {
					send.FromAddress = account.String()
					if tc.name != "non self send" {
						send.ToAddress = account.String()
					}
				}
				msgs[index] = msg
			}

			tx := newTestTx(msgs, "", []sdk.AccAddress{account})
			tx.memo = controlMemo(t, &mfatypes.MemoMFA{
				RecoveryStart: &mfatypes.MemoRecoveryStart{
					Account: account.String(),
					Action:  mfatypes.RecoveryActionDisable,
				},
			})

			_, err := fx.handler(fx.ctx, tx, false)
			require.ErrorIs(t, err, mfatypes.ErrInvalidMFAApproval)
		})
	}
}

func TestControlActionDoesNotApplyWhenLaterAnteFails(t *testing.T) {
	fx := newFixture(t)
	account, _, mfaPriv := fx.addAccount(t, 3)

	blockingHandler := sdk.ChainAnteDecorators(
		mfaante.NewMFARequirementDecorator(fx.cdc, fx.accounts, fx.keeper),
		blockingDecorator{},
		mfaante.NewMFAControlApplyDecorator(fx.keeper),
	)
	enableTx := newTestTx(nil, "", []sdk.AccAddress{account})
	enableTx.memo = controlMemo(t, &mfatypes.MemoMFA{
		Approvals: []mfatypes.MemoApproval{approval(t, fx.ctx, fx.cdc, enableTx, account, mfaPriv, 1_700_000_300, 3)},
		Enable: &mfatypes.MemoEnable{
			Account:        account.String(),
			ApprovalPubKey: mfatypes.EncodeApprovalPubKey(mfaPriv.PubKey().Bytes()),
		},
	})

	_, err := blockingHandler(fx.ctx, enableTx, false)
	require.ErrorContains(t, err, "signature verification failed")
	require.False(t, fx.keeper.HasPolicy(fx.ctx, account))
}

func (f fixture) addAccount(t *testing.T, sequence uint64) (sdk.AccAddress, *secp256k1.PrivKey, *secp256k1.PrivKey) {
	t.Helper()
	walletPriv := secp256k1.GenPrivKey()
	mfaPriv := secp256k1.GenPrivKey()
	account := sdk.AccAddress(walletPriv.PubKey().Address())
	base := authtypes.NewBaseAccountWithAddress(account)
	require.NoError(t, base.SetSequence(sequence))
	f.accounts.set(base)
	return account, walletPriv, mfaPriv
}

func signApprovalMemo(t *testing.T, ctx sdk.Context, cdc codec.Codec, tx testTx, account sdk.AccAddress, priv *secp256k1.PrivKey, expiresAt int64, sequence uint64) string {
	t.Helper()
	return controlMemo(t, &mfatypes.MemoMFA{
		Approvals: []mfatypes.MemoApproval{approval(t, ctx, cdc, tx, account, priv, expiresAt, sequence)},
	})
}

func approval(t *testing.T, ctx sdk.Context, cdc codec.Codec, tx testTx, account sdk.AccAddress, priv *secp256k1.PrivKey, expiresAt int64, sequence uint64) mfatypes.MemoApproval {
	t.Helper()
	payload, err := mfaante.BuildApprovalPayload(ctx, tx, cdc, account, expiresAt, []mfatypes.SignerSequence{{
		Address:  account.String(),
		Sequence: sequence,
	}})
	require.NoError(t, err)
	signature, err := priv.Sign(payload.SignBytes())
	require.NoError(t, err)
	return mfatypes.MemoApproval{
		Account:   account.String(),
		ExpiresAt: expiresAt,
		Signature: mfatypes.EncodeApprovalSignature(signature),
	}
}

func guardianApproval(t *testing.T, ctx sdk.Context, cdc codec.Codec, tx testTx, account sdk.AccAddress, guardian sdk.AccAddress, priv *secp256k1.PrivKey, action string, approvalPubKey string, expiresAt int64, sequence uint64) *mfatypes.MemoGuardianApproval {
	t.Helper()
	payload, err := mfaante.BuildGuardianApprovalPayload(ctx, tx, cdc, account, guardian.String(), action, approvalPubKey, expiresAt, []mfatypes.SignerSequence{{
		Address:  account.String(),
		Sequence: sequence,
	}})
	require.NoError(t, err)
	signature, err := priv.Sign(payload.SignBytes())
	require.NoError(t, err)
	return &mfatypes.MemoGuardianApproval{
		Account:         account.String(),
		GuardianAddress: guardian.String(),
		Action:          action,
		ApprovalPubKey:  approvalPubKey,
		GuardianPubKey:  mfatypes.EncodeApprovalPubKey(priv.PubKey().Bytes()),
		ExpiresAt:       expiresAt,
		Signature:       mfatypes.EncodeApprovalSignature(signature),
	}
}

func selfSend(account sdk.AccAddress) sdk.Msg {
	return banktypes.NewMsgSend(account, account, sdk.NewCoins(sdk.NewInt64Coin(core.MicroDoDenom, 1)))
}

func controlMemo(t *testing.T, mfa *mfatypes.MemoMFA) string {
	t.Helper()
	bz, err := json.Marshal(mfatypes.MemoEnvelope{MFA: mfa})
	require.NoError(t, err)
	return string(bz)
}

func passthrough(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
	return ctx, nil
}

type blockingDecorator struct{}

func (blockingDecorator) AnteHandle(ctx sdk.Context, _ sdk.Tx, _ bool, _ sdk.AnteHandler) (sdk.Context, error) {
	return ctx, errors.New("signature verification failed")
}

type testTx struct {
	msgs    []sdk.Msg
	memo    string
	signers []sdk.AccAddress
}

func newTestTx(msgs []sdk.Msg, memo string, signers []sdk.AccAddress) testTx {
	return testTx{msgs: msgs, memo: memo, signers: signers}
}

func (t testTx) GetMsgs() []sdk.Msg { return t.msgs }

func (t testTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

func (t testTx) GetMemo() string { return t.memo }

func (t testTx) GetTimeoutHeight() uint64 { return 0 }

func (t testTx) GetSigners() ([][]byte, error) {
	signers := make([][]byte, len(t.signers))
	for i, signer := range t.signers {
		signers[i] = signer.Bytes()
	}
	return signers, nil
}

func (t testTx) GetPubKeys() ([]cryptotypes.PubKey, error) { return nil, nil }

func (t testTx) GetSignaturesV2() ([]sdksigning.SignatureV2, error) { return nil, nil }

type mockAccountKeeper struct {
	accounts map[string]sdk.AccountI
	codec    coreaddress.Codec
}

func newMockAccountKeeper() *mockAccountKeeper {
	return &mockAccountKeeper{
		accounts: make(map[string]sdk.AccountI),
		codec:    address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
	}
}

func (m *mockAccountKeeper) set(account sdk.AccountI) {
	m.accounts[account.GetAddress().String()] = account
}

func (m *mockAccountKeeper) GetParams(context.Context) authtypes.Params {
	return authtypes.DefaultParams()
}

func (m *mockAccountKeeper) GetAccount(_ context.Context, addr sdk.AccAddress) sdk.AccountI {
	return m.accounts[addr.String()]
}

func (m *mockAccountKeeper) SetAccount(_ context.Context, acc sdk.AccountI) {
	m.accounts[acc.GetAddress().String()] = acc
}

func (m *mockAccountKeeper) GetModuleAddress(string) sdk.AccAddress { return nil }

func (m *mockAccountKeeper) AddressCodec() coreaddress.Codec { return m.codec }

func (m *mockAccountKeeper) UnorderedTransactionsEnabled() bool { return false }

func (m *mockAccountKeeper) RemoveExpiredUnorderedNonces(sdk.Context) error { return nil }

func (m *mockAccountKeeper) TryAddUnorderedNonce(sdk.Context, []byte, time.Time) error {
	return nil
}
