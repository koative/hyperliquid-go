package hyperliquid

import (
	"context"
	"encoding/json"
)

// PerpDex describes a builder-deployed (HIP-3) perp dex.
type PerpDex struct {
	// Name prefixes the dex's coins, as in "xyz:TSLA".
	Name          string   `json:"name"`
	FullName      string   `json:"fullName"`
	Deployer      Address  `json:"deployer"`
	OracleUpdater *Address `json:"oracleUpdater"`
	FeeRecipient  *Address `json:"feeRecipient"`
	// AssetToStreamingOICap maps coins to their streaming open interest cap.
	AssetToStreamingOICap TupleMap[string, Decimal] `json:"assetToStreamingOiCap"`
	// SubDeployers maps delegated permissions to the addresses allowed to
	// use them on the deployer's behalf.
	SubDeployers               TupleMap[SubDeployerVariant, []Address] `json:"subDeployers"`
	AssetToFundingMultiplier   TupleMap[string, Decimal]               `json:"assetToFundingMultiplier"`
	AssetToFundingInterestRate TupleMap[string, Decimal]               `json:"assetToFundingInterestRate"`
	AssetToFundingClamp        TupleMap[string, Decimal]               `json:"assetToFundingClamp"`
}

// SubDeployerVariant is a permission delegated by a deployer: an action
// variant such as "setOracle" (the wire form "setOracle"), or, with HIP3Star
// set, a trading action such as "order" (the wire form {"hip3Star":"order"}).
type SubDeployerVariant struct {
	Name     string
	HIP3Star bool
}

// MarshalJSON encodes v in its wire form.
func (v SubDeployerVariant) MarshalJSON() ([]byte, error) {
	if v.HIP3Star {
		return json.Marshal(map[string]string{"hip3Star": v.Name})
	}
	return json.Marshal(v.Name)
}

// UnmarshalJSON decodes "name" or {"hip3Star":"name"}.
func (v *SubDeployerVariant) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '{' {
		var o struct {
			HIP3Star string `json:"hip3Star"`
		}
		err := json.Unmarshal(b, &o)
		*v = SubDeployerVariant{Name: o.HIP3Star, HIP3Star: true}
		return err
	}
	*v = SubDeployerVariant{}
	return json.Unmarshal(b, &v.Name)
}

// PerpDexs returns all perp dexes. Element 0 is nil and stands for the main
// dex; a builder dex's index is used in its asset IDs (see [Markets]).
func (c *InfoClient) PerpDexs(ctx context.Context) ([]*PerpDex, error) {
	return infoRequest[[]*PerpDex](ctx, c, "perpDexs", noParams{})
}

// PerpDexLimitsRequest is the request for [InfoClient.PerpDexLimits].
type PerpDexLimitsRequest struct {
	Dex string `json:"dex"`
}

// PerpDexLimits are the open interest and transfer limits of a builder dex.
type PerpDexLimits struct {
	TotalOICap     Decimal `json:"totalOiCap"`
	OISzCapPerPerp Decimal `json:"oiSzCapPerPerp"`
	MaxTransferNtl Decimal `json:"maxTransferNtl"`
	// CoinToOICap maps coins to their open interest cap.
	CoinToOICap TupleMap[string, Decimal] `json:"coinToOiCap"`
}

// PerpDexLimits returns the limits of a builder dex, or nil for the main dex
// and unknown dexes.
func (c *InfoClient) PerpDexLimits(ctx context.Context, req PerpDexLimitsRequest) (*PerpDexLimits, error) {
	return infoRequest[*PerpDexLimits](ctx, c, "perpDexLimits", req)
}

// PerpDexStatusRequest is the request for [InfoClient.PerpDexStatus].
type PerpDexStatusRequest struct {
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex"`
}

// PerpDexStatus is the state of a perp dex.
type PerpDexStatus struct {
	TotalNetDeposit Decimal `json:"totalNetDeposit"`
}

// PerpDexStatus returns the state of a perp dex.
func (c *InfoClient) PerpDexStatus(ctx context.Context, req PerpDexStatusRequest) (*PerpDexStatus, error) {
	return infoRequest[*PerpDexStatus](ctx, c, "perpDexStatus", req)
}
