package cli

import (
	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

// GetTxCmd returns DODx staking transaction commands.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "dodxstaking",
		Short:                      "DODx governance staking transactions",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetStakeCmd(),
		GetUnstakeCmd(),
	)

	return cmd
}

// GetStakeCmd creates a stake transaction command.
func GetStakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake [amount]",
		Args:  cobra.ExactArgs(1),
		Short: "Stake DODx for governance voting power",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgStake(clientCtx.GetFromAddress(), amount)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetUnstakeCmd creates an unstake transaction command.
func GetUnstakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake [amount]",
		Args:  cobra.ExactArgs(1),
		Short: "Unstake DODx and remove governance voting power",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgUnstake(clientCtx.GetFromAddress(), amount)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
