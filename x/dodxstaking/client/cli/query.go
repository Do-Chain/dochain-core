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
		GetStakesQueryCmd(),
		GetTotalStakedQueryCmd(),
		GetPendingRewardsQueryCmd(),
		GetRewardPoolQueryCmd(),
	)

	return cmd
}

// GetStakesQueryCmd creates a query command for all native DODx stake records.
func GetStakesQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stakes",
		Args:  cobra.NoArgs,
		Short: "Query all native DODx stake records",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Stakes(context.Background(), &types.QueryStakesRequest{Pagination: pageReq})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, "stakes")
	flags.AddQueryFlagsToCmd(cmd)
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

// GetPendingRewardsQueryCmd creates a pending rewards query command.
func GetPendingRewardsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pending-rewards [address]",
		Args:  cobra.ExactArgs(1),
		Short: "Query pending DEX fee rewards for a native DODx staker",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PendingRewards(context.Background(), &types.QueryPendingRewardsRequest{Address: args[0]})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetRewardPoolQueryCmd creates a reward pool query command.
func GetRewardPoolQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward-pool",
		Args:  cobra.NoArgs,
		Short: "Query accounted, unclaimed DEX fee reward pools",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.RewardPool(context.Background(), &types.QueryRewardPoolRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
