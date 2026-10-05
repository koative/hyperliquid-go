package hyperliquid

import (
	"context"
	"encoding/json"
)

// ValidatorSummary describes a validator.
type ValidatorSummary struct {
	Validator     Address `json:"validator"`
	Signer        Address `json:"signer"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	NRecentBlocks int     `json:"nRecentBlocks"`
	// Stake is in HYPE wei (1e-8 HYPE).
	Stake    int64 `json:"stake"`
	IsJailed bool  `json:"isJailed"`
	// UnjailableAfter is in milliseconds since the Unix epoch, or nil.
	UnjailableAfter *int64  `json:"unjailableAfter"`
	IsActive        bool    `json:"isActive"`
	Commission      Decimal `json:"commission"`
	// Stats holds the "day", "week" and "month" statistics, in that order.
	Stats []ValidatorStats `json:"stats"`
}

// ValidatorStats are a validator's statistics over a period.
type ValidatorStats struct {
	// Period is "day", "week" or "month".
	Period         string  `json:"-"`
	UptimeFraction Decimal `json:"uptimeFraction"`
	PredictedAPR   Decimal `json:"predictedApr"`
	NSamples       int     `json:"nSamples"`
}

// UnmarshalJSON decodes the wire form [period, stats].
func (s *ValidatorStats) UnmarshalJSON(b []byte) error {
	type plain ValidatorStats
	return unmarshalTuple(b, &s.Period, (*plain)(s))
}

// ValidatorSummaries returns all validators.
func (c *InfoClient) ValidatorSummaries(ctx context.Context) ([]ValidatorSummary, error) {
	return infoRequest[[]ValidatorSummary](ctx, c, "validatorSummaries", noParams{})
}

// ValidatorL1Vote is a pending L1 governance vote among validators.
type ValidatorL1Vote struct {
	// ExpireTime is in milliseconds since the Unix epoch.
	ExpireTime int64 `json:"expireTime"`
	// Action is the voted-on action, a single-key object such as {"D": ...},
	// {"C": [...]}, {"E": {...}} or {"O": {...}} whose payloads are not
	// documented.
	Action        json.RawMessage `json:"action"`
	Votes         []Address       `json:"votes"`
	QuorumReached bool            `json:"quorumReached"`
}

// ValidatorL1Votes returns the pending validator L1 votes.
func (c *InfoClient) ValidatorL1Votes(ctx context.Context) ([]ValidatorL1Vote, error) {
	return infoRequest[[]ValidatorL1Vote](ctx, c, "validatorL1Votes", noParams{})
}

// GossipRootIPs returns the IPv4 addresses of the gossip root peers that
// non-validating nodes connect to.
func (c *InfoClient) GossipRootIPs(ctx context.Context) ([]string, error) {
	return infoRequest[[]string](ctx, c, "gossipRootIps", noParams{})
}

// GossipPriorityAuctionStatus is the state of the gossip priority slot
// auctions.
type GossipPriorityAuctionStatus struct {
	// IPs holds the IP address that won each slot, or nil for an empty slot.
	IPs []*string
	// Auctions holds the auction of each slot.
	Auctions []AuctionStatus
}

// UnmarshalJSON decodes the wire form [ips, auctions].
func (s *GossipPriorityAuctionStatus) UnmarshalJSON(b []byte) error {
	return unmarshalTuple(b, &s.IPs, &s.Auctions)
}

// GossipPriorityAuctionStatus returns the gossip priority slot auctions.
func (c *InfoClient) GossipPriorityAuctionStatus(ctx context.Context) (*GossipPriorityAuctionStatus, error) {
	return infoRequest[*GossipPriorityAuctionStatus](ctx, c, "gossipPriorityAuctionStatus", noParams{})
}
