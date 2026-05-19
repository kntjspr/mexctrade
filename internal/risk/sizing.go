package risk

import (
	"errors"
	"math"
)

type Side int

const (
	SideLong Side = iota + 1
	SideShort
)

type Inputs struct {
	Balance           float64
	RiskPct           float64
	Entry             float64
	SL                float64
	TP                float64  // 0 means not provided
	Side              Side
	MaxLeverage       int
	SymbolMaxLeverage int
	ContractSize      float64
}

type Result struct {
	RiskAmount    float64
	SLDistancePct float64
	NotionalUSDT  float64
	RawLeverage   int
	Leverage      int
	Contracts     int
}

var (
	ErrNoSL               = errors.New("RISK_NO_SL: stop loss required")
	ErrRiskPctOutOfBounds = errors.New("RISK_PCT_OUT_OF_BOUNDS: --risk-pct must be > 0 and <= 5")
	ErrSLWrongSide        = errors.New("RISK_SL_WRONG_SIDE: SL must be below entry for long, above for short")
	ErrTPWrongSide        = errors.New("RISK_TP_WRONG_SIDE: TP must be above entry for long, below for short")
	ErrSLTooTight         = errors.New("RISK_SL_TOO_TIGHT: SL distance < 0.1% of entry, vision likely misread")
	ErrLeverageExceedsMax = errors.New("RISK_LEVERAGE_EXCEEDS_MAX: required leverage exceeds max_leverage")
	ErrContractsZero      = errors.New("RISK_CONTRACTS_ZERO: position too small to express as valid contract increment")
)

// Compute runs the full sizing pipeline. Refusal errors are sentinel errors
// (errors.Is-comparable). On success, all Result fields are populated.
func Compute(in Inputs) (*Result, error) {
	if in.SL == 0 {
		return nil, ErrNoSL
	}
	if in.RiskPct <= 0 || in.RiskPct > 5 {
		return nil, ErrRiskPctOutOfBounds
	}
	if in.Side == SideLong && in.SL >= in.Entry {
		return nil, ErrSLWrongSide
	}
	if in.Side == SideShort && in.SL <= in.Entry {
		return nil, ErrSLWrongSide
	}
	if in.TP != 0 {
		if in.Side == SideLong && in.TP <= in.Entry {
			return nil, ErrTPWrongSide
		}
		if in.Side == SideShort && in.TP >= in.Entry {
			return nil, ErrTPWrongSide
		}
	}

	r := &Result{}
	r.RiskAmount = in.Balance * (in.RiskPct / 100)
	r.SLDistancePct = math.Abs(in.Entry-in.SL) / in.Entry
	if r.SLDistancePct < 0.001 {
		return nil, ErrSLTooTight
	}
	r.NotionalUSDT = r.RiskAmount / r.SLDistancePct
	r.RawLeverage = int(math.Ceil(r.NotionalUSDT / in.Balance))
	if r.RawLeverage > in.MaxLeverage {
		return nil, ErrLeverageExceedsMax
	}
	r.Leverage = r.RawLeverage
	if r.Leverage > in.SymbolMaxLeverage {
		r.Leverage = in.SymbolMaxLeverage
	}
	r.Contracts = int(math.Floor(r.NotionalUSDT / in.Entry / in.ContractSize))
	if r.Contracts == 0 {
		return nil, ErrContractsZero
	}
	return r, nil
}
