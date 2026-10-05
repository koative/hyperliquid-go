package hyperliquid

import "context"

// Abstraction is an account abstraction mode, which decides how balances
// are shared between spot and the perp dexes.
type Abstraction string

// Abstraction modes.
const (
	AbstractionDisabled        Abstraction = "disabled"
	AbstractionUnifiedAccount  Abstraction = "unifiedAccount"
	AbstractionPortfolioMargin Abstraction = "portfolioMargin"
	// AbstractionDefault is reported by [InfoClient] for accounts that
	// never set a mode; it cannot be set.
	AbstractionDefault Abstraction = "default"
)

// wireCode returns the one-letter form of a used by agentSetAbstraction
// and inside multi-sig payloads ("i", "u", "p"). Unknown modes are returned
// unchanged.
func (a Abstraction) wireCode() string {
	switch a {
	case AbstractionDisabled:
		return "i"
	case AbstractionUnifiedAccount:
		return "u"
	case AbstractionPortfolioMargin:
		return "p"
	default:
		return string(a)
	}
}

// UserSetAbstractionAction sets the abstraction mode of the signer or of
// one of its sub-accounts. It must be signed by the account's own key.
type UserSetAbstractionAction struct {
	User        Address     `json:"user"`
	Abstraction Abstraction `json:"abstraction"`
}

func (UserSetAbstractionAction) actionType() string { return "userSetAbstraction" }

var userSetAbstractionSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:UserSetAbstraction",
	Fields:      []typedField{{"user", "address"}, {"abstraction", "string"}},
	NonceField:  "nonce",
}

func (UserSetAbstractionAction) userSignedSpec() *userSignedSpec { return &userSetAbstractionSpec }

// multiSigPayload abbreviates the mode: signers sign the long form, but the
// multi-sig payload carries the one-letter code.
func (a UserSetAbstractionAction) multiSigPayload() Action {
	a.Abstraction = Abstraction(a.Abstraction.wireCode())
	return a
}

// UserSetAbstraction sets an account's abstraction mode.
func (c *ExchangeClient) UserSetAbstraction(ctx context.Context, a UserSetAbstractionAction) error {
	return c.do(ctx, a, nil)
}

// UserDexAbstractionAction enables or disables HIP-3 dex abstraction, which
// lets builder-deployed perp dexes use the spot balance as collateral. It
// must be signed by the account's own key.
//
// Deprecated: Use [UserSetAbstractionAction].
type UserDexAbstractionAction struct {
	User    Address `json:"user"`
	Enabled bool    `json:"enabled"`
}

func (UserDexAbstractionAction) actionType() string { return "userDexAbstraction" }

var userDexAbstractionSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:UserDexAbstraction",
	Fields:      []typedField{{"user", "address"}, {"enabled", "bool"}},
	NonceField:  "nonce",
}

func (UserDexAbstractionAction) userSignedSpec() *userSignedSpec { return &userDexAbstractionSpec }

// UserDexAbstraction enables or disables dex abstraction.
//
// Deprecated: Use [ExchangeClient.UserSetAbstraction].
func (c *ExchangeClient) UserDexAbstraction(ctx context.Context, a UserDexAbstractionAction) error {
	return c.do(ctx, a, nil)
}

// UserPortfolioMarginAction enables or disables portfolio margin for an
// account. It must be signed by the account's own key.
type UserPortfolioMarginAction struct {
	User    Address `json:"user"`
	Enabled bool    `json:"enabled"`
}

func (UserPortfolioMarginAction) actionType() string { return "userPortfolioMargin" }

var userPortfolioMarginSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:UserPortfolioMargin",
	Fields:      []typedField{{"user", "address"}, {"enabled", "bool"}},
	NonceField:  "nonce",
}

func (UserPortfolioMarginAction) userSignedSpec() *userSignedSpec { return &userPortfolioMarginSpec }

// UserPortfolioMargin enables or disables portfolio margin.
func (c *ExchangeClient) UserPortfolioMargin(ctx context.Context, a UserPortfolioMarginAction) error {
	return c.do(ctx, a, nil)
}
