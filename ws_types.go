package hyperliquid

import (
	"compress/flate"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// AllMidsEvent is a message of [WebSocketClient.AllMids].
type AllMidsEvent struct {
	// Mids maps every coin to its mid price.
	Mids map[string]Decimal `json:"mids"`
	// Dex is the perp dex; empty for the main dex.
	Dex string `json:"dex,omitempty"`
}

// AssetCtxsEvent is a message of [WebSocketClient.AssetCtxs].
type AssetCtxsEvent struct {
	Dex string `json:"dex"`
	// Ctxs is ordered like the dex's [Meta.Universe].
	Ctxs []PerpAssetCtx `json:"ctxs"`
}

// AllDexsAssetCtxsEvent is a message of [WebSocketClient.AllDexsAssetCtxs].
type AllDexsAssetCtxsEvent struct {
	// Ctxs maps each perp dex ("" for the main dex) to its asset contexts.
	Ctxs TupleMap[string, []PerpAssetCtx] `json:"ctxs"`
}

// FastAssetCtxs is a message of [WebSocketClient.FastAssetCtxs]: price
// updates keyed by coin. The first message holds every coin, later ones
// only the coins that changed.
type FastAssetCtxs map[string]FastAssetCtx

// FastAssetCtx holds the prices of one coin in [FastAssetCtxs].
type FastAssetCtx struct {
	// MarkPx is nil when unchanged.
	MarkPx *Decimal `json:"markPx,omitempty"`
	// MidPx is nil when unchanged or when the coin has no mid price.
	MidPx *Decimal `json:"midPx,omitempty"`
}

// UnmarshalJSON decodes the wire form: a base64 string of raw-DEFLATE
// (RFC 1951) compressed JSON.
func (m *FastAssetCtxs) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	r := flate.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(s)))
	defer r.Close()
	return json.NewDecoder(r).Decode((*map[string]FastAssetCtx)(m))
}

// ActiveAssetCtxEvent is a message of [WebSocketClient.ActiveAssetCtx].
type ActiveAssetCtxEvent struct {
	Coin string       `json:"coin"`
	Ctx  PerpAssetCtx `json:"ctx"`
}

// ActiveSpotAssetCtxEvent is a message of
// [WebSocketClient.ActiveSpotAssetCtx].
type ActiveSpotAssetCtxEvent struct {
	Coin string       `json:"coin"`
	Ctx  SpotAssetCtx `json:"ctx"`
}

// BboEvent is a message of [WebSocketClient.Bbo].
type BboEvent struct {
	Coin string `json:"coin"`
	// Time is in milliseconds since the Unix epoch.
	Time int64 `json:"time"`
	// Bbo holds the best bid and the best ask; either is nil when that side
	// is empty.
	Bbo [2]*L2Level `json:"bbo"`
}

// ClearinghouseStateEvent is a message of
// [WebSocketClient.ClearinghouseState].
type ClearinghouseStateEvent struct {
	Dex                string             `json:"dex"`
	User               Address            `json:"user"`
	ClearinghouseState ClearinghouseState `json:"clearinghouseState"`
}

// AllDexsClearinghouseStateEvent is a message of
// [WebSocketClient.AllDexsClearinghouseState].
type AllDexsClearinghouseStateEvent struct {
	User Address `json:"user"`
	// ClearinghouseStates maps each perp dex ("" for the main dex) to the
	// user's state on it.
	ClearinghouseStates TupleMap[string, ClearinghouseState] `json:"clearinghouseStates"`
}

// SpotStateEvent is a message of [WebSocketClient.SpotState].
type SpotStateEvent struct {
	User      Address                `json:"user"`
	SpotState SpotClearinghouseState `json:"spotState"`
}

// OpenOrdersEvent is a message of [WebSocketClient.OpenOrders].
type OpenOrdersEvent struct {
	Dex    string              `json:"dex"`
	User   Address             `json:"user"`
	Orders []FrontendOpenOrder `json:"orders"`
}

// TwapStatesEvent is a message of [WebSocketClient.TwapStates].
type TwapStatesEvent struct {
	Dex  string  `json:"dex"`
	User Address `json:"user"`
	// States maps TWAP IDs to the running TWAP orders.
	States TupleMap[int64, TwapState] `json:"states"`
}

// OrderUpdate is a message item of [WebSocketClient.OrderUpdates].
type OrderUpdate struct {
	Order  OpenOrder             `json:"order"`
	Status OrderProcessingStatus `json:"status"`
	// StatusTimestamp is in milliseconds since the Unix epoch.
	StatusTimestamp int64 `json:"statusTimestamp"`
}

// UserEvent is a message of [WebSocketClient.UserEvents]. Exactly one
// field is set.
type UserEvent struct {
	Fills          []UserFill         `json:"fills,omitempty"`
	Funding        *FundingPayment    `json:"funding,omitempty"`
	Liquidation    *LiquidationEvent  `json:"liquidation,omitempty"`
	NonUserCancel  []NonUserCancel    `json:"nonUserCancel,omitempty"`
	TwapHistory    []TwapHistoryEntry `json:"twapHistory,omitempty"`
	TwapSliceFills []TwapSliceFill    `json:"twapSliceFills,omitempty"`
}

// FundingPayment is a funding payment of one position.
type FundingPayment struct {
	// Time is in milliseconds since the Unix epoch.
	Time int64  `json:"time"`
	Coin string `json:"coin"`
	// USDC is the amount paid (negative) or received (positive).
	USDC        Decimal `json:"usdc"`
	Szi         Decimal `json:"szi"`
	FundingRate Decimal `json:"fundingRate"`
	// NSamples is the number of premium samples, or nil.
	NSamples *int `json:"nSamples"`
}

