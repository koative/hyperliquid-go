package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// OrderAction places one or more orders.
type OrderAction struct {
	Orders []Order `json:"orders"`
	// Grouping links the orders; the zero value is [GroupingNA].
	Grouping Grouping `json:"grouping"`
	// Builder optionally pays a builder fee on the orders.
	Builder *Builder `json:"builder,omitempty"`
}

func (OrderAction) actionType() string { return "order" }

// Order is a single order of an [OrderAction].
type Order struct {
	// Asset is the asset ID: the index in [Meta] universe for perps,
	// 10000 + spot pair index for spot, and 100000 + 10000*dexIndex + index
	// for builder-deployed perp dexes.
	Asset int  `json:"a"`
	IsBuy bool `json:"b"`
	// Price is the limit price. Round it with [FormatPrice].
	Price Decimal `json:"p"`
	// Size is in base units. Round it with [FormatSize].
	Size       Decimal   `json:"s"`
	ReduceOnly bool      `json:"r"`
	Type       OrderType `json:"t"`
	// Cloid is an optional client order ID.
	Cloid *Cloid `json:"c,omitempty"`
}

// OrderType is either a limit or a trigger order; set exactly one field.
type OrderType struct {
	Limit   *LimitOrder   `json:"limit,omitempty"`
	Trigger *TriggerOrder `json:"trigger,omitempty"`
}

// LimitOrder is a limit order type.
type LimitOrder struct {
	Tif Tif `json:"tif"`
}

// TriggerOrder is a take-profit or stop-loss order type.
type TriggerOrder struct {
	// IsMarket executes as a market order when triggered; otherwise the
	// order's Price is the limit price.
	IsMarket  bool    `json:"isMarket"`
	TriggerPx Decimal `json:"triggerPx"`
	Tpsl      Tpsl    `json:"tpsl"`
}

// Tif is an order's time in force.
type Tif string

// Time-in-force values.
const (
	// TifGtc rests on the book until filled or canceled.
	TifGtc Tif = "Gtc"
	// TifIoc fills immediately what it can and cancels the rest.
	TifIoc Tif = "Ioc"
	// TifAlo (add liquidity only) is canceled instead of taking liquidity.
	TifAlo Tif = "Alo"
	// TifFrontendMarket is the market order type used by the Hyperliquid UI.
	TifFrontendMarket Tif = "FrontendMarket"
)

// Tpsl is the kind of a trigger order.
type Tpsl string

// Trigger order kinds.
const (
	TpslTakeProfit Tpsl = "tp"
	TpslStopLoss   Tpsl = "sl"
)

// Grouping controls how the orders of an [OrderAction] are linked.
type Grouping struct {
	kind     string
	priority uint64
}

var (
	// GroupingNA places independent orders.
	GroupingNA = Grouping{kind: "na"}
	// GroupingNormalTpsl attaches fixed-size TP/SL orders to the first order.
	GroupingNormalTpsl = Grouping{kind: "normalTpsl"}
	// GroupingPositionTpsl attaches TP/SL orders that track the position size.
	GroupingPositionTpsl = Grouping{kind: "positionTpsl"}
)

// GroupingPriority bids a priority fee of rate/1e8 of notional for faster
// execution. All orders must be IOC, or all must be non-reduce-only ALO.
func GroupingPriority(rate uint64) Grouping { return Grouping{priority: rate} }

// MarshalJSON encodes g as "na", "normalTpsl", "positionTpsl" or {"p":rate}.
func (g Grouping) MarshalJSON() ([]byte, error) {
	switch {
	case g.kind != "":
		return json.Marshal(g.kind)
	case g.priority != 0:
		return fmt.Appendf(nil, `{"p":%d}`, g.priority), nil
	}
	return []byte(`"na"`), nil
}

// Builder is a builder fee attached to orders.
type Builder struct {
	Address Address `json:"b"`
	// Fee is in tenths of a basis point: 10 means 0.01%.
	Fee int `json:"f"`
}

