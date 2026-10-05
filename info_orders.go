package hyperliquid

import (
	"context"
	"fmt"
)

// OpenOrder is a resting order.
type OpenOrder struct {
	Coin    string  `json:"coin"`
	Side    Side    `json:"side"`
	LimitPx Decimal `json:"limitPx"`
	// Sz is the remaining size; OrigSz the size when placed.
	Sz  Decimal `json:"sz"`
	Oid int64   `json:"oid"`
	// Timestamp is the placement time in milliseconds since the Unix epoch.
	Timestamp  int64   `json:"timestamp"`
	OrigSz     Decimal `json:"origSz"`
	Cloid      *Cloid  `json:"cloid,omitempty"`
	ReduceOnly bool    `json:"reduceOnly,omitempty"`
}

// FrontendOpenOrder is an order with the trigger and TP/SL details shown by
// the Hyperliquid web app.
type FrontendOpenOrder struct {
	Coin    string  `json:"coin"`
	Side    Side    `json:"side"`
	LimitPx Decimal `json:"limitPx"`
	// Sz is the remaining size; OrigSz the size when placed.
	Sz  Decimal `json:"sz"`
	Oid int64   `json:"oid"`
	// Timestamp is the placement time in milliseconds since the Unix epoch.
	Timestamp        int64   `json:"timestamp"`
	OrigSz           Decimal `json:"origSz"`
	TriggerCondition string  `json:"triggerCondition"`
	IsTrigger        bool    `json:"isTrigger"`
	TriggerPx        Decimal `json:"triggerPx"`
	// Children are the TP/SL orders attached to this order.
	Children       []FrontendOpenOrder `json:"children"`
	IsPositionTpsl bool                `json:"isPositionTpsl"`
	ReduceOnly     bool                `json:"reduceOnly"`
	// OrderType is "Market", "Limit", "Stop Market", "Stop Limit",
	// "Take Profit Market", "Take Profit Limit", "Twap Slice", "Vault Close"
	// or "Spot Dust Conversion".
	OrderType string `json:"orderType"`
	// Tif is nil for trigger orders; liquidations use "LiquidationMarket".
	Tif   *Tif   `json:"tif"`
	Cloid *Cloid `json:"cloid"`
}

// OrderProcessingStatus is the lifecycle status of an order: "open",
// "filled", "canceled", "triggered", "rejected", or a specific cancel or
// reject reason such as "marginCanceled", "selfTradeCanceled",
// "reduceOnlyCanceled", "scheduledCancel", "tickRejected" or
// "perpMarginRejected".
type OrderProcessingStatus string

// OrderWithStatus is an order with its latest status.
type OrderWithStatus struct {
	Order  FrontendOpenOrder     `json:"order"`
	Status OrderProcessingStatus `json:"status"`
	// StatusTimestamp is in milliseconds since the Unix epoch.
	StatusTimestamp int64 `json:"statusTimestamp"`
}

