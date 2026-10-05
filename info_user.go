package hyperliquid

import (
	"context"
	"encoding/json"
)

// SubAccountsRequest is the request for [InfoClient.SubAccounts].
type SubAccountsRequest struct {
	User Address `json:"user"`
}

// SubAccount is a sub-account with its main-dex state.
type SubAccount struct {
	Name               string                 `json:"name"`
	SubAccountUser     Address                `json:"subAccountUser"`
	Master             Address                `json:"master"`
	ClearinghouseState ClearinghouseState     `json:"clearinghouseState"`
	SpotState          SpotClearinghouseState `json:"spotState"`
}

// SubAccounts returns a master account's sub-accounts.
func (c *InfoClient) SubAccounts(ctx context.Context, req SubAccountsRequest) ([]SubAccount, error) {
	return infoRequest[[]SubAccount](ctx, c, "subAccounts", req)
}

// SubAccounts2Request is the request for [InfoClient.SubAccounts2].
type SubAccounts2Request struct {
	User Address `json:"user"`
}

// SubAccount2 is a sub-account with its state on every perp dex.
type SubAccount2 struct {
	Name           string  `json:"name"`
	SubAccountUser Address `json:"subAccountUser"`
	Master         Address `json:"master"`
	// DexToClearinghouseState maps perp dex names ("" for the main dex) to
	// the sub-account's state on that dex.
	DexToClearinghouseState TupleMap[string, ClearinghouseState] `json:"dexToClearinghouseState"`
	SpotState               SpotClearinghouseState               `json:"spotState"`
	Abstraction             Abstraction                          `json:"abstraction,omitempty"`
}

// SubAccounts2 returns a master account's sub-accounts with their state on
// every perp dex.
func (c *InfoClient) SubAccounts2(ctx context.Context, req SubAccounts2Request) ([]SubAccount2, error) {
	return infoRequest[[]SubAccount2](ctx, c, "subAccounts2", req)
}

// UserRoleRequest is the request for [InfoClient.UserRole].
type UserRoleRequest struct {
	User Address `json:"user"`
}

// UserRole is the kind of an account.
type UserRole struct {
	// Role is "missing", "user", "vault", "agent" or "subAccount".
	Role string `json:"role"`
	// Data is set for agents and sub-accounts.
	Data *UserRoleData `json:"data,omitempty"`
}

// UserRoleData links an agent or sub-account to its owner.
type UserRoleData struct {
	// User is the account an agent trades for.
	User Address `json:"user,omitzero"`
	// Master is the owner of a sub-account.
	Master Address `json:"master,omitzero"`
}

// UserRole returns whether an address is a user, vault, agent or
// sub-account.
func (c *InfoClient) UserRole(ctx context.Context, req UserRoleRequest) (*UserRole, error) {
	return infoRequest[*UserRole](ctx, c, "userRole", req)
}

// ExtraAgentsRequest is the request for [InfoClient.ExtraAgents].
type ExtraAgentsRequest struct {
	User Address `json:"user"`
}

// ExtraAgent is a named API wallet approved by a user.
type ExtraAgent struct {
	Address Address `json:"address"`
	Name    string  `json:"name"`
	// ValidUntil is the expiry in milliseconds since the Unix epoch, or nil.
	ValidUntil *int64 `json:"validUntil"`
}

// ExtraAgents returns a user's named API wallets.
func (c *InfoClient) ExtraAgents(ctx context.Context, req ExtraAgentsRequest) ([]ExtraAgent, error) {
	return infoRequest[[]ExtraAgent](ctx, c, "extraAgents", req)
}

// UserToMultiSigSignersRequest is the request for
// [InfoClient.UserToMultiSigSigners].
type UserToMultiSigSignersRequest struct {
	User Address `json:"user"`
}

// UserToMultiSigSigners returns the signers of a multi-sig user, or nil if
// the user is not multi-sig.
func (c *InfoClient) UserToMultiSigSigners(ctx context.Context, req UserToMultiSigSignersRequest) (*MultiSigSigners, error) {
	return infoRequest[*MultiSigSigners](ctx, c, "userToMultiSigSigners", req)
}

