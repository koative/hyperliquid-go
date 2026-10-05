package hyperliquid

import "context"

// ExchangeStatus is the status of the exchange.
type ExchangeStatus struct {
	// Time is the server time in milliseconds since the Unix epoch.
	Time int64 `json:"time"`
	// SpecialStatuses describes an exceptional state, such as an
	// upgrade, or is nil.
	SpecialStatuses *string `json:"specialStatuses"`
}

// ExchangeStatus returns the status of the exchange.
func (c *InfoClient) ExchangeStatus(ctx context.Context) (*ExchangeStatus, error) {
	return infoRequest[*ExchangeStatus](ctx, c, "exchangeStatus", noParams{})
}

// UsdcRouting is how USDC moves between Hyperliquid and Arbitrum.
type UsdcRouting struct {
	// DepositRoute and WithdrawalRoute are "bridge" or "cctp".
	DepositRoute    string `json:"depositRoute"`
	WithdrawalRoute string `json:"withdrawalRoute"`
}

// UsdcRouting returns the current USDC deposit and withdrawal routes.
func (c *InfoClient) UsdcRouting(ctx context.Context) (*UsdcRouting, error) {
	return infoRequest[*UsdcRouting](ctx, c, "usdcRouting", noParams{})
}