// OpenOrdersRequest is the request for [InfoClient.OpenOrders].
type OpenOrdersRequest struct {
	User Address `json:"user"`
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// OpenOrders returns a user's resting orders.
func (c *InfoClient) OpenOrders(ctx context.Context, req OpenOrdersRequest) ([]OpenOrder, error) {
	return infoRequest[[]OpenOrder](ctx, c, "openOrders", req)
}

// FrontendOpenOrdersRequest is the request for [InfoClient.FrontendOpenOrders].
type FrontendOpenOrdersRequest struct {
	User Address `json:"user"`
	// Dex is the perp dex name; empty selects the main dex.
	Dex string `json:"dex,omitempty"`
}

// FrontendOpenOrders returns a user's resting orders with trigger and TP/SL
// details.
func (c *InfoClient) FrontendOpenOrders(ctx context.Context, req FrontendOpenOrdersRequest) ([]FrontendOpenOrder, error) {
	return infoRequest[[]FrontendOpenOrder](ctx, c, "frontendOpenOrders", req)
}

// HistoricalOrdersRequest is the request for [InfoClient.HistoricalOrders].
type HistoricalOrdersRequest struct {
	User Address `json:"user"`
}

// HistoricalOrders returns up to 2000 of a user's most recent orders.
func (c *InfoClient) HistoricalOrders(ctx context.Context, req HistoricalOrdersRequest) ([]OrderWithStatus, error) {
	return infoRequest[[]OrderWithStatus](ctx, c, "historicalOrders", req)
}

// OrderStatusRequest is the request for [InfoClient.OrderStatus].
type OrderStatusRequest struct {
	User Address `json:"user"`
	// Oid selects the order by ID or client order ID.
	Oid OrderRef `json:"oid"`
}

// OrderStatus returns an order and its status, or nil if the order is
// unknown.
func (c *InfoClient) OrderStatus(ctx context.Context, req OrderStatusRequest) (*OrderWithStatus, error) {
	resp, err := infoRequest[struct {
		Status string           `json:"status"`
		Order  *OrderWithStatus `json:"order"`
	}](ctx, c, "orderStatus", req)
	if err != nil {
		return nil, err
	}
	if resp.Status != "order" && resp.Status != "unknownOid" {
		return nil, fmt.Errorf("hyperliquid: unexpected orderStatus status %q", resp.Status)
	}
	return resp.Order, nil
}

// UserFill is a fill of one of a user's orders.
type UserFill struct {
	Coin string  `json:"coin"`
	Px   Decimal `json:"px"`
	Sz   Decimal `json:"sz"`
	Side Side    `json:"side"`
	// Time is in milliseconds since the Unix epoch.
	Time          int64   `json:"time"`
	StartPosition Decimal `json:"startPosition"`
	// Dir describes the fill's effect, such as "Open Long" or "Close Short".
	Dir       string  `json:"dir"`
	ClosedPnl Decimal `json:"closedPnl"`
	Hash      string  `json:"hash"`
	Oid       int64   `json:"oid"`
	// Crossed reports whether the order took liquidity.
	Crossed bool `json:"crossed"`
	// Fee is negative for a maker rebate.
	Fee            Decimal          `json:"fee"`
	BuilderFee     Decimal          `json:"builderFee,omitempty"`
	Tid            int64            `json:"tid"`
	FeeToken       string           `json:"feeToken"`
	FeeTrialEscrow Decimal          `json:"feeTrialEscrow,omitempty"`
	TwapID         *int64           `json:"twapId"`
	Cloid          *Cloid           `json:"cloid,omitempty"`
	Liquidation    *FillLiquidation `json:"liquidation,omitempty"`
}

// FillLiquidation describes the liquidation a fill was part of.
type FillLiquidation struct {
	LiquidatedUser Address `json:"liquidatedUser,omitzero"`
	MarkPx         Decimal `json:"markPx"`
	// Method is "market" or "backstop".
	Method string `json:"method"`
}

// UserFillsRequest is the request for [InfoClient.UserFills].
type UserFillsRequest struct {
	User Address `json:"user"`
	// AggregateByTime merges partial fills of the same order at the same
	// time.
	AggregateByTime bool `json:"aggregateByTime,omitempty"`
}

// UserFills returns up to 2000 of a user's most recent fills.
func (c *InfoClient) UserFills(ctx context.Context, req UserFillsRequest) ([]UserFill, error) {
	return infoRequest[[]UserFill](ctx, c, "userFills", req)
}

// UserFillsByTimeRequest is the request for [InfoClient.UserFillsByTime].
type UserFillsByTimeRequest struct {
	User Address `json:"user"`
	// StartTime and EndTime bound the results, in milliseconds since the
	// Unix epoch. A zero EndTime means now.
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime,omitempty"`
	// AggregateByTime merges partial fills of the same order at the same
	// time.
	AggregateByTime bool `json:"aggregateByTime,omitempty"`
	// Reversed returns the newest fills first.
	Reversed bool `json:"reversed,omitempty"`
}

// UserFillsByTime returns up to 2000 of a user's fills in a time range.
func (c *InfoClient) UserFillsByTime(ctx context.Context, req UserFillsByTimeRequest) ([]UserFill, error) {
	return infoRequest[[]UserFill](ctx, c, "userFillsByTime", req)
}

// TwapState is the state of a TWAP order.
type TwapState struct {
	Coin        string   `json:"coin"`
	ExecutedNtl Decimal  `json:"executedNtl"`
	ExecutedSz  Decimal  `json:"executedSz"`
	Minutes     int      `json:"minutes"`
	Randomize   bool     `json:"randomize"`
	ReduceOnly  bool     `json:"reduceOnly"`
	Side        Side     `json:"side"`
	StopPx      *Decimal `json:"stopPx"`
	Sz          Decimal  `json:"sz"`
	// Timestamp is the start time in milliseconds since the Unix epoch.
	Timestamp int64        `json:"timestamp"`
	Trigger   *TwapTrigger `json:"trigger"`
	User      Address      `json:"user"`
}

// TwapTrigger is the price condition that starts a TWAP.
type TwapTrigger struct {
	Px    Decimal `json:"px"`
	Above bool    `json:"above"`
}

// TwapSliceFill is a fill of a TWAP slice.
type TwapSliceFill struct {
	Fill   UserFill `json:"fill"`
	TwapID int64    `json:"twapId"`
}

// UserTwapSliceFillsRequest is the request for [InfoClient.UserTwapSliceFills].
type UserTwapSliceFillsRequest struct {
	User Address `json:"user"`
}

// UserTwapSliceFills returns up to 2000 of a user's most recent TWAP slice
// fills.
func (c *InfoClient) UserTwapSliceFills(ctx context.Context, req UserTwapSliceFillsRequest) ([]TwapSliceFill, error) {
	return infoRequest[[]TwapSliceFill](ctx, c, "userTwapSliceFills", req)
}

// UserTwapSliceFillsByTimeRequest is the request for
// [InfoClient.UserTwapSliceFillsByTime].
type UserTwapSliceFillsByTimeRequest struct {
	User Address `json:"user"`
	// StartTime and EndTime bound the results, in milliseconds since the
	// Unix epoch. A zero EndTime means now.
	StartTime int64 `json:"startTime"`
	EndTime   int64 `json:"endTime,omitempty"`
}

// UserTwapSliceFillsByTime returns a user's TWAP slice fills in a time range.
func (c *InfoClient) UserTwapSliceFillsByTime(ctx context.Context, req UserTwapSliceFillsByTimeRequest) ([]TwapSliceFill, error) {
	return infoRequest[[]TwapSliceFill](ctx, c, "userTwapSliceFillsByTime", req)
}

// TwapHistoryRequest is the request for [InfoClient.TwapHistory].
type TwapHistoryRequest struct {
	User Address `json:"user"`
}

// TwapHistoryEntry is a TWAP status change.
type TwapHistoryEntry struct {
	// Time is in seconds (not milliseconds) since the Unix epoch.
	Time   int64      `json:"time"`
	State  TwapState  `json:"state"`
	Status TwapStatus `json:"status"`
	TwapID int64      `json:"twapId,omitempty"`
}

// TwapStatus is the status of a TWAP.
type TwapStatus struct {
	// Status is "activated", "finished", "terminated", "waitingForTrigger",
	// "stopped" or "error".
	Status string `json:"status"`
	// Description explains an "error" status.
	Description string `json:"description,omitempty"`
}

// TwapHistory returns a user's TWAP history.
func (c *InfoClient) TwapHistory(ctx context.Context, req TwapHistoryRequest) ([]TwapHistoryEntry, error) {
	return infoRequest[[]TwapHistoryEntry](ctx, c, "twapHistory", req)
}
