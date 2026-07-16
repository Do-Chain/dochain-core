package cli

import (
	"context"

	"github.com/Daviddochain/dochain-core/v4/x/dodxstaking/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns DODx staking query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "dodxstaking",
		Short:                      "DODx governance staking queries",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetStakeQueryCmd(),
		GetTotalStakedQueryCmd(),
	)

	return cmd
}

// GetStakeQueryCmd creates a stake query command.
func GetStakeQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake [address]",
		Args:  cobra.ExactArgs(1),
		Short: "Query staked DODx for an address",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Stake(context.Background(), &types.QueryStakeRequest{Address: args[0]})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetTotalStakedQueryCmd creates a total staked query command.
func GetTotalStakedQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "total-staked",
		Args:  cobra.NoArgs,
		Short: "Query total staked DODx",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.TotalStaked(context.Background(), &types.QueryTotalStakedRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
