package cli

import (
	"context"
	"strconv"

	"github.com/Daviddochain/dochain-core/v4/x/validatorrewards/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "validatorrewards",
		Short:                      "Validator delegator reward campaign queries",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		GetCampaignCmd(),
		GetCampaignsCmd(),
		GetValidatorCampaignsCmd(),
		GetRewardDenomsCmd(),
	)
	return cmd
}

func GetCampaignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "campaign [campaign-id]",
		Args:  cobra.ExactArgs(1),
		Short: "Query a validator reward campaign",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Campaign(context.Background(), &types.QueryCampaignRequest{CampaignId: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetCampaignsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "campaigns",
		Args:  cobra.NoArgs,
		Short: "Query validator reward campaigns",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Campaigns(context.Background(), &types.QueryCampaignsRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddPaginationFlagsToCmd(cmd, "campaigns")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetValidatorCampaignsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-campaigns [validator-address]",
		Args:  cobra.ExactArgs(1),
		Short: "Query campaigns funding one validator",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).ValidatorCampaigns(context.Background(), &types.QueryValidatorCampaignsRequest{ValidatorAddress: args[0], Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddPaginationFlagsToCmd(cmd, "campaigns")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func GetRewardDenomsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward-denoms",
		Args:  cobra.NoArgs,
		Short: "Query registered validator reward denoms",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).RewardDenoms(context.Background(), &types.QueryRewardDenomsRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddPaginationFlagsToCmd(cmd, "reward denoms")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
