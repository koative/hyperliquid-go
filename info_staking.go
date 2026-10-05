package hyperliquid

import "context"

// DelegationsRequest is the request for [InfoClient.Delegations].
type DelegationsRequest struct {
	User Address `json:"user"`
}

// Delegation is HYPE staked to a validator.
type Delegation struct {
	Validator Address `json:"validator"`
	Amount    Decimal `json:"amount"`
	// LockedUntilTimestamp is in milliseconds since the Unix epoch.
	LockedUntilTimestamp int64 `json:"lockedUntilTimestamp"`
}

// Delegations returns a user's staking delegations.
func (c *InfoClient) Delegations(ctx context.Context, req DelegationsRequest) ([]Delegation, error) {
	return infoRequest[[]Delegation](ctx, c, "delegations", req)
}

// DelegatorSummaryRequest is the request for [InfoClient.DelegatorSummary].
type DelegatorSummaryRequest struct {
	User Address `json:"user"`
}

// DelegatorSummary summarizes a user's staking balances.
type DelegatorSummary struct {
	Delegated              Decimal `json:"delegated"`
	Undelegated            Decimal `json:"undelegated"`
	TotalPendingWithdrawal Decimal `json:"totalPendingWithdrawal"`
	NPendingWithdrawals    int     `json:"nPendingWithdrawals"`
}

// DelegatorSummary returns a summary of a user's staking balances.
func (c *InfoClient) DelegatorSummary(ctx context.Context, req DelegatorSummaryRequest) (*DelegatorSummary, error) {
	return infoRequest[*DelegatorSummary](ctx, c, "delegatorSummary", req)
}

// DelegatorHistoryRequest is the request for [InfoClient.DelegatorHistory].
type DelegatorHistoryRequest struct {
	User Address `json:"user"`
}

// DelegatorEvent is a change to a user's staking balances.
type DelegatorEvent struct {
	// Time is in milliseconds since the Unix epoch.
	Time  int64          `json:"time"`
	Hash  string         `json:"hash"`
	Delta DelegatorDelta `json:"delta"`
}

// DelegatorDelta is the content of a [DelegatorEvent]; exactly one field is
// set.
type DelegatorDelta struct {
	Delegate   *DelegateDelta     `json:"delegate,omitempty"`
	CDeposit   *StakingDeposit    `json:"cDeposit,omitempty"`
	Withdrawal *StakingWithdrawal `json:"withdrawal,omitempty"`
}

// DelegateDelta is a delegation to or undelegation from a validator.
type DelegateDelta struct {
	Validator    Address `json:"validator"`
	Amount       Decimal `json:"amount"`
	IsUndelegate bool    `json:"isUndelegate"`
}

// StakingDeposit is a transfer from spot into the staking account.
type StakingDeposit struct {
	Amount Decimal `json:"amount"`
}

// StakingWithdrawal is a transfer from the staking account to spot.
type StakingWithdrawal struct {
	Amount Decimal `json:"amount"`
	// Phase is "initiated" or "finalized".
	Phase string `json:"phase"`
}

// DelegatorHistory returns a user's staking history.
func (c *InfoClient) DelegatorHistory(ctx context.Context, req DelegatorHistoryRequest) ([]DelegatorEvent, error) {
	return infoRequest[[]DelegatorEvent](ctx, c, "delegatorHistory", req)
}

// DelegatorRewardsRequest is the request for [InfoClient.DelegatorRewards].
type DelegatorRewardsRequest struct {
	User Address `json:"user"`
}

// DelegatorReward is a staking reward.
type DelegatorReward struct {
	// Time is in milliseconds since the Unix epoch.
	Time int64 `json:"time"`
	// Source is "delegation" or "commission".
	Source      string  `json:"source"`
	TotalAmount Decimal `json:"totalAmount"`
}

// DelegatorRewards returns a user's staking rewards.
func (c *InfoClient) DelegatorRewards(ctx context.Context, req DelegatorRewardsRequest) ([]DelegatorReward, error) {
	return infoRequest[[]DelegatorReward](ctx, c, "delegatorRewards", req)
}
