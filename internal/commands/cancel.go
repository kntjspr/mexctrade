package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/kntjspr/mexctrade/internal/mexc"
	"github.com/kntjspr/mexctrade/internal/output"
)

// CancelArgs is the input bundle for `mexctrade cancel`.
type CancelArgs struct {
	Symbol  string
	OrderID string // optional; when set, cancels only that order
}

// CancelOutput is the JSON shape for the cancel result.
type CancelOutput struct {
	DryRun  bool   `json:"dry_run"`
	Symbol  string `json:"symbol"`
	OrderID string `json:"order_id,omitempty"`
}

// Cancel runs the `cancel` command.
func Cancel(ctx context.Context, c *Ctx, a CancelArgs) ExitCode {
	contract, err := c.Resolver.Resolve(a.Symbol)
	if err != nil {
		var unk *mexc.UnknownSymbolError
		if errors.As(err, &unk) {
			emitErr(c, "UNKNOWN_SYMBOL", err.Error(), ExitUnknown)
			return ExitUnknown
		}
		return printErr(c, err)
	}

	if c.DryRun {
		out := CancelOutput{DryRun: true, Symbol: contract.Symbol, OrderID: a.OrderID}
		if c.JSON {
			_ = output.WriteJSON(c.Stdout, out)
		} else {
			fmt.Fprintf(c.Stdout, "[DRY-RUN] would cancel %s\n", contract.Symbol)
		}
		return ExitOK
	}

	if a.OrderID != "" {
		if err := c.API.CancelOrders(ctx, []string{a.OrderID}); err != nil {
			return printErr(c, err)
		}
	} else {
		if err := c.API.CancelAll(ctx, contract.Symbol); err != nil {
			return printErr(c, err)
		}
	}

	out := CancelOutput{DryRun: false, Symbol: contract.Symbol, OrderID: a.OrderID}
	if c.JSON {
		_ = output.WriteJSON(c.Stdout, out)
	} else {
		fmt.Fprintf(c.Stdout, "cancelled %s\n", contract.Symbol)
	}
	return ExitOK
}
