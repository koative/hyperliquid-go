package hyperliquid

import "context"

// CDepositAction moves HYPE from the spot balance into staking, where it
// can be delegated with [TokenDelegateAction]. It must be signed by the
// account's own key.
type CDepositAction struct {
	// Wei is the amount in units of 1e-8 HYPE.
	Wei uint64 `json:"wei"`
}

func (CDepositAction) actionType() string { return "cDeposit" }

var cDepositSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:CDeposit",
	Fields:      []typedField{{"wei", "uint64"}},
	NonceField:  "nonce",
}

func (CDepositAction) userSignedSpec() *userSignedSpec { return &cDepositSpec }

// CDeposit moves HYPE from spot into staking.
func (c *ExchangeClient) CDeposit(ctx context.Context, a CDepositAction) error {
	return c.do(ctx, a, nil)
}

// CWithdrawAction moves undelegated HYPE from staking back to the spot
// balance, after an unstaking queue. It must be signed by the account's own
// key.
type CWithdrawAction struct {
	// Wei is the amount in units of 1e-8 HYPE.
	Wei uint64 `json:"wei"`
}

func (CWithdrawAction) actionType() string { return "cWithdraw" }

var cWithdrawSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:CWithdraw",
	Fields:      []typedField{{"wei", "uint64"}},
	NonceField:  "nonce",
}

func (CWithdrawAction) userSignedSpec() *userSignedSpec { return &cWithdrawSpec }

// CWithdraw moves HYPE from staking to spot.
func (c *ExchangeClient) CWithdraw(ctx context.Context, a CWithdrawAction) error {
	return c.do(ctx, a, nil)
}

// TokenDelegateAction delegates staked HYPE to a validator, or undelegates
// it. It must be signed by the account's own key.
type TokenDelegateAction struct {
	Validator Address `json:"validator"`
	// Wei is the amount in units of 1e-8 HYPE.
	Wei          uint64 `json:"wei"`
	IsUndelegate bool   `json:"isUndelegate"`
}

func (TokenDelegateAction) actionType() string { return "tokenDelegate" }

var tokenDelegateSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:TokenDelegate",
	Fields:      []typedField{{"validator", "address"}, {"wei", "uint64"}, {"isUndelegate", "bool"}},
	NonceField:  "nonce",
}

func (TokenDelegateAction) userSignedSpec() *userSignedSpec { return &tokenDelegateSpec }

// TokenDelegate delegates or undelegates staked HYPE.
func (c *ExchangeClient) TokenDelegate(ctx context.Context, a TokenDelegateAction) error {
	return c.do(ctx, a, nil)
}

// LinkStakingUserAction links a trading account to a staking account so
// the trading account gets the staking account's fee discount. The trading
// account sends it first naming the staking account, then the staking
// account finalizes it naming the trading account. It must be signed by the
// account's own key.
type LinkStakingUserAction struct {
	User       Address `json:"user"`
	IsFinalize bool    `json:"isFinalize"`
}

func (LinkStakingUserAction) actionType() string { return "linkStakingUser" }

var linkStakingUserSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:LinkStakingUser",
	Fields:      []typedField{{"user", "address"}, {"isFinalize", "bool"}},
	NonceField:  "nonce",
}

func (LinkStakingUserAction) userSignedSpec() *userSignedSpec { return &linkStakingUserSpec }

// LinkStakingUser requests or finalizes a staking link.
func (c *ExchangeClient) LinkStakingUser(ctx context.Context, a LinkStakingUserAction) error {
	return c.do(ctx, a, nil)
}

// StakingLinkDisableTradingUserAction permanently disables a trading
// account linked to the signing staking account and locks its funds, which
// move to the staking account after one year. It is irreversible and must
// be signed by the account's own key.
type StakingLinkDisableTradingUserAction struct {
	TradingUser Address `json:"tradingUser"`
}

func (StakingLinkDisableTradingUserAction) actionType() string {
	return "stakingLinkDisableTradingUser"
}

var stakingLinkDisableTradingUserSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:StakingLinkDisableTradingUser",
	Fields:      []typedField{{"tradingUser", "address"}},
	NonceField:  "nonce",
}

func (StakingLinkDisableTradingUserAction) userSignedSpec() *userSignedSpec {
	return &stakingLinkDisableTradingUserSpec
}

// StakingLinkDisableTradingUser permanently disables a linked trading
// account.
func (c *ExchangeClient) StakingLinkDisableTradingUser(ctx context.Context, a StakingLinkDisableTradingUserAction) error {
	return c.do(ctx, a, nil)
}
