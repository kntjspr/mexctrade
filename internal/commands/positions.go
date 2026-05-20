package commands

import (
	"context"

	"github.com/kntjspr/mexctrade/internal/output"
)

// PositionOutput is the JSON shape for `mexctrade positions --json`.
type PositionOutput struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Contracts     int64   `json:"contracts"`
	EntryPrice    float64 `json:"entry_price"`
	MarkPrice     float64 `json:"mark_price"`
	UnrealizedPNL float64 `json:"unrealized_pnl"`
	Leverage      int     `json:"leverage"`
}

// Positions runs the `positions` command.
func Positions(ctx context.Context, c *Ctx) ExitCode {
	raw, err := c.API.GetOpenPositions(ctx, "")
	if err != nil {
		return printErr(c, err)
	}
	out := make([]PositionOutput, 0, len(raw))
	for _, p := range raw {
		out = append(out, PositionOutput{
			Symbol:        p.Symbol,
			Side:          sideName(p.PositionType),
			Contracts:     p.HoldVol,
			EntryPrice:    p.HoldAvgPrice,
			MarkPrice:     p.MarkPrice,
			UnrealizedPNL: p.UnrealizedPNL,
			Leverage:      p.Leverage,
		})
	}
	if c.JSON {
		_ = output.WriteJSON(c.Stdout, out)
	} else {
		for _, p := range out {
			output.PrintLine(c.Stdout, p.Symbol, p.Side, p.Contracts, p.EntryPrice, p.MarkPrice, p.UnrealizedPNL, p.Leverage)
		}
	}
	return ExitOK
}

func sideName(t int) string {
	if t == 1 {
		return "long"
	}
	return "short"
}
