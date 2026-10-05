package hyperliquid

import (
	"context"
	"encoding/json"
)

// UpdateLeverageAction sets the leverage and margin mode of a perp asset.
type UpdateLeverageAction struct {
	Asset    int  `json:"asset"`
	IsCross  bool `json:"isCross"`
	Leverage int  `json:"leverage"`
}

func (UpdateLeverageAction) actionType() string { return "updateLeverage" }

// UpdateLeverage sets the leverage of a perp asset.
func (c *ExchangeClient) UpdateLeverage(ctx context.Context, a UpdateLeverageAction) error {
	return c.do(ctx, a, nil)
}

// UpdateIsolatedMarginAction adds margin to or removes margin from an
// isolated position.
type UpdateIsolatedMarginAction struct {
	Asset int `json:"asset"`
	// IsBuy is the position side; the official SDKs always send true.
	IsBuy bool `json:"isBuy"`
	// Ntli is the signed USDC amount to add (positive) or remove (negative),
	// times 1e6.
	Ntli int64 `json:"ntli"`
}

func (UpdateIsolatedMarginAction) actionType() string { return "updateIsolatedMargin" }

// UpdateIsolatedMargin changes the margin of an isolated position.
func (c *ExchangeClient) UpdateIsolatedMargin(ctx context.Context, a UpdateIsolatedMarginAction) error {
	return c.do(ctx, a, nil)
}

// TopUpIsolatedOnlyMarginAction sets the margin of an isolated-only position
// to reach a target leverage.
type TopUpIsolatedOnlyMarginAction struct {
	Asset    int     `json:"asset"`
	Leverage Decimal `json:"leverage"`
}

func (TopUpIsolatedOnlyMarginAction) actionType() string { return "topUpIsolatedOnlyMargin" }

// TopUpIsolatedOnlyMargin tops up an isolated-only position's margin.
func (c *ExchangeClient) TopUpIsolatedOnlyMargin(ctx context.Context, a TopUpIsolatedOnlyMarginAction) error {
	return c.do(ctx, a, nil)
}

// BorrowLendAction supplies, withdraws, borrows or repays a token.
type BorrowLendAction struct {
	Operation BorrowLendOperation `json:"operation"`
	// Token is the spot token index.
	Token int `json:"token"`
	// Amount is the token amount; nil means the full balance or debt.
	Amount *Decimal `json:"amount"`
}

func (BorrowLendAction) actionType() string { return "borrowLend" }

// BorrowLendOperation is the operation of a [BorrowLendAction].
type BorrowLendOperation string

// Borrow/lend operations.
const (
	BorrowLendSupply   BorrowLendOperation = "supply"
	BorrowLendWithdraw BorrowLendOperation = "withdraw"
	BorrowLendRepay    BorrowLendOperation = "repay"
	BorrowLendBorrow   BorrowLendOperation = "borrow"
)

// BorrowLend supplies, withdraws, borrows or repays a token.
func (c *ExchangeClient) BorrowLend(ctx context.Context, a BorrowLendAction) error {
	return c.do(ctx, a, nil)
}

// NoopAction does nothing; submitting it consumes its nonce.
type NoopAction struct{}

func (NoopAction) actionType() string { return "noop" }

// Noop submits a no-op with the given nonce, invalidating any pending
// action signed with that nonce.
func (c *ExchangeClient) Noop(ctx context.Context, nonce uint64) error {
	return c.send(ctx, NoopAction{}, nonce, nil)
}

// ReserveRequestWeightAction buys additional request rate limit.
type ReserveRequestWeightAction struct {
	Weight uint64 `json:"weight"`
	// Destination optionally reserves the weight for another existing user.
	Destination *Address `json:"destination,omitempty"`
}

func (ReserveRequestWeightAction) actionType() string { return "reserveRequestWeight" }

// ReserveRequestWeight buys additional request rate limit.
func (c *ExchangeClient) ReserveRequestWeight(ctx context.Context, a ReserveRequestWeightAction) error {
	return c.do(ctx, a, nil)
}

// EVMUserModifyAction selects the HyperEVM block type of the user's
// transactions.
type EVMUserModifyAction struct {
	UsingBigBlocks bool `json:"usingBigBlocks"`
}

func (EVMUserModifyAction) actionType() string { return "evmUserModify" }

