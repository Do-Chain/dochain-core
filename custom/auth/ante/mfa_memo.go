package ante

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
)

const maxMFAMemoCharacters uint64 = 4096

type MFAMemoAwareValidateMemoDecorator struct {
	ak authante.AccountKeeper
}

func NewMFAMemoAwareValidateMemoDecorator(ak authante.AccountKeeper) MFAMemoAwareValidateMemoDecorator {
	return MFAMemoAwareValidateMemoDecorator{ak: ak}
}

func (vmd MFAMemoAwareValidateMemoDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	memoTx, ok := tx.(sdk.TxWithMemo)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	memo := memoTx.GetMemo()
	memoLength := len(memo)
	if memoLength > 0 {
		params := vmd.ak.GetParams(ctx)
		if uint64(memoLength) > params.MaxMemoCharacters && !isAllowedMFAMemo(memo, memoLength) {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrMemoTooLarge,
				"maximum number of characters is %d but received %d characters",
				params.MaxMemoCharacters, memoLength,
			)
		}
	}

	return next(ctx, tx, simulate)
}

func isAllowedMFAMemo(memo string, memoLength int) bool {
	if uint64(memoLength) > maxMFAMemoCharacters {
		return false
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(memo), &envelope); err != nil {
		return false
	}

	_, ok := envelope["dochain_mfa"]
	return ok
}
