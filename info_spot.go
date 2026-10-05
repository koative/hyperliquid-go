package hyperliquid

import "context"

// SpotMeta describes the spot tokens and trading pairs.
type SpotMeta struct {
	Universe []SpotPairMeta  `json:"universe"`
	Tokens   []SpotTokenMeta `json:"tokens"`
}

// SpotPairMeta describes a spot trading pair. Its asset ID is 10000 + Index.
type SpotPairMeta struct {
	// Tokens holds the base and quote token indices.
	Tokens [2]int `json:"tokens"`
	// Name is "BASE/QUOTE" for a few canonical pairs and "@<Index>" otherwise;
	// it is the coin name used by info endpoints and subscriptions.
	Name        string `json:"name"`
	Index       int    `json:"index"`
	IsCanonical bool   `json:"isCanonical"`
}

// SpotTokenMeta describes a spot token.
type SpotTokenMeta struct {
	Name        string `json:"name"`
	SzDecimals  int    `json:"szDecimals"`
	WeiDecimals int    `json:"weiDecimals"`
	Index       int    `json:"index"`
	// TokenID is the 0x-prefixed 32-hex-digit token ID.
	TokenID     string       `json:"tokenId"`
	IsCanonical bool         `json:"isCanonical"`
	EvmContract *EvmContract `json:"evmContract"`
	FullName    *string      `json:"fullName"`
	// DeployerTradingFeeShare is the fraction of trading fees paid to the
	// token deployer.
	DeployerTradingFeeShare Decimal `json:"deployerTradingFeeShare"`
	// DeployerLabel is the deployer's display label, if set.
	DeployerLabel string `json:"deployerLabel,omitempty"`
}

// EvmContract is the HyperEVM contract linked to a spot token.
type EvmContract struct {
	Address Address `json:"address"`
	// EvmExtraWeiDecimals is the EVM token's decimals minus the spot token's
	// wei decimals.
	EvmExtraWeiDecimals int `json:"evm_extra_wei_decimals"`
}

// SpotMeta returns spot metadata.
func (c *InfoClient) SpotMeta(ctx context.Context) (*SpotMeta, error) {
	return infoRequest[*SpotMeta](ctx, c, "spotMeta", noParams{})
}

// SpotAssetCtx is the live market state of a spot pair.
type SpotAssetCtx struct {
	PrevDayPx         Decimal  `json:"prevDayPx"`
	DayNtlVlm         Decimal  `json:"dayNtlVlm"`
	MarkPx            Decimal  `json:"markPx"`
	MidPx             *Decimal `json:"midPx"`
	CirculatingSupply Decimal  `json:"circulatingSupply"`
	// Coin is the pair name, as in [SpotPairMeta.Name].
	Coin        string  `json:"coin"`
	TotalSupply Decimal `json:"totalSupply"`
	DayBaseVlm  Decimal `json:"dayBaseVlm"`
}

// SpotMetaAndAssetCtxs is spot metadata with the market state of each pair.
type SpotMetaAndAssetCtxs struct {
	Meta SpotMeta
	// AssetCtxs is parallel to Meta.Universe.
	AssetCtxs []SpotAssetCtx
}

// UnmarshalJSON decodes the wire form [meta, assetCtxs].
func (m *SpotMetaAndAssetCtxs) UnmarshalJSON(b []byte) error {
	return unmarshalTuple(b, &m.Meta, &m.AssetCtxs)
}

// SpotMetaAndAssetCtxs returns spot metadata and asset contexts.
func (c *InfoClient) SpotMetaAndAssetCtxs(ctx context.Context) (*SpotMetaAndAssetCtxs, error) {
	return infoRequest[*SpotMetaAndAssetCtxs](ctx, c, "spotMetaAndAssetCtxs", noParams{})
}

// TokenDetailsRequest is the request for [InfoClient.TokenDetails].
type TokenDetailsRequest struct {
	// TokenID is a token ID from [SpotTokenMeta.TokenID].
	TokenID string `json:"tokenId"`
}

// TokenDetails describes a spot token's supply and deployment.
type TokenDetails struct {
	Name              string  `json:"name"`
	MaxSupply         Decimal `json:"maxSupply"`
	TotalSupply       Decimal `json:"totalSupply"`
	CirculatingSupply Decimal `json:"circulatingSupply"`
	SzDecimals        int     `json:"szDecimals"`
	WeiDecimals       int     `json:"weiDecimals"`
	MidPx             Decimal `json:"midPx"`
	MarkPx            Decimal `json:"markPx"`
	PrevDayPx         Decimal `json:"prevDayPx"`
	// Genesis is nil for tokens without a genesis distribution.
	Genesis                    *TokenGenesis              `json:"genesis"`
	Deployer                   *Address                   `json:"deployer"`
	DeployGas                  *Decimal                   `json:"deployGas"`
	DeployTime                 *string                    `json:"deployTime"`
	SeededUsdc                 Decimal                    `json:"seededUsdc"`
	NonCirculatingUserBalances TupleMap[Address, Decimal] `json:"nonCirculatingUserBalances"`
	FutureEmissions            Decimal                    `json:"futureEmissions"`
}

// TokenGenesis is a token's genesis distribution.
type TokenGenesis struct {
	UserBalances TupleMap[Address, Decimal] `json:"userBalances"`
	// ExistingTokenBalances maps token indices to the balance granted to
	// holders of that token.
	ExistingTokenBalances TupleMap[int, Decimal] `json:"existingTokenBalances"`
	BlacklistUsers        []Address              `json:"blacklistUsers"`
}

// TokenDetails returns details of a spot token.
func (c *InfoClient) TokenDetails(ctx context.Context, req TokenDetailsRequest) (*TokenDetails, error) {
	return infoRequest[*TokenDetails](ctx, c, "tokenDetails", req)
}
