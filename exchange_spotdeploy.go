package hyperliquid

import (
	"context"
	"encoding/json"
)

// SpotDeployAction deploys and manages HIP-1 spot tokens and HIP-2
// hyperliquidity. Set exactly one field.
//
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/deploying-hip-1-and-hip-2-assets.
type SpotDeployAction struct {
	// RegisterToken2 registers a token, bidding in the deploy auction.
	RegisterToken2 *RegisterToken2 `json:"registerToken2,omitempty"`
	// UserGenesis assigns genesis balances; it may be sent repeatedly
	// before Genesis.
	UserGenesis *UserGenesis `json:"userGenesis,omitempty"`
	// Genesis finalizes the token's supply.
	Genesis *SpotGenesis `json:"genesis,omitempty"`
	// RegisterSpot registers a trading pair.
	RegisterSpot *RegisterSpot `json:"registerSpot,omitempty"`
	// RegisterHyperliquidity seeds a pair with the HIP-2 market maker.
	RegisterHyperliquidity *RegisterHyperliquidity `json:"registerHyperliquidity,omitempty"`
	// SetDeployerTradingFeeShare sets the deployer's share of trading fees.
	SetDeployerTradingFeeShare *SetDeployerTradingFeeShare `json:"setDeployerTradingFeeShare,omitempty"`
	// EnableQuoteToken makes a token usable as a quote token.
	EnableQuoteToken *SpotDeployToken `json:"enableQuoteToken,omitempty"`
	// DisableQuoteToken stops a token being usable as a quote token.
	DisableQuoteToken *SpotDeployToken `json:"disableQuoteToken,omitempty"`
	// EnableFreezePrivilege lets the deployer freeze users' balances; it must
	// be sent before Genesis.
	EnableFreezePrivilege *SpotDeployToken `json:"enableFreezePrivilege,omitempty"`
	// FreezeUser freezes or unfreezes a user's balance of the token.
	FreezeUser *FreezeUser `json:"freezeUser,omitempty"`
	// RevokeFreezePrivilege permanently gives up the freeze privilege.
	RevokeFreezePrivilege *SpotDeployToken `json:"revokeFreezePrivilege,omitempty"`
	// RequestEVMContract requests a link to an ERC-20 contract on the
	// HyperEVM, completed with [ExchangeClient.FinalizeEVMContract].
	RequestEVMContract *RequestEVMContract `json:"requestEvmContract,omitempty"`
	// SetTokenAnnotation sets the token's display metadata, at most once
	// per day.
	SetTokenAnnotation *SetTokenAnnotation `json:"setTokenAnnotation,omitempty"`
	// SetDeployerLabel sets the label displayed for all of the deployer's
	// tokens, once per deployer.
	SetDeployerLabel *SetDeployerLabel `json:"setDeployerLabel,omitempty"`
}

func (SpotDeployAction) actionType() string { return "spotDeploy" }

// SpotDeploy submits a spot deployer action.
func (c *ExchangeClient) SpotDeploy(ctx context.Context, a SpotDeployAction) error {
	return c.do(ctx, a, nil)
}

// RegisterToken2 registers a spot token.
type RegisterToken2 struct {
	Spec TokenSpec `json:"spec"`
	// MaxGas is the most to pay in the deploy auction, in HYPE wei (1e-8).
	MaxGas   uint64 `json:"maxGas"`
	FullName string `json:"fullName,omitempty"`
}

// TokenSpec describes a new spot token.
type TokenSpec struct {
	Name        string `json:"name"`
	SzDecimals  int    `json:"szDecimals"`
	WeiDecimals int    `json:"weiDecimals"`
}

// UserGenesis assigns genesis balances of a token, in wei.
type UserGenesis struct {
	Token               int                `json:"token"`
	UserAndWei          []UserWei          `json:"userAndWei"`
	ExistingTokenAndWei []ExistingTokenWei `json:"existingTokenAndWei"`
	// BlacklistUsers adds users to, or removes them from, the genesis
	// blacklist.
	BlacklistUsers []BlacklistUser `json:"blacklistUsers,omitempty"`
}

// MarshalJSON encodes nil required lists as [].
func (g UserGenesis) MarshalJSON() ([]byte, error) {
	type plain UserGenesis
	g.UserAndWei = orEmpty(g.UserAndWei)
	g.ExistingTokenAndWei = orEmpty(g.ExistingTokenAndWei)
	return json.Marshal(plain(g))
}

