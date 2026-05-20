package commands

import (
	"context"

	"github.com/kntjspr/mexctrade/internal/output"
)

// OrderOutput is the JSON shape for `mexctrade orders --json`.
type OrderOutput struct {
	OrderID string  `json:"order_id"`
	Symbol  string  `json:"symbol"`
	Side    string  `json:"side"`
	Type    string  `json:"type"`
	Price   float64 `json:"price"`
	Vol     int64   `json:"vol"`
	State   int     `json:"state"`
}

// Orders runs the `orders` command.
func Orders(ctx context.Context, c *Ctx) ExitCode {
	raw, err := c.API.GetOpenOrders(ctx, "")
	if err != nil {
		return printErr(c, err)
	}
	out := make([]OrderOutput, 0, len(raw))
	for _, o := range raw {
		out = append(out, OrderOutput{
			OrderID: o.OrderID,
			Symbol:  o.Symbol,
			Side:    orderSideName(o.Side),
			Type:    orderTypeName(o.Type),
			Price:   o.Price,
			Vol:     o.Vol,
			State:   o.State,
		})
	}
	if c.JSON {
		_ = output.WriteJSON(c.Stdout, out)
	} else {
		for _, o := range out {
			output.PrintLine(c.Stdout, o.OrderID, o.Symbol, o.Side, o.Type, o.Price, o.Vol)
		}
	}
	return ExitOK
}

func orderSideName(s int) string {
	switch s {
	case 1:
		return "open long"
	case 2:
		return "close short"
	case 3:
		return "open short"
	case 4:
		return "close long"
	default:
		return "?"
	}
}

func orderTypeName(t int) string {
	if t == 5 {
		return "market"
	}
	return "limit"
}
