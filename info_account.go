package hyperliquid

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

// TupleMap is a map that Hyperliquid encodes as a JSON array of [key, value]
// pairs, such as tokenToState: [[0, {...}], [150, {...}]].
type TupleMap[K comparable, V any] map[K]V

// MarshalJSON encodes m as [[key, value], ...] sorted by key, the order the
// exchange requires in signed actions. A nil map encodes as [].
func (m TupleMap[K, V]) MarshalJSON() ([]byte, error) {
	keys := slices.SortedFunc(maps.Keys(m), compareKeys[K])
	pairs := make([][2]any, len(keys))
	for i, k := range keys {
		pairs[i] = [2]any{k, m[k]}
	}
	return json.Marshal(pairs)
}

// compareKeys orders TupleMap keys: numerically for integers, bytewise for
// strings and addresses.
func compareKeys[K comparable](a, b K) int {
	switch a := any(a).(type) {
	case string:
		return cmp.Compare(a, any(b).(string))
	case int:
		return cmp.Compare(a, any(b).(int))
	case int64:
		return cmp.Compare(a, any(b).(int64))
	case Address:
		b := any(b).(Address)
		return bytes.Compare(a[:], b[:])
	}
	// ponytail: other key types fall back to their formatted text.
	return cmp.Compare(fmt.Sprint(a), fmt.Sprint(b))
}

// UnmarshalJSON decodes the wire form [[key, value], ...].
func (m *TupleMap[K, V]) UnmarshalJSON(b []byte) error {
	var pairs []json.RawMessage
	if err := json.Unmarshal(b, &pairs); err != nil || pairs == nil {
		return err
	}
	*m = make(TupleMap[K, V], len(pairs))
	for _, p := range pairs {
		var k K
		var v V
		if err := unmarshalTuple(p, &k, &v); err != nil {
			return err
		}
		(*m)[k] = v
	}
	return nil
}

// TimeValue is a timestamped value of a time series.
type TimeValue struct {
	// Time is in milliseconds since the Unix epoch.
	Time  int64
	Value Decimal
}

// UnmarshalJSON decodes the wire form [time, value].
func (t *TimeValue) UnmarshalJSON(b []byte) error {
	return unmarshalTuple(b, &t.Time, &t.Value)
}