// EVMUserModify toggles HyperEVM big blocks.
func (c *ExchangeClient) EVMUserModify(ctx context.Context, a EVMUserModifyAction) error {
	return c.do(ctx, a, nil)
}

// SetReferrerAction sets the account's referrer by referral code.
type SetReferrerAction struct {
	Code string `json:"code"`
}

func (SetReferrerAction) actionType() string { return "setReferrer" }

// SetReferrer sets the account's referrer.
func (c *ExchangeClient) SetReferrer(ctx context.Context, a SetReferrerAction) error {
	return c.do(ctx, a, nil)
}

// RegisterReferrerAction registers a referral code for the account.
type RegisterReferrerAction struct {
	Code string `json:"code"`
}

func (RegisterReferrerAction) actionType() string { return "registerReferrer" }

// RegisterReferrer registers a referral code.
func (c *ExchangeClient) RegisterReferrer(ctx context.Context, a RegisterReferrerAction) error {
	return c.do(ctx, a, nil)
}

// SetDisplayNameAction sets the account's leaderboard display name.
type SetDisplayNameAction struct {
	// DisplayName is at most 20 characters; empty clears it.
	DisplayName string `json:"displayName"`
}

func (SetDisplayNameAction) actionType() string { return "setDisplayName" }

// SetDisplayName sets the account's display name.
func (c *ExchangeClient) SetDisplayName(ctx context.Context, a SetDisplayNameAction) error {
	return c.do(ctx, a, nil)
}

// ClaimRewardsAction claims referral and builder rewards.
type ClaimRewardsAction struct{}

func (ClaimRewardsAction) actionType() string { return "claimRewards" }

// ClaimRewards claims referral and builder rewards.
func (c *ExchangeClient) ClaimRewards(ctx context.Context) error {
	return c.do(ctx, ClaimRewardsAction{}, nil)
}

// CreateSubAccountAction creates a sub-account.
type CreateSubAccountAction struct {
	// Name is 1 to 16 characters.
	Name string `json:"name"`
}

func (CreateSubAccountAction) actionType() string { return "createSubAccount" }

// CreateSubAccount creates a sub-account and returns its address.
func (c *ExchangeClient) CreateSubAccount(ctx context.Context, a CreateSubAccountAction) (Address, error) {
	var addr Address
	return addr, c.do(ctx, a, &addr)
}

// SubAccountModifyAction renames a sub-account.
type SubAccountModifyAction struct {
	SubAccountUser Address `json:"subAccountUser"`
	Name           string  `json:"name"`
}

func (SubAccountModifyAction) actionType() string { return "subAccountModify" }

// SubAccountModify renames a sub-account.
func (c *ExchangeClient) SubAccountModify(ctx context.Context, a SubAccountModifyAction) error {
	return c.do(ctx, a, nil)
}

// SubAccountTransferAction moves perp USDC between the account and a
// sub-account.
type SubAccountTransferAction struct {
	SubAccountUser Address `json:"subAccountUser"`
	// IsDeposit moves funds into the sub-account; otherwise out of it.
	IsDeposit bool `json:"isDeposit"`
	// USD is the USDC amount times 1e6.
	USD uint64 `json:"usd"`
}

func (SubAccountTransferAction) actionType() string { return "subAccountTransfer" }

// SubAccountTransfer moves perp USDC to or from a sub-account.
func (c *ExchangeClient) SubAccountTransfer(ctx context.Context, a SubAccountTransferAction) error {
	return c.do(ctx, a, nil)
}

// SubAccountSpotTransferAction moves a spot token between the account and a
// sub-account.
type SubAccountSpotTransferAction struct {
	SubAccountUser Address `json:"subAccountUser"`
	// IsDeposit moves funds into the sub-account; otherwise out of it.
	IsDeposit bool `json:"isDeposit"`
	// Token is "<name>:<tokenId>", e.g. "USDC:0xeb62eee3685fc4c43992febcd9e75443".
	Token  string  `json:"token"`
	Amount Decimal `json:"amount"`
}

func (SubAccountSpotTransferAction) actionType() string { return "subAccountSpotTransfer" }

// SubAccountSpotTransfer moves a spot token to or from a sub-account.
func (c *ExchangeClient) SubAccountSpotTransfer(ctx context.Context, a SubAccountSpotTransferAction) error {
	return c.do(ctx, a, nil)
}