// OrderResult is the outcome of one order. Exactly one of Resting, Filled,
// Error and Status is set.
type OrderResult struct {
	Resting *RestingOrder `json:"resting,omitempty"`
	Filled  *FilledOrder  `json:"filled,omitempty"`
	Error   string        `json:"error,omitempty"`
	// Status holds plain statuses such as "waitingForFill" and
	// "waitingForTrigger".
	Status string `json:"-"`
}

// UnmarshalJSON accepts both object statuses and plain string statuses.
func (s *OrderResult) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &s.Status)
	}
	type plain OrderResult
	return json.Unmarshal(b, (*plain)(s))
}

// RestingOrder is an order resting on the book.
type RestingOrder struct {
	Oid   int64  `json:"oid"`
	Cloid *Cloid `json:"cloid,omitempty"`
}

// FilledOrder is an order that filled immediately.
type FilledOrder struct {
	TotalSz Decimal `json:"totalSz"`
	AvgPx   Decimal `json:"avgPx"`
	Oid     int64   `json:"oid"`
	Cloid   *Cloid  `json:"cloid,omitempty"`
}

// Order places orders and returns one status per order, in request order.
// If some orders are rejected, the statuses are returned together with a
// joined [*StatusError] per rejected order.
func (c *ExchangeClient) Order(ctx context.Context, a OrderAction) ([]OrderResult, error) {
	return c.placeOrders(ctx, a)
}

// placeOrders submits an action whose response is a list of order statuses.
func (c *ExchangeClient) placeOrders(ctx context.Context, a Action) ([]OrderResult, error) {
	var data struct {
		Statuses []OrderResult `json:"statuses"`
	}
	if err := c.do(ctx, a, &data); err != nil {
		return nil, err
	}
	return data.Statuses, orderResultErrors(data.Statuses)
}

func orderResultErrors(statuses []OrderResult) error {
	msgs := make([]string, len(statuses))
	for i, s := range statuses {
		msgs[i] = s.Error
	}
	return statusErrors(msgs)
}

// CancelAction cancels orders by order ID.
type CancelAction struct {
	Cancels []Cancel `json:"cancels"`
	// Fast prioritizes the cancel in the mempool.
	Fast bool `json:"f,omitempty"`
}

func (CancelAction) actionType() string { return "cancel" }

// Cancel identifies an order to cancel.
type Cancel struct {
	Asset int   `json:"a"`
	Oid   int64 `json:"o"`
}

// Cancel cancels orders. If some cancels fail, it returns a joined
// [*StatusError] per failed cancel.
func (c *ExchangeClient) Cancel(ctx context.Context, a CancelAction) error {
	return c.cancel(ctx, a)
}

// cancel submits a cancel-style action whose response is a list of
// "success" or {"error": msg} statuses.
func (c *ExchangeClient) cancel(ctx context.Context, a Action) error {
	var data struct {
		Statuses []OrderResult `json:"statuses"`
	}
	if err := c.do(ctx, a, &data); err != nil {
		return err
	}
	return orderResultErrors(data.Statuses)
}

// CancelByCloidAction cancels orders by client order ID.
type CancelByCloidAction struct {
	Cancels []CancelByCloid `json:"cancels"`
	// Fast prioritizes the cancel in the mempool.
	Fast bool `json:"f,omitempty"`
}

func (CancelByCloidAction) actionType() string { return "cancelByCloid" }

// CancelByCloid identifies an order to cancel by its client order ID.
type CancelByCloid struct {
	Asset int   `json:"asset"`
	Cloid Cloid `json:"cloid"`
}

// CancelByCloid cancels orders by client order ID. If some cancels fail, it
// returns a joined [*StatusError] per failed cancel.
func (c *ExchangeClient) CancelByCloid(ctx context.Context, a CancelByCloidAction) error {
	return c.cancel(ctx, a)
}

// OrderRef identifies an order by exchange order ID or, when Cloid is set,
// by client order ID.
type OrderRef struct {
	Oid   int64
	Cloid *Cloid
}

