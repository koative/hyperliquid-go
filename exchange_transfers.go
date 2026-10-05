package hyperliquid

import "context"

// UsdSendAction sends USDC from the perp balance to another address on
// Hyperliquid. It must be signed by the account's own key.
type UsdSendAction struct {
	Destination Address `json:"destination"`
	Amount      Decimal `json:"amount"`
}

func (UsdSendAction) actionType() string { return "usdSend" }

var usdSendSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:UsdSend",
	Fields:      []typedField{{"destination", "string"}, {"amount", "string"}},
	NonceField:  "time",
}

func (UsdSendAction) userSignedSpec() *userSignedSpec { return &usdSendSpec }

// UsdSend sends USDC to another address.
func (c *ExchangeClient) UsdSend(ctx context.Context, a UsdSendAction) error {
	return c.do(ctx, a, nil)
}
