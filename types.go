package hyperliquid

// Side is the side of an order, fill or trade.
type Side string

// Sides.
const (
	SideBid Side = "B" // buy
	SideAsk Side = "A" // sell
)