// ClearinghouseStateRequest is the request for [InfoClient.ClearinghouseState].
type ClearinghouseStateRequest struct {
	User Address `json:"user"`
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// ClearinghouseState is a user's perpetuals account on one dex.
type ClearinghouseState struct {
	// MarginSummary covers all positions; CrossMarginSummary only
	// cross-margined ones.
	MarginSummary              MarginSummary   `json:"marginSummary"`
	CrossMarginSummary         MarginSummary   `json:"crossMarginSummary"`
	CrossMaintenanceMarginUsed Decimal         `json:"crossMaintenanceMarginUsed"`
	Withdrawable               Decimal         `json:"withdrawable"`
	AssetPositions             []AssetPosition `json:"assetPositions"`
	// Time is in milliseconds since the Unix epoch.
	Time int64 `json:"time"`
}

// MarginSummary aggregates account value and margin usage.
type MarginSummary struct {
	AccountValue    Decimal `json:"accountValue"`
	TotalNtlPos     Decimal `json:"totalNtlPos"`
	TotalRawUSD     Decimal `json:"totalRawUsd"`
	TotalMarginUsed Decimal `json:"totalMarginUsed"`
}

// AssetPosition is an open perpetual position.
type AssetPosition struct {
	// Type is always "oneWay".
	Type     string   `json:"type"`
	Position Position `json:"position"`
}

// Position is the state of an open perpetual position.
type Position struct {
	Coin string `json:"coin"`
	// Szi is the signed size: positive for long, negative for short.
	Szi            Decimal  `json:"szi"`
	Leverage       Leverage `json:"leverage"`
	EntryPx        Decimal  `json:"entryPx"`
	PositionValue  Decimal  `json:"positionValue"`
	UnrealizedPnl  Decimal  `json:"unrealizedPnl"`
	ReturnOnEquity Decimal  `json:"returnOnEquity"`
	// LiquidationPx is nil when the position cannot be liquidated.
	LiquidationPx *Decimal   `json:"liquidationPx"`
	MarginUsed    Decimal    `json:"marginUsed"`
	MaxLeverage   int        `json:"maxLeverage"`
	CumFunding    CumFunding `json:"cumFunding"`
}

// Leverage is a position's margin mode and leverage.
type Leverage struct {
	// Type is "cross" or "isolated".
	Type  string `json:"type"`
	Value int    `json:"value"`
	// RawUSD is the isolated margin's raw USD value; empty for cross.
	RawUSD Decimal `json:"rawUsd,omitempty"`
}

// CumFunding is the cumulative funding paid by a position (negative when
// received).
type CumFunding struct {
	AllTime     Decimal `json:"allTime"`
	SinceOpen   Decimal `json:"sinceOpen"`
	SinceChange Decimal `json:"sinceChange"`
}

// ClearinghouseState returns a user's perpetuals account state.
func (c *InfoClient) ClearinghouseState(ctx context.Context, req ClearinghouseStateRequest) (*ClearinghouseState, error) {
	return infoRequest[*ClearinghouseState](ctx, c, "clearinghouseState", req)
}

// SpotClearinghouseStateRequest is the request for
// [InfoClient.SpotClearinghouseState].
type SpotClearinghouseStateRequest struct {
	User Address `json:"user"`
}

// SpotClearinghouseState is a user's spot account.
type SpotClearinghouseState struct {
	PortfolioMarginEnabled bool          `json:"portfolioMarginEnabled,omitempty"`
	Balances               []SpotBalance `json:"balances"`
	// EVMEscrows are balances in transit from HyperEVM.
	EVMEscrows                       []EVMEscrow            `json:"evmEscrows,omitempty"`
	PortfolioMarginRatio             Decimal                `json:"portfolioMarginRatio,omitempty"`
	TokenToPortfolioBorrowRatio      TupleMap[int, Decimal] `json:"tokenToPortfolioBorrowRatio,omitempty"`
	TokenToPortfolioSupplyRatio      TupleMap[int, Decimal] `json:"tokenToPortfolioSupplyRatio,omitempty"`
	TokenToAvailableAfterMaintenance TupleMap[int, Decimal] `json:"tokenToAvailableAfterMaintenance,omitempty"`
}

// SpotBalance is a token or outcome balance. Outcome balances have a Coin of
// the form "+N" or "oN" and no Token; their borrow/lend fields are empty.
type SpotBalance struct {
	Coin string `json:"coin"`
	// Token is the spot token index.
	Token    int     `json:"token"`
	Total    Decimal `json:"total"`
	Hold     Decimal `json:"hold"`
	EntryNtl Decimal `json:"entryNtl"`
	SpotHold Decimal `json:"spotHold,omitempty"`
	LTV      Decimal `json:"ltv,omitempty"`
	Borrowed Decimal `json:"borrowed,omitempty"`
	Supplied Decimal `json:"supplied,omitempty"`
}

// EVMEscrow is a token amount in transit from HyperEVM to HyperCore.
type EVMEscrow struct {
	Coin  string  `json:"coin"`
	Token int     `json:"token"`
	Total Decimal `json:"total"`
}

// SpotClearinghouseState returns a user's spot balances.
func (c *InfoClient) SpotClearinghouseState(ctx context.Context, req SpotClearinghouseStateRequest) (*SpotClearinghouseState, error) {
	return infoRequest[*SpotClearinghouseState](ctx, c, "spotClearinghouseState", req)
}

// ActiveAssetDataRequest is the request for [InfoClient.ActiveAssetData].
type ActiveAssetDataRequest struct {
	Coin string  `json:"coin"`
	User Address `json:"user"`
}

// ActiveAssetData is a user's trading limits for one perpetual.
type ActiveAssetData struct {
	User     Address  `json:"user"`
	Coin     string   `json:"coin"`
	Leverage Leverage `json:"leverage"`
	// MaxTradeSzs and AvailableToTrade are [buy, sell].
	MaxTradeSzs      [2]Decimal `json:"maxTradeSzs"`
	AvailableToTrade [2]Decimal `json:"availableToTrade"`
	MarkPx           Decimal    `json:"markPx"`
}

// ActiveAssetData returns a user's leverage and maximum trade sizes for a
// perpetual.
func (c *InfoClient) ActiveAssetData(ctx context.Context, req ActiveAssetDataRequest) (*ActiveAssetData, error) {
	return infoRequest[*ActiveAssetData](ctx, c, "activeAssetData", req)
}

// UserFundingRequest is the request for [InfoClient.UserFunding].
type UserFundingRequest struct {
	User Address `json:"user"`
	// StartTime and EndTime bound the results, in milliseconds since the
	// Unix epoch. Zero leaves the bound open.
	StartTime int64 `json:"startTime,omitempty"`
	EndTime   int64 `json:"endTime,omitempty"`
}

// FundingUpdate is a funding payment of one position.
type FundingUpdate struct {
	// Time is in milliseconds since the Unix epoch.
	Time  int64        `json:"time"`
	Hash  string       `json:"hash"`
	Delta FundingDelta `json:"delta"`
}

// FundingDelta is the content of a [FundingUpdate].
type FundingDelta struct {
	// Type is always "funding".
	Type string `json:"type"`
	Coin string `json:"coin"`
	// USDC is the amount paid (negative) or received (positive).
	USDC        Decimal `json:"usdc"`
	Szi         Decimal `json:"szi"`
	FundingRate Decimal `json:"fundingRate"`
	// NSamples is the number of premium samples, or nil.
	NSamples *int `json:"nSamples"`
}

// UserFunding returns a user's funding payments.
func (c *InfoClient) UserFunding(ctx context.Context, req UserFundingRequest) ([]FundingUpdate, error) {
	return infoRequest[[]FundingUpdate](ctx, c, "userFunding", req)
}

// UserNonFundingLedgerUpdatesRequest is the request for
// [InfoClient.UserNonFundingLedgerUpdates].
type UserNonFundingLedgerUpdatesRequest struct {
	User Address `json:"user"`
	// StartTime and EndTime bound the results, in milliseconds since the
	// Unix epoch. Zero leaves the bound open.
	StartTime int64 `json:"startTime,omitempty"`
	EndTime   int64 `json:"endTime,omitempty"`
}

// LedgerUpdate is a non-funding change to a user's balances.
type LedgerUpdate struct {
	// Time is in milliseconds since the Unix epoch.
	Time  int64       `json:"time"`
	Hash  string      `json:"hash"`
	Delta LedgerDelta `json:"delta"`
}

// LedgerDelta is the content of a [LedgerUpdate]. Type names the variant and
// selects which other fields are set:
//
//   - "accountClassTransfer": USDC, ToPerp
//   - "deposit": USDC
//   - "internalTransfer": USDC, User, Destination, Fee
//   - "liquidation": LiquidatedNtlPos, AccountValue, LeverageType,
//     LiquidatedPositions
//   - "rewardsClaim": Amount, Token
//   - "spotTransfer": Token, Amount, USDCValue, User, Destination, Fee,
//     NativeTokenFee, Nonce, FeeToken
//   - "subAccountTransfer": USDC, User, Destination
//   - "vaultCreate": Vault, USDC, Fee
//   - "vaultDeposit", "vaultDistribution": Vault, USDC
//   - "vaultWithdraw": Vault, User, RequestedUSD, Commission, ClosingCost,
//     Basis, NetWithdrawnUSD
//   - "withdraw": USDC, Nonce, Fee
//   - "send": User, Destination, SourceDex, DestinationDex, Token, Amount,
//     USDCValue, Fee, NativeTokenFee, Nonce, FeeToken
//   - "deployGasAuction", "spotGenesis": Token, Amount
//   - "cStakingTransfer": Token, Amount, IsDeposit
//   - "borrowLend": Token, Operation, Amount, InterestAmount
//   - "activateDexAbstraction": Dex, Token, Amount
//   - "vaultLeaderCommission": User, USDC
type LedgerDelta struct {
	Type        string  `json:"type"`
	USDC        Decimal `json:"usdc,omitempty"`
	ToPerp      bool    `json:"toPerp,omitempty"`
	User        Address `json:"user,omitzero"`
	Destination Address `json:"destination,omitzero"`
	Fee         Decimal `json:"fee,omitempty"`
	// SourceDex and DestinationDex are "" for the main perp dex and "spot"
	// for the spot account.
	SourceDex           string               `json:"sourceDex,omitempty"`
	DestinationDex      string               `json:"destinationDex,omitempty"`
	Dex                 string               `json:"dex,omitempty"`
	Token               string               `json:"token,omitempty"`
	Amount              Decimal              `json:"amount,omitempty"`
	USDCValue           Decimal              `json:"usdcValue,omitempty"`
	NativeTokenFee      Decimal              `json:"nativeTokenFee,omitempty"`
	Nonce               int64                `json:"nonce,omitempty"`
	FeeToken            string               `json:"feeToken,omitempty"`
	LiquidatedNtlPos    Decimal              `json:"liquidatedNtlPos,omitempty"`
	AccountValue        Decimal              `json:"accountValue,omitempty"`
	LeverageType        string               `json:"leverageType,omitempty"` // "Cross" or "Isolated"
	LiquidatedPositions []LiquidatedPosition `json:"liquidatedPositions,omitempty"`
	Vault               Address              `json:"vault,omitzero"`
	RequestedUSD        Decimal              `json:"requestedUsd,omitempty"`
	Commission          Decimal              `json:"commission,omitempty"`
	ClosingCost         Decimal              `json:"closingCost,omitempty"`
	Basis               Decimal              `json:"basis,omitempty"`
	NetWithdrawnUSD     Decimal              `json:"netWithdrawnUsd,omitempty"`
	IsDeposit           bool                 `json:"isDeposit,omitempty"`
	// Operation is "supply", "withdraw", "repay" or "borrow".
	Operation      string  `json:"operation,omitempty"`
	InterestAmount Decimal `json:"interestAmount,omitempty"`
}

// LiquidatedPosition is a position closed by a liquidation.
type LiquidatedPosition struct {
	Coin string  `json:"coin"`
	Szi  Decimal `json:"szi"`
}

// UserNonFundingLedgerUpdates returns a user's deposits, withdrawals,
// transfers, liquidations and other non-funding balance changes.
func (c *InfoClient) UserNonFundingLedgerUpdates(ctx context.Context, req UserNonFundingLedgerUpdatesRequest) ([]LedgerUpdate, error) {
	return infoRequest[[]LedgerUpdate](ctx, c, "userNonFundingLedgerUpdates", req)
}

// PortfolioRequest is the request for [InfoClient.Portfolio].
type PortfolioRequest struct {
	User Address `json:"user"`
}

// Portfolio is an account's history keyed by period: "day", "week", "month",
// "allTime", and their perpetuals-only counterparts "perpDay", "perpWeek",
// "perpMonth", "perpAllTime".
type Portfolio = TupleMap[string, PortfolioHistory]

// PortfolioHistory is an account's value and PnL history over one period.
type PortfolioHistory struct {
	AccountValueHistory []TimeValue `json:"accountValueHistory"`
	PnlHistory          []TimeValue `json:"pnlHistory"`
	// Vlm is the traded volume over the period.
	Vlm Decimal `json:"vlm"`
}

// Portfolio returns a user's account value and PnL history.
func (c *InfoClient) Portfolio(ctx context.Context, req PortfolioRequest) (Portfolio, error) {
	return infoRequest[Portfolio](ctx, c, "portfolio", req)
}

// BorrowLendUserStateRequest is the request for
// [InfoClient.BorrowLendUserState].
type BorrowLendUserStateRequest struct {
	User Address `json:"user"`
}

// BorrowLendUserState is a user's borrow/lend positions.
type BorrowLendUserState struct {
	// TokenToState maps spot token indices to positions.
	TokenToState TupleMap[int, BorrowLendTokenState] `json:"tokenToState"`
	// Health is "healthy", "atRisk", "marketLiquidatable" or
	// "backstopLiquidatable".
	Health       string   `json:"health"`
	HealthFactor *Decimal `json:"healthFactor"`
}

// BorrowLendTokenState is a user's borrow and supply of one token.
type BorrowLendTokenState struct {
	Borrow BorrowLendBalance `json:"borrow"`
	Supply BorrowLendBalance `json:"supply"`
}

// BorrowLendBalance is a borrowed or supplied amount.
type BorrowLendBalance struct {
	// Basis is the principal; Value includes accrued interest.
	Basis Decimal `json:"basis"`
	Value Decimal `json:"value"`
}

// BorrowLendUserState returns a user's borrow/lend positions and health.
func (c *InfoClient) BorrowLendUserState(ctx context.Context, req BorrowLendUserStateRequest) (*BorrowLendUserState, error) {
	return infoRequest[*BorrowLendUserState](ctx, c, "borrowLendUserState", req)
}

// UserBorrowLendInterestRequest is the request for
// [InfoClient.UserBorrowLendInterest].
type UserBorrowLendInterestRequest struct {
	User Address `json:"user"`
	// StartTime and EndTime bound the results, in milliseconds since the
	// Unix epoch. A zero EndTime means now.
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime,omitempty"`
}

// BorrowLendInterest is the interest accrued on one token over one period.
type BorrowLendInterest struct {
	// Time is in milliseconds since the Unix epoch.
	Time   int64   `json:"time"`
	Token  string  `json:"token"`
	Borrow Decimal `json:"borrow"`
	Supply Decimal `json:"supply"`
	// NSamples is the number of interest samples aggregated in the entry.
	NSamples int `json:"nSamples"`
}

// UserBorrowLendInterest returns the interest a user paid and earned.
func (c *InfoClient) UserBorrowLendInterest(ctx context.Context, req UserBorrowLendInterestRequest) ([]BorrowLendInterest, error) {
	return infoRequest[[]BorrowLendInterest](ctx, c, "userBorrowLendInterest", req)
}

// WebData2Request is the request for [InfoClient.WebData2].
type WebData2Request struct {
	User Address `json:"user"`
}

// WebData2 is the aggregate account and market snapshot the Hyperliquid web
// app renders.
type WebData2 struct {
	ClearinghouseState ClearinghouseState  `json:"clearinghouseState"`
	LeadingVaults      []LeadingVault      `json:"leadingVaults"`
	TotalVaultEquity   Decimal             `json:"totalVaultEquity"`
	OpenOrders         []FrontendOpenOrder `json:"openOrders"`
	AgentAddress       *Address            `json:"agentAddress"`
	// AgentValidUntil is in milliseconds since the Unix epoch.
	AgentValidUntil *int64         `json:"agentValidUntil"`
	CumLedger       Decimal        `json:"cumLedger"`
	Meta            Meta           `json:"meta"`
	AssetCtxs       []PerpAssetCtx `json:"assetCtxs"`
	// ServerTime is in milliseconds since the Unix epoch.
	ServerTime int64   `json:"serverTime"`
	IsVault    bool    `json:"isVault"`
	User       Address `json:"user"`
	// TwapStates maps TWAP IDs to running TWAPs.
	TwapStates             TupleMap[int64, TwapState] `json:"twapStates"`
	SpotState              *SpotClearinghouseState    `json:"spotState,omitempty"`
	SpotAssetCtxs          []SpotAssetCtx             `json:"spotAssetCtxs"`
	OptOutOfSpotDusting    bool                       `json:"optOutOfSpotDusting,omitempty"`
	PerpsAtOpenInterestCap []string                   `json:"perpsAtOpenInterestCap,omitempty"`
}

// WebData2 returns a user's account together with main-dex market data.
func (c *InfoClient) WebData2(ctx context.Context, req WebData2Request) (*WebData2, error) {
	return infoRequest[*WebData2](ctx, c, "webData2", req)
}