// UserWei is a genesis balance of a user, encoded as [user, wei].
type UserWei struct {
	User Address
	Wei  Decimal
}

// MarshalJSON encodes u as [user, wei].
func (u UserWei) MarshalJSON() ([]byte, error) { return json.Marshal([2]any{u.User, u.Wei}) }

// ExistingTokenWei gives the holders of an existing token a genesis
// balance, distributed pro rata. It encodes as [token, wei].
type ExistingTokenWei struct {
	Token int
	Wei   Decimal
}

// MarshalJSON encodes e as [token, wei].
func (e ExistingTokenWei) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]any{e.Token, e.Wei})
}

// BlacklistUser blacklists a user from genesis, or removes them from the
// blacklist. It encodes as [user, blacklisted].
type BlacklistUser struct {
	User        Address
	Blacklisted bool
}

// MarshalJSON encodes b as [user, blacklisted].
func (b BlacklistUser) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]any{b.User, b.Blacklisted})
}

// SpotGenesis finalizes a token's supply.
type SpotGenesis struct {
	Token     int     `json:"token"`
	MaxSupply Decimal `json:"maxSupply"`
	// NoHyperliquidity sets the hyperliquidity balance to zero.
	NoHyperliquidity bool `json:"noHyperliquidity,omitempty"`
}

// RegisterSpot registers a spot pair.
type RegisterSpot struct {
	// Tokens are the base and quote token indices.
	Tokens [2]int `json:"tokens"`
}

// RegisterHyperliquidity seeds a spot pair with the HIP-2 market maker.
type RegisterHyperliquidity struct {
	// Spot is the spot pair index, not the token index.
	Spot    int     `json:"spot"`
	StartPx Decimal `json:"startPx"`
	// OrderSz is the size of each order, in tokens (not wei).
	OrderSz Decimal `json:"orderSz"`
	NOrders int     `json:"nOrders"`
	// NSeededLevels is the number of levels to seed with USDC.
	NSeededLevels *int `json:"nSeededLevels,omitempty"`
}

// SetDeployerTradingFeeShare sets the deployer's share of a token's trading
// fees.
type SetDeployerTradingFeeShare struct {
	Token int `json:"token"`
	// Share is a percentage between 0 and 100, such as "12.5".
	Share Decimal `json:"share"`
}

// MarshalJSON encodes Share with the trailing "%" the exchange expects.
func (s SetDeployerTradingFeeShare) MarshalJSON() ([]byte, error) {
	share, err := percent(s.Share)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Token int    `json:"token"`
		Share string `json:"share"`
	}{s.Token, share})
}

// SpotDeployToken names the token of a token-only [SpotDeployAction]
// variant.
type SpotDeployToken struct {
	Token int `json:"token"`
}

// FreezeUser freezes or unfreezes a user's balance of a token.
type FreezeUser struct {
	Token  int     `json:"token"`
	User   Address `json:"user"`
	Freeze bool    `json:"freeze"`
}

// RequestEVMContract requests a link between a spot token and an ERC-20
// contract on the HyperEVM.
type RequestEVMContract struct {
	Token   int     `json:"token"`
	Address Address `json:"address"`
	// EVMExtraWeiDecimals is the EVM token's decimals minus the spot
	// token's wei decimals, between -2 and 18.
	EVMExtraWeiDecimals int `json:"evmExtraWeiDecimals"`
}

// SetTokenAnnotation sets a spot token's display metadata.
type SetTokenAnnotation struct {
	Token      int             `json:"token"`
	Annotation TokenAnnotation `json:"annotation"`
}

// TokenAnnotation is a spot token's display metadata.
type TokenAnnotation struct {
	// Category is a classification label of at most 15 characters.
	Category string `json:"category"`
	// Description is at most 400 characters.
	Description string `json:"description"`
	// DisplayName of at most 9 characters replaces the token name in
	// frontends; nil leaves it unset.
	DisplayName *string `json:"displayName"`
	// Keywords are at most 2 search hints of at most 10 characters each.
	Keywords []string `json:"keywords"`
}

// MarshalJSON encodes a nil Keywords as [].
func (a TokenAnnotation) MarshalJSON() ([]byte, error) {
	type plain TokenAnnotation
	a.Keywords = orEmpty(a.Keywords)
	return json.Marshal(plain(a))
}

// SetDeployerLabel sets the label shown for a deployer's tokens.
type SetDeployerLabel struct {
	// Label is 2-4 lowercase characters, unique across deployer labels and
	// perp dexes.
	Label string `json:"label"`
}