// LiquidationEvent is a liquidation that involves the user.
type LiquidationEvent struct {
	// Lid is the liquidation ID.
	Lid                    int64   `json:"lid"`
	Liquidator             Address `json:"liquidator"`
	LiquidatedUser         Address `json:"liquidated_user"`
	LiquidatedNtlPos       Decimal `json:"liquidated_ntl_pos"`
	LiquidatedAccountValue Decimal `json:"liquidated_account_value"`
}

// NonUserCancel is an order canceled by the exchange rather than the user.
type NonUserCancel struct {
	Coin string `json:"coin"`
	Oid  int64  `json:"oid"`
}

// UserFillsEvent is a message of [WebSocketClient.UserFills].
type UserFillsEvent struct {
	User  Address    `json:"user"`
	Fills []UserFill `json:"fills"`
	// IsSnapshot is set on the first message, which holds recent history.
	IsSnapshot bool `json:"isSnapshot,omitempty"`
}

// UserFundingsEvent is a message of [WebSocketClient.UserFundings].
type UserFundingsEvent struct {
	User     Address          `json:"user"`
	Fundings []FundingPayment `json:"fundings"`
	// IsSnapshot is set on the first message, which holds recent history.
	IsSnapshot bool `json:"isSnapshot,omitempty"`
}

// UserNonFundingLedgerUpdatesEvent is a message of
// [WebSocketClient.UserNonFundingLedgerUpdates].
type UserNonFundingLedgerUpdatesEvent struct {
	User                    Address        `json:"user"`
	NonFundingLedgerUpdates []LedgerUpdate `json:"nonFundingLedgerUpdates"`
	// IsSnapshot is set on the first message, which holds recent history.
	IsSnapshot bool `json:"isSnapshot,omitempty"`
}

// UserHistoricalOrdersEvent is a message of
// [WebSocketClient.UserHistoricalOrders].
type UserHistoricalOrdersEvent struct {
	User         Address           `json:"user"`
	OrderHistory []OrderWithStatus `json:"orderHistory"`
	// IsSnapshot is set on the first message, which holds recent history.
	IsSnapshot bool `json:"isSnapshot,omitempty"`
}

// UserTwapHistoryEvent is a message of [WebSocketClient.UserTwapHistory].
type UserTwapHistoryEvent struct {
	User    Address            `json:"user"`
	History []TwapHistoryEntry `json:"history"`
	// IsSnapshot is set on the first message, which holds recent history.
	IsSnapshot bool `json:"isSnapshot,omitempty"`
}

// UserTwapSliceFillsEvent is a message of
// [WebSocketClient.UserTwapSliceFills].
type UserTwapSliceFillsEvent struct {
	User           Address         `json:"user"`
	TwapSliceFills []TwapSliceFill `json:"twapSliceFills"`
	// IsSnapshot is set on the first message, which holds recent history.
	IsSnapshot bool `json:"isSnapshot,omitempty"`
}

// NotificationEvent is a message of [WebSocketClient.Notification].
type NotificationEvent struct {
	Notification string `json:"notification"`
}

// WebData3 is a message of [WebSocketClient.WebData3]: account data shown
// by the Hyperliquid web app.
type WebData3 struct {
	UserState WebData3UserState `json:"userState"`
	// PerpDexStates holds one entry per perp dex, main dex first.
	PerpDexStates []WebData3PerpDexState `json:"perpDexStates"`
}

// WebData3UserState is the account part of [WebData3].
type WebData3UserState struct {
	// AgentAddress is the web app's API wallet, or nil.
	AgentAddress *Address `json:"agentAddress"`
	// AgentValidUntil is in milliseconds since the Unix epoch, or nil.
	AgentValidUntil *int64 `json:"agentValidUntil"`
	// CumLedger is the cumulative net deposits.
	CumLedger Decimal `json:"cumLedger"`
	// ServerTime is in milliseconds since the Unix epoch.
	ServerTime            int64       `json:"serverTime"`
	IsVault               bool        `json:"isVault"`
	User                  Address     `json:"user"`
	OptOutOfSpotDusting   bool        `json:"optOutOfSpotDusting,omitempty"`
	DexAbstractionEnabled bool        `json:"dexAbstractionEnabled,omitempty"`
	Abstraction           Abstraction `json:"abstraction,omitempty"`
}

// WebData3PerpDexState is the per-dex part of [WebData3].
type WebData3PerpDexState struct {
	TotalVaultEquity       Decimal        `json:"totalVaultEquity"`
	PerpsAtOpenInterestCap []string       `json:"perpsAtOpenInterestCap,omitempty"`
	LeadingVaults          []LeadingVault `json:"leadingVaults,omitempty"`
}

// OutcomeMetaUpdatesEvent is a message of
// [WebSocketClient.OutcomeMetaUpdates].
type OutcomeMetaUpdatesEvent struct {
	Updates []OutcomeMetaUpdate `json:"updates"`
}

// OutcomeMetaUpdate is a change to outcome market metadata. Exactly one
// field is set.
type OutcomeMetaUpdate struct {
	OutcomeCreated  *OutcomeSpec     `json:"outcomeCreated,omitempty"`
	OutcomeSettled  *OutcomeSettled  `json:"outcomeSettled,omitempty"`
	QuestionUpdated *OutcomeQuestion `json:"questionUpdated,omitempty"`
	QuestionSettled *QuestionSettled `json:"questionSettled,omitempty"`
}

// OutcomeSettled reports that an outcome settled.
type OutcomeSettled struct {
	Outcome int `json:"outcome"`
}

// QuestionSettled reports that a question settled.
type QuestionSettled struct {
	Question int `json:"question"`
}