// AgentSendAssetAction moves a token between perp dexes, spot and accounts,
// like [SendAssetAction], but may be signed by an API wallet.
type AgentSendAssetAction struct {
	Destination Address `json:"destination"`
	// SourceDex and DestinationDex are perp dex names; "" is the default
	// perp dex and "spot" is the spot balance.
	SourceDex      string `json:"sourceDex"`
	DestinationDex string `json:"destinationDex"`
	// Token is "<name>:<tokenId>", e.g. "USDC:0xeb62eee3685fc4c43992febcd9e75443".
	Token  string  `json:"token"`
	Amount Decimal `json:"amount"`
	// FromSubAccount optionally sends from a sub-account.
	FromSubAccount *Address `json:"fromSubAccount"`
}

func (AgentSendAssetAction) actionType() string { return "agentSendAsset" }
func (AgentSendAssetAction) l1Nonce()           {}

// MarshalJSON encodes a nil FromSubAccount as "".
func (a AgentSendAssetAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(sendAssetWire{a.Destination, a.SourceDex, a.DestinationDex, a.Token, a.Amount, subAccountWire(a.FromSubAccount)})
}

// AgentSendAsset moves a token between perp dexes, spot and accounts.
func (c *ExchangeClient) AgentSendAsset(ctx context.Context, a AgentSendAssetAction) error {
	return c.do(ctx, a, nil)
}

// AgentSetAbstractionAction sets the account abstraction mode, like
// [UserSetAbstractionAction], but may be signed by an API wallet.
type AgentSetAbstractionAction struct {
	Abstraction Abstraction `json:"abstraction"`
}

func (AgentSetAbstractionAction) actionType() string { return "agentSetAbstraction" }

// MarshalJSON encodes the abstraction in its short wire form ("i", "u", "p").
func (a AgentSetAbstractionAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Abstraction string `json:"abstraction"`
	}{a.Abstraction.wireCode()})
}

// AgentSetAbstraction sets the account abstraction mode.
func (c *ExchangeClient) AgentSetAbstraction(ctx context.Context, a AgentSetAbstractionAction) error {
	return c.do(ctx, a, nil)
}

// SpotUserAction changes spot account settings. Set exactly one field.
type SpotUserAction struct {
	ToggleSpotDusting *ToggleSpotDusting `json:"toggleSpotDusting,omitempty"`
}

func (SpotUserAction) actionType() string { return "spotUser" }

// ToggleSpotDusting opts the account out of (or back into) the automatic
// conversion of spot dust.
type ToggleSpotDusting struct {
	OptOut bool `json:"optOut"`
}

// SpotUser changes spot account settings.
func (c *ExchangeClient) SpotUser(ctx context.Context, a SpotUserAction) error {
	return c.do(ctx, a, nil)
}

// UserOutcomeAction splits, merges or negates HIP-4 outcome shares. Set
// exactly one field.
type UserOutcomeAction struct {
	SplitOutcome  *SplitOutcome  `json:"splitOutcome,omitempty"`
	MergeOutcome  *MergeOutcome  `json:"mergeOutcome,omitempty"`
	MergeQuestion *MergeQuestion `json:"mergeQuestion,omitempty"`
	NegateOutcome *NegateOutcome `json:"negateOutcome,omitempty"`
}

func (UserOutcomeAction) actionType() string { return "userOutcome" }

// SplitOutcome splits Amount quote tokens into Amount Yes and Amount No
// shares of an outcome.
type SplitOutcome struct {
	Outcome int     `json:"outcome"`
	Amount  Decimal `json:"amount"`
}

// MergeOutcome merges Yes and No shares of an outcome back into quote
// tokens.
type MergeOutcome struct {
	Outcome int `json:"outcome"`
	// Amount is the number of share pairs; nil merges the maximum.
	Amount *Decimal `json:"amount"`
}

// MergeQuestion merges Yes shares of every outcome of a question into quote
// tokens.
type MergeQuestion struct {
	Question int `json:"question"`
	// Amount is the number of share sets; nil merges the maximum.
	Amount *Decimal `json:"amount"`
}

// NegateOutcome converts No shares of an outcome into Yes shares of every
// other outcome of its question.
type NegateOutcome struct {
	Question int     `json:"question"`
	Outcome  int     `json:"outcome"`
	Amount   Decimal `json:"amount"`
}

// UserOutcome splits, merges or negates outcome shares.
func (c *ExchangeClient) UserOutcome(ctx context.Context, a UserOutcomeAction) error {
	return c.do(ctx, a, nil)
}
