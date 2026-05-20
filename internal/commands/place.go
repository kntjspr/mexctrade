package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/kntjspr/mexctrade/internal/mexc"
	"github.com/kntjspr/mexctrade/internal/output"
	"github.com/kntjspr/mexctrade/internal/risk"
)

// PlaceArgs is the input bundle for `mexctrade place`.
type PlaceArgs struct {
	Symbol     string
	Side       string  // "long" | "short"
	Entry      string  // "market" or a price string
	EntryPrice float64 // current mark price when Entry == "market"
	SL         float64
	TP         float64 // 0 means not provided
	RiskPct    float64
}

// PlaceLiveOutput is the JSON shape for a successful live order.
type PlaceLiveOutput struct {
	DryRun    bool    `json:"dry_run"`
	OrderID   string  `json:"order_id"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Type      string  `json:"type"`
	Entry     float64 `json:"entry"`
	SL        float64 `json:"sl"`
	TP        float64 `json:"tp,omitempty"`
	Leverage  int     `json:"leverage"`
	Contracts int     `json:"contracts"`
}

// PlaceDryRunOutput is the JSON shape for a dry-run.
type PlaceDryRunOutput struct {
	DryRun      bool                   `json:"dry_run"`
	WouldCall   string                 `json:"would_call"`
	RequestBody mexc.PlaceOrderRequest `json:"request_body"`
	Computed    risk.Result            `json:"computed"`
}

// Place runs the `place` command.
func Place(ctx context.Context, c *Ctx, a PlaceArgs) ExitCode {
	contract, err := c.Resolver.Resolve(a.Symbol)
	if err != nil {
		var unk *mexc.UnknownSymbolError
		if errors.As(err, &unk) {
			emitErr(c, "UNKNOWN_SYMBOL", err.Error(), ExitUnknown)
			return ExitUnknown
		}
		return printErr(c, err)
	}

	assets, err := c.API.GetAssets(ctx)
	if err != nil {
		return printErr(c, err)
	}
	var balance float64
	for _, asset := range assets {
		if asset.Currency == "USDT" {
			balance = asset.AvailableCash + asset.PositionMargin
		}
	}
	if balance == 0 {
		emitErr(c, "NO_BALANCE", "no USDT balance found", ExitRefused)
		return ExitRefused
	}

	entry := a.EntryPrice
	orderType := 5 // market
	if a.Entry != "market" {
		p, perr := strconv.ParseFloat(a.Entry, 64)
		if perr != nil {
			emitErr(c, "BAD_ENTRY", "--entry must be 'market' or a price", ExitUsage)
			return ExitUsage
		}
		entry = p
		orderType = 1 // limit
	}
	if entry == 0 {
		emitErr(c, "BAD_ENTRY", "market price unknown; supply --entry-price", ExitUsage)
		return ExitUsage
	}

	var side risk.Side
	switch a.Side {
	case "long":
		side = risk.SideLong
	case "short":
		side = risk.SideShort
	default:
		emitErr(c, "BAD_SIDE", "--side must be long or short", ExitUsage)
		return ExitUsage
	}

	in := risk.Inputs{
		Balance:           balance,
		RiskPct:           a.RiskPct,
		Entry:             entry,
		SL:                a.SL,
		TP:                a.TP,
		Side:              side,
		MaxLeverage:       c.Cfg.MaxLeverage,
		SymbolMaxLeverage: contract.MaxLeverage,
		ContractSize:      contract.ContractSize,
	}
	result, err := risk.Compute(in)
	if err != nil {
		emitErr(c, riskCode(err), err.Error(), ExitRefused)
		return ExitRefused
	}

	apiSide := 1 // open long
	if side == risk.SideShort {
		apiSide = 3
	}
	req := mexc.PlaceOrderRequest{
		Symbol:          contract.Symbol,
		Side:            apiSide,
		Type:            orderType,
		OpenType:        1, // cross
		PositionMode:    1, // one-way
		Leverage:        result.Leverage,
		Vol:             result.Contracts,
		StopLossPrice:   a.SL,
		TakeProfitPrice: a.TP,
	}
	if orderType == 1 {
		req.Price = entry
	}

	if c.DryRun {
		out := PlaceDryRunOutput{
			DryRun:      true,
			WouldCall:   "POST /api/v1/private/order/submit",
			RequestBody: req,
			Computed:    *result,
		}
		if c.JSON {
			_ = output.WriteJSON(c.Stdout, out)
		} else {
			fmt.Fprintf(c.Stdout, "[DRY-RUN] would place %s %s %s @ %v sl=%v tp=%v leverage=%dx contracts=%d\n",
				a.Side, contract.Symbol, typeName(orderType), entry, a.SL, a.TP, result.Leverage, result.Contracts)
		}
		return ExitOK
	}

	if err := c.API.ChangeLeverage(ctx, contract.Symbol, result.Leverage, 1); err != nil {
		return printErr(c, err)
	}
	orderID, err := c.API.PlaceOrder(ctx, req)
	if err != nil {
		return printErr(c, err)
	}

	live := PlaceLiveOutput{
		DryRun: false, OrderID: orderID, Symbol: contract.Symbol,
		Side: a.Side, Type: typeName(orderType), Entry: entry,
		SL: a.SL, TP: a.TP, Leverage: result.Leverage, Contracts: result.Contracts,
	}
	if c.JSON {
		_ = output.WriteJSON(c.Stdout, live)
	} else {
		fmt.Fprintf(c.Stdout, "placed %s %s %s @ %v leverage=%dx contracts=%d  (order=%s)\n",
			a.Side, contract.Symbol, typeName(orderType), entry, result.Leverage, result.Contracts, orderID)
	}
	return ExitOK
}

// emitErr writes a structured error in JSON mode, or a plain message in pretty mode.
// Used by all write-path commands.
func emitErr(c *Ctx, code, msg string, exit ExitCode) {
	if c.JSON {
		_ = output.WriteJSONError(c.Stderr, code, msg, int(exit))
	} else {
		fmt.Fprintf(c.Stderr, "%s: %s\n", code, msg)
	}
}

// riskCode maps a risk.Compute sentinel error to its public code.
func riskCode(err error) string {
	switch {
	case errors.Is(err, risk.ErrNoSL):
		return "RISK_NO_SL"
	case errors.Is(err, risk.ErrRiskPctOutOfBounds):
		return "RISK_PCT_OUT_OF_BOUNDS"
	case errors.Is(err, risk.ErrSLWrongSide):
		return "RISK_SL_WRONG_SIDE"
	case errors.Is(err, risk.ErrTPWrongSide):
		return "RISK_TP_WRONG_SIDE"
	case errors.Is(err, risk.ErrSLTooTight):
		return "RISK_SL_TOO_TIGHT"
	case errors.Is(err, risk.ErrLeverageExceedsMax):
		return "RISK_LEVERAGE_EXCEEDS_MAX"
	case errors.Is(err, risk.ErrContractsZero):
		return "RISK_CONTRACTS_ZERO"
	}
	return "RISK_UNKNOWN"
}

func typeName(t int) string {
	if t == 5 {
		return "market"
	}
	return "limit"
}
