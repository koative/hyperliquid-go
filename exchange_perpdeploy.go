package hyperliquid

import (
	"context"
	"encoding/json"
)

// PerpDeployAction deploys and manages HIP-3 perpetual dexes and assets.
// Set exactly one field. Only the dex deployer (or an authorized
// sub-deployer, see [SetSubDeployers]) may submit it.
//
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/hip-3-deployer-actions.
type PerpDeployAction struct {
	// RegisterAsset registers an asset; superseded by RegisterAsset2.
	RegisterAsset *RegisterPerpAsset `json:"registerAsset,omitempty"`
	// RegisterAsset2 registers an asset, and the dex itself when Schema is set.
	RegisterAsset2 *RegisterPerpAsset2 `json:"registerAsset2,omitempty"`
	// SetOracle updates oracle and mark prices.
	SetOracle *SetOracle `json:"setOracle,omitempty"`
	// SetFundingMultipliers maps coins to funding multipliers between 0
	// and 10.
	SetFundingMultipliers TupleMap[string, Decimal] `json:"setFundingMultipliers,omitempty"`
	// SetFundingInterestRates maps coins to 8-hour funding interest rates
	// between -0.01 and 0.01.
	SetFundingInterestRates TupleMap[string, Decimal] `json:"setFundingInterestRates,omitempty"`
	// SetFundingClamps maps coins to 8-hour funding clamps between 0 and
	// 0.01, which bound how far funding can move from the average premium
	// toward the interest rate (default 0.0003).
	SetFundingClamps TupleMap[string, Decimal] `json:"setFundingClamps,omitempty"`
	// HaltTrading halts or resumes trading of an asset.
	HaltTrading *HaltTrading `json:"haltTrading,omitempty"`
	// SetMarginTableIDs maps coins to margin table IDs.
	SetMarginTableIDs TupleMap[string, int] `json:"setMarginTableIds,omitempty"`
	// InsertMarginTable adds a margin table to a dex.
	InsertMarginTable *InsertMarginTable `json:"insertMarginTable,omitempty"`
	// SetFeeRecipient sets the dex's fee recipient.
	SetFeeRecipient *SetFeeRecipient `json:"setFeeRecipient,omitempty"`
	// SetOpenInterestCaps maps coins to open interest caps, in quote
	// notional; nil removes an asset's custom cap.
	SetOpenInterestCaps TupleMap[string, *uint64] `json:"setOpenInterestCaps,omitempty"`
	// SetSubDeployers grants or revokes sub-deployer permissions.
	SetSubDeployers *SetSubDeployers `json:"setSubDeployers,omitempty"`
	// SetMarginModes maps coins to margin modes.
	SetMarginModes TupleMap[string, MarginMode] `json:"setMarginModes,omitempty"`
	// SetDeployerFees maps coins to deployer fee settings.
	SetDeployerFees TupleMap[string, DeployerFee] `json:"setDeployerFees,omitempty"`
	// SetPerpAnnotation sets an asset's display metadata.
	SetPerpAnnotation *SetPerpAnnotation `json:"setPerpAnnotation,omitempty"`
	// DisableDex names a dex to disable.
	DisableDex string `json:"disableDex,omitempty"`
}

func (PerpDeployAction) actionType() string { return "perpDeploy" }

// PerpDeploy submits a HIP-3 deployer action.
func (c *ExchangeClient) PerpDeploy(ctx context.Context, a PerpDeployAction) error {
	return c.do(ctx, a, nil)
}

// MarginMode restricts how an asset can be margined.
type MarginMode string

// Margin modes.
const (
	MarginModeNormal MarginMode = "normal"
	// MarginModeStrictIsolated allows isolated margin only, and margin
	// cannot be removed from open positions.
	MarginModeStrictIsolated MarginMode = "strictIsolated"
	// MarginModeNoCross allows isolated margin only.
	MarginModeNoCross MarginMode = "noCross"
)

// RegisterPerpAsset is the legacy asset registration of a
// [PerpDeployAction]; prefer [RegisterPerpAsset2].
type RegisterPerpAsset struct {
	// MaxGas is the most to pay in the deploy auction, in HYPE wei (1e-8);
	// nil pays the current auction price.
	MaxGas       *uint64          `json:"maxGas"`
	AssetRequest PerpAssetRequest `json:"assetRequest"`
	Dex          string           `json:"dex"`
	// Schema registers the dex itself with the first asset; nil otherwise.
	Schema *PerpDexSchema `json:"schema"`
}

// PerpAssetRequest describes a new asset of a [RegisterPerpAsset].
type PerpAssetRequest struct {
	Coin          string  `json:"coin"`
	SzDecimals    int     `json:"szDecimals"`
	OraclePx      Decimal `json:"oraclePx"`
	MarginTableID int     `json:"marginTableId"`
	OnlyIsolated  bool    `json:"onlyIsolated"`
}

