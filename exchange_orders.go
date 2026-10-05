package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
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

// OrderStatus is the outcome of one order. Exactly one of Resting, Filled,
// Error and Status is set.
type OrderStatus struct {
	Resting *RestingOrder `json:"resting,omitempty"`
	Filled  *FilledOrder  `json:"filled,omitempty"`
	Error   string        `json:"error,omitempty"`
	// Status holds plain statuses such as "waitingForFill" and
	// "waitingForTrigger".
	Status string `json:"-"`
}

// UnmarshalJSON accepts both object statuses and plain string statuses.
func (s *OrderStatus) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &s.Status)
	}
	type plain OrderStatus
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
func (c *ExchangeClient) Order(ctx context.Context, a OrderAction) ([]OrderStatus, error) {
	var data struct {
		Statuses []OrderStatus `json:"statuses"`
	}
	if err := c.do(ctx, a, &data); err != nil {
		return nil, err
	}
	return data.Statuses, orderStatusErrors(data.Statuses)
}

func orderStatusErrors(statuses []OrderStatus) error {
	msgs := make([]string, len(statuses))
	for i, s := range statuses {
		msgs[i] = s.Error
	}
	return statusErrors(msgs)
}

// CancelAction cancels orders by order ID.
type CancelAction struct {
	Cancels []Cancel `json:"cancels"`
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
		Statuses []OrderStatus `json:"statuses"`
	}
	if err := c.do(ctx, a, &data); err != nil {
		return err
	}
	return orderStatusErrors(data.Statuses)
}
