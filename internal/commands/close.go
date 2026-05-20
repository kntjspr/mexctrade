package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/kntjspr/mexctrade/internal/mexc"
	"github.com/kntjspr/mexctrade/internal/output"
)

// CloseArgs is the input bundle for `mexctrade close`.
type CloseArgs struct {
	Symbol string
}

// CloseOutput is the JSON shape for the close result.
type CloseOutput struct {
	DryRun  bool   `json:"dry_run"`
	OrderID string `json:"order_id,omitempty"`
	Symbol  string `json:"symbol"`
	Vol     int64  `json:"vol"`
	Side    string `json:"side"` // "close long" or "close short"
}

// Close runs the `close` command. Always reduce-only.
func Close(ctx context.Context, c *Ctx, a CloseArgs) ExitCode {
	contract, err := c.Resolver.Resolve(a.Symbol)
	if err != nil {
		var unk *mexc.UnknownSymbolError
		if errors.As(err, &unk) {
			emitErr(c, "UNKNOWN_SYMBOL", err.Error(), ExitUnknown)
			return ExitUnknown
		}
		return printErr(c, err)
	}

	positions, err := c.API.GetOpenPositions(ctx, contract.Symbol)
	if err != nil {
		return printErr(c, err)
	}
	var pos *mexc.Position
	for i := range positions {
		if positions[i].Symbol == contract.Symbol && positions[i].HoldVol > 0 {
			pos = &positions[i]
			break
		}
	}
	if pos == nil {
		emitErr(c, "NO_POSITION", "no open position for "+contract.Symbol, ExitRefused)
		return ExitRefused
	}

	apiSide := 2 // close long
	sideLabel := "close long"
	if pos.PositionType == 2 {
		apiSide = 4 // close short
		sideLabel = "close short"
	}

	if c.DryRun {
		out := CloseOutput{DryRun: true, Symbol: contract.Symbol, Vol: pos.HoldVol, Side: sideLabel}
		if c.JSON {
			_ = output.WriteJSON(c.Stdout, out)
		} else {
			fmt.Fprintf(c.Stdout, "[DRY-RUN] would %s %s vol=%d\n", sideLabel, contract.Symbol, pos.HoldVol)
		}
		return ExitOK
	}

	req := mexc.PlaceOrderRequest{
		Symbol:     contract.Symbol,
		Side:       apiSide,
		Type:       5, // market
		OpenType:   1, // cross
		Vol:        int(pos.HoldVol),
		ReduceOnly: true,
	}
	orderID, err := c.API.PlaceOrder(ctx, req)
	if err != nil {
		return printErr(c, err)
	}
	out := CloseOutput{
		DryRun: false, OrderID: orderID, Symbol: contract.Symbol,
		Vol: pos.HoldVol, Side: sideLabel,
	}
	if c.JSON {
		_ = output.WriteJSON(c.Stdout, out)
	} else {
		fmt.Fprintf(c.Stdout, "closed %s vol=%d (order=%s)\n", contract.Symbol, pos.HoldVol, orderID)
	}
	return ExitOK
}
