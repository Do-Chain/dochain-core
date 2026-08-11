package types

const (
	EventTypeRegisterRewardDenom = "validator_rewards_register_denom"
	EventTypeCreateCampaign      = "validator_rewards_create_campaign"
	EventTypeReleaseCampaign     = "validator_rewards_release_campaign"
	EventTypeCancelCampaign      = "validator_rewards_cancel_campaign"

	AttributeKeyAdmin     = "admin"
	AttributeKeyAuthority = "authority"
	AttributeKeyCampaign  = "campaign_id"
	AttributeKeyDenom     = "denom"
	AttributeKeyAmount    = "amount"
	AttributeKeyValidator = "validator"
)
