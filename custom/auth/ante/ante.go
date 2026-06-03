package ante

import (
	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	txsigning "cosmossdk.io/x/tx/signing"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	dyncommante "github.com/Daviddochain/dochain-core/v4/x/dyncomm/ante"
	dyncommkeeper "github.com/Daviddochain/dochain-core/v4/x/dyncomm/keeper"
	mfaante "github.com/Daviddochain/dochain-core/v4/x/mfa/ante"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
)

type HandlerOptions struct {
	AccountKeeper          ante.AccountKeeper
	BankKeeper             BankKeeper
	ExtensionOptionChecker ante.ExtensionOptionChecker
	FeegrantKeeper         ante.FeegrantKeeper
	OracleKeeper           OracleKeeper
	TreasuryKeeper         TreasuryKeeper
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         ante.SignatureVerificationGasConsumer
	TxFeeChecker           ante.TxFeeChecker
	IBCKeeper              ibckeeper.Keeper
	WasmKeeper             *wasmkeeper.Keeper
	DistributionKeeper     distributionkeeper.Keeper
	GovKeeper              govkeeper.Keeper
	WasmConfig             *wasmtypes.NodeConfig
	TXCounterStore         corestoretypes.KVStoreService
	DyncommKeeper          dyncommkeeper.Keeper
	MFAKeeper              mfaante.MFAKeeper
	StakingKeeper          *stakingkeeper.Keeper
	Cdc                    codec.Codec
}

func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for ante builder")
	}

	if options.OracleKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "oracle keeper is required for ante builder")
	}

	if options.TreasuryKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "treasury keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}

	if options.WasmConfig == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "wasm config is required for ante builder")
	}

	if options.TXCounterStore == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "tx counter store service is required for ante builder")
	}

	return sdk.ChainAnteDecorators(
		ante.NewSetUpContextDecorator(),
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(options.TXCounterStore),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		NewMFAMemoAwareValidateMemoDecorator(options.AccountKeeper),
		NewSpammingPreventionDecorator(options.OracleKeeper),
		NewIBCTransferSpamPreventionDecorator(),
		NewMinInitialDepositDecorator(options.GovKeeper, options.TreasuryKeeper),
		mfaante.NewMFARequirementDecorator(options.Cdc, options.AccountKeeper, options.MFAKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		NewFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TreasuryKeeper, options.DistributionKeeper),
		dyncommante.NewDyncommDecorator(options.Cdc, options.DyncommKeeper, options.StakingKeeper),
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(&options.IBCKeeper),
	), nil
}