// UserAbstractionRequest is the request for [InfoClient.UserAbstraction].
type UserAbstractionRequest struct {
	User Address `json:"user"`
}

// UserAbstraction returns a user's account abstraction mode.
func (c *InfoClient) UserAbstraction(ctx context.Context, req UserAbstractionRequest) (Abstraction, error) {
	return infoRequest[Abstraction](ctx, c, "userAbstraction", req)
}

// UserDexAbstractionRequest is the request for [InfoClient.UserDexAbstraction].
type UserDexAbstractionRequest struct {
	User Address `json:"user"`
}

// UserDexAbstraction reports whether a user has enabled dex abstraction,
// which lets builder perp dexes use the main dex's collateral.
func (c *InfoClient) UserDexAbstraction(ctx context.Context, req UserDexAbstractionRequest) (bool, error) {
	return infoRequest[bool](ctx, c, "userDexAbstraction", req)
}

// ReferralRequest is the request for [InfoClient.Referral].
type ReferralRequest struct {
	User Address `json:"user"`
}

// Referral is a user's referral status and rewards.
type Referral struct {
	// ReferredBy is the user's referrer, or nil.
	ReferredBy       *ReferredBy      `json:"referredBy"`
	CumVlm           Decimal          `json:"cumVlm"`
	UnclaimedRewards Decimal          `json:"unclaimedRewards"`
	ClaimedRewards   Decimal          `json:"claimedRewards"`
	BuilderRewards   Decimal          `json:"builderRewards"`
	ReferrerState    ReferrerState    `json:"referrerState"`
	RewardHistory    []ReferralReward `json:"rewardHistory"`
	// TokenToState breaks the rewards down by spot token index.
	TokenToState TupleMap[int, ReferralTokenState] `json:"tokenToState"`
}

// ReferredBy is the referrer of a user.
type ReferredBy struct {
	Referrer Address `json:"referrer"`
	Code     string  `json:"code"`
}

// ReferrerState is a user's progress as a referrer.
type ReferrerState struct {
	// Stage is "ready", "needToCreateCode" or "needToTrade".
	Stage string `json:"stage"`
	// Data is set for the "ready" and "needToTrade" stages.
	Data *ReferrerData `json:"data,omitempty"`
}

// ReferrerData holds Code, NReferrals and ReferralStates in the "ready"
// stage, and Required in the "needToTrade" stage.
type ReferrerData struct {
	Code           string          `json:"code,omitempty"`
	NReferrals     int             `json:"nReferrals,omitempty"`
	ReferralStates []ReferralState `json:"referralStates,omitempty"`
	// Required is the volume left to trade before creating a code.
	Required Decimal `json:"required,omitempty"`
}

// ReferralState is a referred user as seen by the referrer.
type ReferralState struct {
	CumVlm                       Decimal `json:"cumVlm"`
	CumRewardedFeesSinceReferred Decimal `json:"cumRewardedFeesSinceReferred"`
	CumFeesRewardedToReferrer    Decimal `json:"cumFeesRewardedToReferrer"`
	// TimeJoined is in milliseconds since the Unix epoch.
	TimeJoined   int64                                  `json:"timeJoined"`
	User         Address                                `json:"user"`
	TokenToState TupleMap[int, ReferralStateTokenState] `json:"tokenToState"`
}

// ReferralStateTokenState is a [ReferralState] for one spot token.
type ReferralStateTokenState struct {
	CumVlm                       Decimal `json:"cumVlm"`
	CumRewardedFeesSinceReferred Decimal `json:"cumRewardedFeesSinceReferred"`
	CumFeesRewardedToReferrer    Decimal `json:"cumFeesRewardedToReferrer"`
}

// ReferralReward is a referral reward accrual.
type ReferralReward struct {
	Earned      Decimal `json:"earned"`
	Vlm         Decimal `json:"vlm"`
	ReferralVlm Decimal `json:"referralVlm"`
	// Time is in milliseconds since the Unix epoch.
	Time int64 `json:"time"`
}