// RegisterPerpAsset2 registers a perpetual asset on a HIP-3 dex.
type RegisterPerpAsset2 struct {
	// MaxGas is the most to pay in the deploy auction, in HYPE wei (1e-8);
	// nil pays the current auction price.
	MaxGas       *uint64           `json:"maxGas"`
	AssetRequest PerpAssetRequest2 `json:"assetRequest"`
	Dex          string            `json:"dex"`
	// Schema registers the dex itself with the first asset; nil otherwise.
	Schema *PerpDexSchema `json:"schema"`
}

// PerpAssetRequest2 describes a new asset of a [RegisterPerpAsset2].
type PerpAssetRequest2 struct {
	Coin          string     `json:"coin"`
	SzDecimals    int        `json:"szDecimals"`
	OraclePx      Decimal    `json:"oraclePx"`
	MarginTableID int        `json:"marginTableId"`
	MarginMode    MarginMode `json:"marginMode"`
}

// PerpDexSchema describes a new HIP-3 dex.
type PerpDexSchema struct {
	FullName        string `json:"fullName"`
	CollateralToken int    `json:"collateralToken"`
	// OracleUpdater may update oracle prices; nil means the deployer.
	OracleUpdater *Address `json:"oracleUpdater"`
}

// SetOracle updates the prices of a dex's assets. Coins are full asset
// names such as "dex:BTC".
type SetOracle struct {
	Dex       string                    `json:"dex"`
	OraclePxs TupleMap[string, Decimal] `json:"oraclePxs"`
	// MarkPxs holds mark price sets; the exchange combines them with the
	// local mark price.
	MarkPxs []TupleMap[string, Decimal] `json:"markPxs"`
	// ExternalPerpPxs are external prices that limit sudden mark price
	// deviations.
	ExternalPerpPxs TupleMap[string, Decimal] `json:"externalPerpPxs"`
}

// MarshalJSON encodes a nil MarkPxs as [].
func (s SetOracle) MarshalJSON() ([]byte, error) {
	type plain SetOracle
	s.MarkPxs = orEmpty(s.MarkPxs)
	return json.Marshal(plain(s))
}

// HaltTrading halts or resumes trading of an asset.
type HaltTrading struct {
	Coin     string `json:"coin"`
	IsHalted bool   `json:"isHalted"`
}

// InsertMarginTable adds a margin table to a dex.
type InsertMarginTable struct {
	Dex         string          `json:"dex"`
	MarginTable MarginTableSpec `json:"marginTable"`
}

// MarginTableSpec is a margin table to insert with [InsertMarginTable].
type MarginTableSpec struct {
	Description string `json:"description"`
	// MarginTiers lists at most 3 tiers by increasing lower bound and
	// decreasing max leverage.
	MarginTiers []MarginTierSpec `json:"marginTiers"`
}

// MarginTierSpec is a tier of a [MarginTableSpec].
type MarginTierSpec struct {
	// LowerBound is the position notional above which MaxLeverage applies.
	LowerBound  uint64 `json:"lowerBound"`
	MaxLeverage int    `json:"maxLeverage"`
}

// SetFeeRecipient sets a dex's fee recipient.
type SetFeeRecipient struct {
	Dex          string  `json:"dex"`
	FeeRecipient Address `json:"feeRecipient"`
}

// SetSubDeployers changes which users may submit which [PerpDeployAction]
// variants on a dex.
type SetSubDeployers struct {
	Dex          string        `json:"dex"`
	SubDeployers []SubDeployer `json:"subDeployers"`
}

// SubDeployer grants or revokes one sub-deployer permission, in a
// [SetSubDeployers] or [OutcomeOperation.SetSubDeployers].
type SubDeployer struct {
	// Variant is the wire name of the action variant, such as "setOracle".
	Variant string  `json:"variant"`
	User    Address `json:"user"`
	Allowed bool    `json:"allowed"`
}

// DeployerFee sets the fees of an asset's trades: the deployer receives
// Scale times the user's normal fee, and the protocol fee rises to match
// when Scale exceeds 1. On mainnet it may change once per 30 days.
//
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/hip-3-deployer-actions.
type DeployerFee struct {
	// Scale is between 0 and 3, or between 0 and 10 (exclusive) in growth
	// mode.
	Scale Decimal `json:"scale"`
	// GrowthMode cuts all trading fees of the asset by 90%.
	GrowthMode bool `json:"growthMode"`
}

// SetPerpAnnotation sets display metadata for an asset.
type SetPerpAnnotation struct {
	Coin string `json:"coin"`
	// Category is a classification label of at most 15 characters.
	Category string `json:"category"`
	// Description is at most 400 characters.
	Description string `json:"description"`
	// DisplayName replaces the L1 name in frontends; nil leaves it unset.
	DisplayName *string `json:"displayName"`
	// Keywords are search hints.
	Keywords []string `json:"keywords"`
}

// MarshalJSON encodes a nil Keywords as [].
func (s SetPerpAnnotation) MarshalJSON() ([]byte, error) {
	type plain SetPerpAnnotation
	s.Keywords = orEmpty(s.Keywords)
	return json.Marshal(plain(s))
}

// orEmpty returns s, or an empty slice if s is nil, so that it encodes as []
// rather than null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
