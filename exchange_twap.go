package hyperliquid

import "context"

// TwapOrderAction places a TWAP order, which trades Size over Minutes in
// slices.
type TwapOrderAction struct {
	Twap Twap `json:"twap"`
	// Details optionally adds an activation trigger and a stop price.
	Details *TwapDetails `json:"details,omitempty"`
}

func (TwapOrderAction) actionType() string { return "twapOrder" }

// Twap holds the parameters of a [TwapOrderAction].
type Twap struct {
	Asset int  `json:"a"`
	IsBuy bool `json:"b"`
	// Size is the total size in base units.
	Size       Decimal `json:"s"`
	ReduceOnly bool    `json:"r"`
	// Minutes is the duration, from 5 to 1440.
	Minutes int `json:"m"`
	// Randomize randomizes the timing of the slices.
	Randomize bool `json:"t"`
}

// TwapDetails holds the optional conditions of a [TwapOrderAction].
type TwapDetails struct {
	// Trigger, if set, delays the TWAP until the mark price crosses a price.
	Trigger *TwapActivation `json:"t"`
	// StopPx, if set, terminates the TWAP when reached.
	StopPx *Decimal `json:"s"`
}

// TwapActivation is the activation condition of a TWAP order.
type TwapActivation struct {
	Price Decimal `json:"p"`
	// Above activates when the mark price is above Price; otherwise below.
	Above bool `json:"a"`
}

// TwapOrder places a TWAP order and returns its TWAP ID. A rejected order
// returns a [*StatusError].
func (c *ExchangeClient) TwapOrder(ctx context.Context, a TwapOrderAction) (int64, error) {
	var data struct {
		Status struct {
			Running *struct {
				TwapID int64 `json:"twapId"`
			} `json:"running"`
			Error string `json:"error"`
		} `json:"status"`
	}
	if err := c.do(ctx, a, &data); err != nil {
		return 0, err
	}
	if data.Status.Running == nil {
		return 0, &StatusError{Message: data.Status.Error}
	}
	return data.Status.Running.TwapID, nil
}

// TwapCancelAction cancels a running TWAP order.
type TwapCancelAction struct {
	Asset  int   `json:"a"`
	TwapID int64 `json:"t"`
}

func (TwapCancelAction) actionType() string { return "twapCancel" }

// TwapCancel cancels a TWAP order. A failed cancel returns a [*StatusError].
func (c *ExchangeClient) TwapCancel(ctx context.Context, a TwapCancelAction) error {
	var data struct {
		Status OrderStatus `json:"status"`
	}
	if err := c.do(ctx, a, &data); err != nil {
		return err
	}
	return statusErrors([]string{data.Status.Error})
}