// ReferralTokenState is a user's referral volume and rewards for one spot
// token.
type ReferralTokenState struct {
	CumVlm           Decimal `json:"cumVlm"`
	UnclaimedRewards Decimal `json:"unclaimedRewards"`
	ClaimedRewards   Decimal `json:"claimedRewards"`
	BuilderRewards   Decimal `json:"builderRewards"`
}

// Referral returns a user's referral status and rewards.
func (c *InfoClient) Referral(ctx context.Context, req ReferralRequest) (*Referral, error) {
	return infoRequest[*Referral](ctx, c, "referral", req)
}

// UserFeesRequest is the request for [InfoClient.UserFees].
type UserFeesRequest struct {
	User Address `json:"user"`
}

// UserFees is a user's fee schedule, current rates and recent volume. Rates
// are fractions: 0.00045 means 4.5 basis points.
type UserFees struct {
	DailyUserVlm      []DailyUserVolume `json:"dailyUserVlm"`
	FeeSchedule       FeeSchedule       `json:"feeSchedule"`
	UserCrossRate     Decimal           `json:"userCrossRate"`
	UserAddRate       Decimal           `json:"userAddRate"`
	UserSpotCrossRate Decimal           `json:"userSpotCrossRate"`
	UserSpotAddRate   Decimal           `json:"userSpotAddRate"`
	// ActiveReferralDiscount is the discount fraction from being referred.
	ActiveReferralDiscount Decimal `json:"activeReferralDiscount"`
	// Trial is an undocumented fee trial, usually null.
	Trial          json.RawMessage `json:"trial"`
	FeeTrialEscrow Decimal         `json:"feeTrialEscrow"`
	// NextTrialAvailableTimestamp is in milliseconds since the Unix epoch,
	// or nil.
	NextTrialAvailableTimestamp *int64          `json:"nextTrialAvailableTimestamp"`
	StakingLink                 *StakingLink    `json:"stakingLink"`
	ActiveStakingDiscount       StakingDiscount `json:"activeStakingDiscount"`
}

// DailyUserVolume is a user's and the exchange's volume on one day.
type DailyUserVolume struct {
	// Date is formatted YYYY-MM-DD.
	Date      string  `json:"date"`
	UserCross Decimal `json:"userCross"`
	UserAdd   Decimal `json:"userAdd"`
	Exchange  Decimal `json:"exchange"`
}

// FeeSchedule is the base fee schedule.
type FeeSchedule struct {
	Cross     Decimal  `json:"cross"`
	Add       Decimal  `json:"add"`
	SpotCross Decimal  `json:"spotCross"`
	SpotAdd   Decimal  `json:"spotAdd"`
	Tiers     FeeTiers `json:"tiers"`
	// ReferralDiscount is the discount fraction for referred users.
	ReferralDiscount     Decimal           `json:"referralDiscount"`
	StakingDiscountTiers []StakingDiscount `json:"stakingDiscountTiers"`
}

// FeeTiers lists the volume and market-maker fee tiers.
type FeeTiers struct {
	VIP []VIPFeeTier `json:"vip"`
	MM  []MMFeeTier  `json:"mm"`
}

// VIPFeeTier is a fee tier reached by 14-day volume.
type VIPFeeTier struct {
	NtlCutoff Decimal `json:"ntlCutoff"`
	Cross     Decimal `json:"cross"`
	Add       Decimal `json:"add"`
	SpotCross Decimal `json:"spotCross"`
	SpotAdd   Decimal `json:"spotAdd"`
}

// MMFeeTier is a maker rebate tier reached by share of maker volume.
type MMFeeTier struct {
	MakerFractionCutoff Decimal `json:"makerFractionCutoff"`
	// Add is the maker rate; negative is a rebate.
	Add Decimal `json:"add"`
}

// StakingDiscount is a fee discount for staking a share of the HYPE supply.
type StakingDiscount struct {
	BpsOfMaxSupply Decimal `json:"bpsOfMaxSupply"`
	Discount       Decimal `json:"discount"`
}

// StakingLink links a trading account to a staking account for fee
// discounts.
type StakingLink struct {
	// Type is "requested" or "tradingUser" (StakingUser set), or
	// "stakingUser" (TradingUser set).
	Type        string  `json:"type"`
	StakingUser Address `json:"stakingUser,omitzero"`
	TradingUser Address `json:"tradingUser,omitzero"`
}

