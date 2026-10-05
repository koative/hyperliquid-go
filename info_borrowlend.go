package hyperliquid

import "context"

// BorrowLendReserveStateRequest is the request for
// [InfoClient.BorrowLendReserveState].
type BorrowLendReserveStateRequest struct {
	// Token is the spot token index.
	Token int `json:"token"`
}

// BorrowLendReserveState is the state of a token's borrow/lend reserve.
type BorrowLendReserveState struct {
	BorrowYearlyRate Decimal `json:"borrowYearlyRate"`
	SupplyYearlyRate Decimal `json:"supplyYearlyRate"`
	Balance          Decimal `json:"balance"`
	Utilization      Decimal `json:"utilization"`
	OraclePx         Decimal `json:"oraclePx"`
	// LTV is the loan-to-value ratio of the token as collateral.
	LTV           Decimal `json:"ltv"`
	TotalSupplied Decimal `json:"totalSupplied"`
	TotalBorrowed Decimal `json:"totalBorrowed"`
}

// BorrowLendReserveState returns a token's reserve state. Tokens without a
// reserve fail with an [*APIError].
func (c *InfoClient) BorrowLendReserveState(ctx context.Context, req BorrowLendReserveStateRequest) (*BorrowLendReserveState, error) {
	return infoRequest[*BorrowLendReserveState](ctx, c, "borrowLendReserveState", req)
}

// AllBorrowLendReserveStates returns every reserve state, keyed by spot
// token index.
func (c *InfoClient) AllBorrowLendReserveStates(ctx context.Context) (TupleMap[int, BorrowLendReserveState], error) {
	return infoRequest[TupleMap[int, BorrowLendReserveState]](ctx, c, "allBorrowLendReserveStates", noParams{})
}
