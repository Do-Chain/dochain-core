package cli

import (
	"strconv"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

const (
	flagTitle          = "title"
	flagDescription    = "description"
	flagStartHeight    = "start-height"
	flagEndHeight      = "end-height"
	flagDelegatorsOnly = "delegators-only"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "validatorrewards",
		Short:                      "Validator delegator reward campaign transactions",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCreateCampaignCmd(),
		GetReleaseCampaignCmd(),
		GetCancelCampaignCmd(),
	)

	return cmd
}

func GetCreateCampaignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-campaign [denom] [validator=amount,...]",
		Args:  cobra.ExactArgs(2),
		Short: "Create and fund a validator delegator reward campaign",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			allocations, err := parseAllocations(args[0], args[1])
			if err != nil {
				return err
			}
			title, err := cmd.Flags().GetString(flagTitle)
			if err != nil {
				return err
			}
			description, err := cmd.Flags().GetString(flagDescription)
			if err != nil {
				return err
			}
			startHeight, err := cmd.Flags().GetInt64(flagStartHeight)
			if err != nil {
				return err
			}
			endHeight, err := cmd.Flags().GetInt64(flagEndHeight)
			if err != nil {
				return err
			}
			delegatorsOnly, err := cmd.Flags().GetBool(flagDelegatorsOnly)
			if err != nil {
				return err
			}
			msg := types.NewMsgCreateCampaign(clientCtx.GetFromAddress(), args[0], title, description, allocations, startHeight, endHeight, delegatorsOnly)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(flagTitle, "", "campaign title for wallets")
	cmd.Flags().String(flagDescription, "", "campaign description for wallets")
	cmd.Flags().Int64(flagStartHeight, 0, "first block height that can release rewards")
	cmd.Flags().Int64(flagEndHeight, 0, "final block height for full drip release; zero releases immediately")
	cmd.Flags().Bool(flagDelegatorsOnly, false, "send campaign rewards to delegator rewards without validator commission")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func GetReleaseCampaignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release-campaign [campaign-id]",
		Args:  cobra.ExactArgs(1),
		Short: "Release vested validator delegator rewards from a campaign",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			msg := types.NewMsgReleaseCampaign(clientCtx.GetFromAddress(), id)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func GetCancelCampaignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-campaign [campaign-id]",
		Args:  cobra.ExactArgs(1),
		Short: "Cancel a campaign and refund unreleased rewards to the admin",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			msg := types.NewMsgCancelCampaign(clientCtx.GetFromAddress(), id)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseAllocations(denom, input string) ([]types.ValidatorAllocation, error) {
	parts := strings.Split(input, ",")
	allocations := make([]types.ValidatorAllocation, 0, len(parts))
	for _, part := range parts {
		fields := strings.SplitN(part, "=", 2)
		if len(fields) != 2 {
			return nil, types.ErrInvalidCampaign.Wrap("allocation must be validator=amount")
		}
		amount, ok := sdkmath.NewIntFromString(fields[1])
		if !ok {
			return nil, types.ErrInvalidCampaign.Wrap("invalid allocation amount")
		}
		allocations = append(allocations, types.ValidatorAllocation{
			ValidatorAddress: fields[0],
			Amount:           sdk.NewCoin(denom, amount),
			Released:         "0",
		})
	}
	return allocations, nil
}