// MarshalJSON encodes r as the order ID number or the client order ID string.
func (r OrderRef) MarshalJSON() ([]byte, error) {
	if r.Cloid != nil {
		return json.Marshal(r.Cloid)
	}
	return strconv.AppendInt(nil, r.Oid, 10), nil
}

// ModifyAction replaces a resting order with a new one.
type ModifyAction struct {
	Oid   OrderRef `json:"oid"`
	Order Order    `json:"order"`
	// AlwaysPlace places the new order even if canceling the old one fails.
	// Otherwise the new order must be a non-trigger ALO order or a
	// non-marketable GTC order.
	AlwaysPlace bool `json:"a,omitempty"`
}

func (ModifyAction) actionType() string { return "modify" }

// Modify replaces a resting order.
func (c *ExchangeClient) Modify(ctx context.Context, a ModifyAction) error {
	return c.do(ctx, a, nil)
}

// BatchModifyAction replaces several resting orders.
type BatchModifyAction struct {
	Modifies []Modify `json:"modifies"`
	// AlwaysPlace places the new orders even if canceling the old ones fails.
	AlwaysPlace bool `json:"a,omitempty"`
}

func (BatchModifyAction) actionType() string { return "batchModify" }

// Modify is one replacement of a [BatchModifyAction].
type Modify struct {
	Oid   OrderRef `json:"oid"`
	Order Order    `json:"order"`
}

// BatchModify replaces orders and returns one status per new order, like
// [ExchangeClient.Order].
func (c *ExchangeClient) BatchModify(ctx context.Context, a BatchModifyAction) ([]OrderResult, error) {
	return c.placeOrders(ctx, a)
}

// ScheduleCancelAction sets or clears a dead man's switch that cancels all
// open orders at a given time. At most 10 triggers are allowed per day.
type ScheduleCancelAction struct {
	// Time is when to cancel, in Unix milliseconds, at least 5 seconds in
	// the future. Zero clears the scheduled cancel.
	Time int64 `json:"time,omitempty"`
}

func (ScheduleCancelAction) actionType() string { return "scheduleCancel" }

// ScheduleCancel sets or clears the scheduled cancel of all open orders.
func (c *ExchangeClient) ScheduleCancel(ctx context.Context, a ScheduleCancelAction) error {
	return c.do(ctx, a, nil)
}

// TrailingStopAction places a trailing stop order, which follows the price
// by Retracement once ActivationPx is reached.
type TrailingStopAction struct {
	Asset      int     `json:"asset"`
	IsBuy      bool    `json:"isBuy"`
	Size       Decimal `json:"sz"`
	ReduceOnly bool    `json:"reduceOnly"`
	// Retracement is the distance the stop trails the best price by.
	Retracement Retracement `json:"retracement"`
	// ActivationPx, if set, starts trailing only once the price reaches it.
	ActivationPx *Decimal `json:"activationPx"`
}

func (TrailingStopAction) actionType() string { return "trailingStop" }

// Retracement is the trailing distance of a [TrailingStopAction]. Set
// exactly one field.
type Retracement struct {
	// Pct is a percentage of the price, such as "1.5" for 1.5%.
	Pct *Decimal
	// Px is an absolute price distance.
	Px *Decimal
}

// MarshalJSON encodes r as {"pct":"<Pct>%"} or {"px":"<Px>"}.
func (r Retracement) MarshalJSON() ([]byte, error) {
	if (r.Pct == nil) == (r.Px == nil) {
		return nil, errors.New("hyperliquid: set exactly one of Retracement.Pct and Retracement.Px")
	}
	if r.Px != nil {
		return json.Marshal(struct {
			Px Decimal `json:"px"`
		}{*r.Px})
	}
	pct, err := percent(*r.Pct)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Pct string `json:"pct"`
	}{pct})
}

// TrailingStop places a trailing stop order and returns its order ID.
func (c *ExchangeClient) TrailingStop(ctx context.Context, a TrailingStopAction) (int64, error) {
	var data struct {
		Oid int64 `json:"oid"`
	}
	err := c.do(ctx, a, &data)
	return data.Oid, err
}
