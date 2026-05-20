package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/kntjspr/mexctrade/internal/mexc"
	"github.com/kntjspr/mexctrade/internal/output"
)

// PortfolioOutput is the JSON shape for `mexctrade portfolio --json`.
type PortfolioOutput struct {
	BalanceUSDT         float64 `json:"balance_usdt"`
	AvailableMarginUSDT float64 `json:"available_margin_usdt"`
	PositionsCount      int     `json:"positions_count"`
	UnrealizedPNLUSDT   float64 `json:"unrealized_pnl_usdt"`
}

// Portfolio runs the `portfolio` command.
func Portfolio(ctx context.Context, c *Ctx) ExitCode {
	assets, err := c.API.GetAssets(ctx)
	if err != nil {
		return printErr(c, err)
	}
	var balance, available, upnl float64
	for _, a := range assets {
		if a.Currency == "USDT" {
			balance = a.AvailableCash + a.PositionMargin
			available = a.AvailableOpen
			upnl = a.UnrealizedPNL
		}
	}
	positions, err := c.API.GetOpenPositions(ctx, "")
	if err != nil {
		return printErr(c, err)
	}
	out := PortfolioOutput{
		BalanceUSDT:         balance,
		AvailableMarginUSDT: available,
		PositionsCount:      len(positions),
		UnrealizedPNLUSDT:   upnl,
	}
	if c.JSON {
		_ = output.WriteJSON(c.Stdout, out)
	} else {
		output.WritePortfolio(c.Stdout, balance, available, upnl, len(positions))
	}
	return ExitOK
}

// printErr maps an error to an exit code and writes the message.
// Shared by all commands.
func printErr(c *Ctx, err error) ExitCode {
	if errors.Is(err, mexc.ErrAuth) {
		if c.JSON {
			_ = output.WriteJSONError(c.Stderr, "AUTH", err.Error(), int(ExitAuth))
		} else {
			fmt.Fprintf(c.Stderr, "auth error: %v\n", err)
		}
		return ExitAuth
	}
	if c.JSON {
		_ = output.WriteJSONError(c.Stderr, "NETWORK", err.Error(), int(ExitNetwork))
	} else {
		fmt.Fprintf(c.Stderr, "error: %v\n", err)
	}
	return ExitNetwork
}
