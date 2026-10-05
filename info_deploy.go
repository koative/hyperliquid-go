package hyperliquid

import "context"

// AuctionStatus is the state of a Dutch gas auction, such as the one for
// deploying a perp dex asset or a spot pair.
type AuctionStatus struct {
	// StartTimeSeconds is in seconds since the Unix epoch.
	StartTimeSeconds int64   `json:"startTimeSeconds"`
	DurationSeconds  int64   `json:"durationSeconds"`
	StartGas         Decimal `json:"startGas"`
	// CurrentGas is nil once the auction has been won.
	CurrentGas *Decimal `json:"currentGas"`
	// EndGas is the winning price, or nil while the auction runs.
	EndGas *Decimal `json:"endGas"`
}

// PerpDeployAuctionStatus returns the auction for deploying a perp dex
// asset.
func (c *InfoClient) PerpDeployAuctionStatus(ctx context.Context) (*AuctionStatus, error) {
	return infoRequest[*AuctionStatus](ctx, c, "perpDeployAuctionStatus", noParams{})
}

// SpotPairDeployAuctionStatus returns the auction for deploying a spot pair.
func (c *InfoClient) SpotPairDeployAuctionStatus(ctx context.Context) (*AuctionStatus, error) {
	return infoRequest[*AuctionStatus](ctx, c, "spotPairDeployAuctionStatus", noParams{})
}

// SpotDeployStateRequest is the request for [InfoClient.SpotDeployState].
type SpotDeployStateRequest struct {
	User Address `json:"user"`
}

// SpotDeployState is a deployer's in-progress spot token deployments.
type SpotDeployState struct {
	States     []SpotDeployTokenState `json:"states"`
	GasAuction AuctionStatus          `json:"gasAuction"`
}

// SpotDeployTokenState is the deployment state of one spot token; see
// [SpotDeployAction] for the steps.
type SpotDeployTokenState struct {
	Token                   int       `json:"token"`
	Spec                    TokenSpec `json:"spec"`
	FullName                *string   `json:"fullName"`
	DeployerTradingFeeShare Decimal   `json:"deployerTradingFeeShare"`
	// Spots are the indices of the token's spot pairs.
	Spots                        []int    `json:"spots"`
	MaxSupply                    *Decimal `json:"maxSupply"`
	HyperliquidityGenesisBalance Decimal  `json:"hyperliquidityGenesisBalance"`
	TotalGenesisBalanceWei       Decimal  `json:"totalGenesisBalanceWei"`
	// UserGenesisBalances and ExistingTokenGenesisBalances are in wei.
	UserGenesisBalances          TupleMap[Address, Decimal] `json:"userGenesisBalances"`
	ExistingTokenGenesisBalances TupleMap[int, Decimal]     `json:"existingTokenGenesisBalances"`
	BlacklistUsers               []Address                  `json:"blacklistUsers"`
}

// SpotDeployState returns a user's spot deployment state and the current
// token deploy auction.
func (c *InfoClient) SpotDeployState(ctx context.Context, req SpotDeployStateRequest) (*SpotDeployState, error) {
	return infoRequest[*SpotDeployState](ctx, c, "spotDeployState", req)
}
