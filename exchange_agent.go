package hyperliquid

import (
	"context"
	"encoding/json"
)

// ApproveAgentAction authorizes an API (agent) wallet to sign L1 actions,
// such as orders, on behalf of the signer. It must be signed by the
// account's own key. Create the agent key with [NewPrivateKeySigner] and
// sign with it via [NewExchangeClient].
type ApproveAgentAction struct {
	AgentAddress Address `json:"agentAddress"`
	// AgentName names the agent (at most 16 characters, optionally followed
	// by " valid_until <ms>"). Approving a name again replaces its agent.
	// An empty name approves the unnamed agent.
	AgentName string `json:"agentName"`
}

func (ApproveAgentAction) actionType() string { return "approveAgent" }

var approveAgentSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:ApproveAgent",
	Fields:      []typedField{{"agentAddress", "address"}, {"agentName", "string"}},
	NonceField:  "nonce",
}

func (ApproveAgentAction) userSignedSpec() *userSignedSpec { return &approveAgentSpec }

// ApproveAgent authorizes an API wallet.
func (c *ExchangeClient) ApproveAgent(ctx context.Context, a ApproveAgentAction) error {
	return c.do(ctx, a, nil)
}

// ApproveBuilderFeeAction allows a builder to charge the signer up to
// MaxFeeRate on orders that name it in [OrderAction.Builder]. It must be
// signed by the account's own key.
type ApproveBuilderFeeAction struct {
	// MaxFeeRate is the maximum fee as a percentage of notional, such as
	// "0.001" for 0.001%. It is sent with the "%" suffix.
	MaxFeeRate Decimal `json:"maxFeeRate"`
	Builder    Address `json:"builder"`
}

func (ApproveBuilderFeeAction) actionType() string { return "approveBuilderFee" }

// MarshalJSON encodes MaxFeeRate with the trailing "%" the exchange expects.
func (a ApproveBuilderFeeAction) MarshalJSON() ([]byte, error) {
	rate, err := percent(a.MaxFeeRate)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		MaxFeeRate string  `json:"maxFeeRate"`
		Builder    Address `json:"builder"`
	}{rate, a.Builder})
}

var approveBuilderFeeSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:ApproveBuilderFee",
	Fields:      []typedField{{"maxFeeRate", "string"}, {"builder", "address"}},
	NonceField:  "nonce",
}

func (ApproveBuilderFeeAction) userSignedSpec() *userSignedSpec { return &approveBuilderFeeSpec }

// ApproveBuilderFee approves a builder's maximum fee rate.
func (c *ExchangeClient) ApproveBuilderFee(ctx context.Context, a ApproveBuilderFeeAction) error {
	return c.do(ctx, a, nil)
}