// UserFees returns a user's fee rates and volume history.
func (c *InfoClient) UserFees(ctx context.Context, req UserFeesRequest) (*UserFees, error) {
	return infoRequest[*UserFees](ctx, c, "userFees", req)
}

// UserRateLimitRequest is the request for [InfoClient.UserRateLimit].
type UserRateLimitRequest struct {
	User Address `json:"user"`
}

// RateLimit is a user's address-based action rate limit. The cap grows with
// traded volume.
type RateLimit struct {
	CumVlm           Decimal `json:"cumVlm"`
	NRequestsUsed    int     `json:"nRequestsUsed"`
	NRequestsCap     int     `json:"nRequestsCap"`
	NRequestsSurplus int     `json:"nRequestsSurplus"`
}

// UserRateLimit returns a user's action rate limit usage.
func (c *InfoClient) UserRateLimit(ctx context.Context, req UserRateLimitRequest) (*RateLimit, error) {
	return infoRequest[*RateLimit](ctx, c, "userRateLimit", req)
}

// MaxBuilderFeeRequest is the request for [InfoClient.MaxBuilderFee].
type MaxBuilderFeeRequest struct {
	User    Address `json:"user"`
	Builder Address `json:"builder"`
}

// MaxBuilderFee returns the maximum fee a user has approved for a builder, in
// tenths of a basis point (see [Builder.Fee]).
func (c *InfoClient) MaxBuilderFee(ctx context.Context, req MaxBuilderFeeRequest) (int, error) {
	return infoRequest[int](ctx, c, "maxBuilderFee", req)
}

// ApprovedBuildersRequest is the request for [InfoClient.ApprovedBuilders].
type ApprovedBuildersRequest struct {
	User Address `json:"user"`
}

// ApprovedBuilders returns the builders a user has approved fees for.
func (c *InfoClient) ApprovedBuilders(ctx context.Context, req ApprovedBuildersRequest) ([]Address, error) {
	return infoRequest[[]Address](ctx, c, "approvedBuilders", req)
}

// IsVIPRequest is the request for [InfoClient.IsVIP].
type IsVIPRequest struct {
	User Address `json:"user"`
}

// IsVIP reports whether a user has VIP status.
func (c *InfoClient) IsVIP(ctx context.Context, req IsVIPRequest) (bool, error) {
	return infoRequest[bool](ctx, c, "isVip", req)
}

// LegalCheckRequest is the request for [InfoClient.LegalCheck].
type LegalCheckRequest struct {
	User Address `json:"user"`
}

// LegalCheck is a user's terms acceptance and jurisdiction status.
type LegalCheck struct {
	AcceptedTerms bool `json:"acceptedTerms"`
	UserAllowed   bool `json:"userAllowed"`
	// Restrictions is an undocumented one-letter code: "n", "a", "o" or
	// "u".
	Restrictions string `json:"restrictions"`
}

// LegalCheck returns whether a user has accepted the terms and may use the
// exchange.
func (c *InfoClient) LegalCheck(ctx context.Context, req LegalCheckRequest) (*LegalCheck, error) {
	return infoRequest[*LegalCheck](ctx, c, "legalCheck", req)
}

// PreTransferCheckRequest is the request for [InfoClient.PreTransferCheck].
type PreTransferCheckRequest struct {
	// User is the transfer destination; Source the sender.
	User   Address `json:"user"`
	Source Address `json:"source"`
}

// PreTransferCheck describes the destination of a transfer.
type PreTransferCheck struct {
	// Fee is the fee to activate a new destination account.
	Fee           Decimal `json:"fee"`
	IsSanctioned  bool    `json:"isSanctioned"`
	UserExists    bool    `json:"userExists"`
	UserHasSentTx bool    `json:"userHasSentTx"`
}

// PreTransferCheck checks a transfer destination before sending to it.
func (c *InfoClient) PreTransferCheck(ctx context.Context, req PreTransferCheckRequest) (*PreTransferCheck, error) {
	return infoRequest[*PreTransferCheck](ctx, c, "preTransferCheck", req)
}
