package risk

import (
	"errors"
	"testing"
)

func TestComputeLongMarket(t *testing.T) {
	in := Inputs{
		Balance: 1000, RiskPct: 2,
		Entry: 100, SL: 95, Side: SideLong,
		MaxLeverage: 20, SymbolMaxLeverage: 50, ContractSize: 0.01,
	}
	out, err := Compute(in)
	if err != nil {
		t.Fatalf("unexpected refusal: %v", err)
	}
	if out.RiskAmount != 20 {
		t.Errorf("RiskAmount: got %f want 20", out.RiskAmount)
	}
	if out.SLDistancePct < 0.0499 || out.SLDistancePct > 0.0501 {
		t.Errorf("SLDistancePct: got %f want ~0.05", out.SLDistancePct)
	}
	if out.NotionalUSDT < 399.9 || out.NotionalUSDT > 400.1 {
		t.Errorf("NotionalUSDT: got %f want ~400", out.NotionalUSDT)
	}
	if out.Leverage != 1 {
		t.Errorf("Leverage: got %d want 1 (notional 400 < balance 1000)", out.Leverage)
	}
	// notional 400 USDT / entry 100 / contract_size 0.01 = 400 contracts exactly
	if out.Contracts != 400 {
		t.Errorf("Contracts: got %d want 400", out.Contracts)
	}
}

func TestComputeShortLimit(t *testing.T) {
	in := Inputs{
		Balance: 500, RiskPct: 1,
		Entry: 200, SL: 210, Side: SideShort,
		MaxLeverage: 20, SymbolMaxLeverage: 100, ContractSize: 0.001,
	}
	out, err := Compute(in)
	if err != nil {
		t.Fatalf("unexpected refusal: %v", err)
	}
	if out.RiskAmount != 5 {
		t.Errorf("RiskAmount: got %f want 5", out.RiskAmount)
	}
	if out.NotionalUSDT < 99.9 || out.NotionalUSDT > 100.1 {
		t.Errorf("NotionalUSDT: got %f want ~100", out.NotionalUSDT)
	}
	// notional 100 USDT / entry 200 / contract_size 0.001 = 500 contracts
	if out.Contracts != 500 {
		t.Errorf("Contracts: got %d want 500", out.Contracts)
	}
}

func TestRefusalNoSL(t *testing.T) {
	in := Inputs{Balance: 100, RiskPct: 2, Entry: 100, SL: 0, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 20, ContractSize: 1}
	_, err := Compute(in)
	if !errors.Is(err, ErrNoSL) {
		t.Fatalf("got %v want ErrNoSL", err)
	}
}

func TestRefusalRiskPctTooHigh(t *testing.T) {
	in := Inputs{Balance: 100, RiskPct: 10, Entry: 100, SL: 90, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 20, ContractSize: 1}
	_, err := Compute(in)
	if !errors.Is(err, ErrRiskPctOutOfBounds) {
		t.Fatalf("got %v want ErrRiskPctOutOfBounds", err)
	}
}

func TestRefusalRiskPctZero(t *testing.T) {
	in := Inputs{Balance: 100, RiskPct: 0, Entry: 100, SL: 90, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 20, ContractSize: 1}
	_, err := Compute(in)
	if !errors.Is(err, ErrRiskPctOutOfBounds) {
		t.Fatalf("got %v want ErrRiskPctOutOfBounds", err)
	}
}

func TestRefusalSLOnWrongSideLong(t *testing.T) {
	in := Inputs{Balance: 100, RiskPct: 2, Entry: 100, SL: 110, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 20, ContractSize: 1}
	_, err := Compute(in)
	if !errors.Is(err, ErrSLWrongSide) {
		t.Fatalf("got %v want ErrSLWrongSide", err)
	}
}

func TestRefusalSLOnWrongSideShort(t *testing.T) {
	in := Inputs{Balance: 100, RiskPct: 2, Entry: 100, SL: 90, Side: SideShort, MaxLeverage: 20, SymbolMaxLeverage: 20, ContractSize: 1}
	_, err := Compute(in)
	if !errors.Is(err, ErrSLWrongSide) {
		t.Fatalf("got %v want ErrSLWrongSide", err)
	}
}

func TestRefusalTPOnWrongSide(t *testing.T) {
	in := Inputs{Balance: 100, RiskPct: 2, Entry: 100, SL: 95, TP: 90, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 20, ContractSize: 1}
	_, err := Compute(in)
	if !errors.Is(err, ErrTPWrongSide) {
		t.Fatalf("got %v want ErrTPWrongSide", err)
	}
}

func TestRefusalSLTooTight(t *testing.T) {
	in := Inputs{Balance: 1000, RiskPct: 2, Entry: 100, SL: 99.95, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 20, ContractSize: 0.01}
	_, err := Compute(in)
	if !errors.Is(err, ErrSLTooTight) {
		t.Fatalf("got %v want ErrSLTooTight", err)
	}
}

func TestRefusalLeverageExceedsMax(t *testing.T) {
	// risk_amount = 5, sl_pct = 0.0011, notional = 4545.45,
	// raw_leverage = ceil(4545.45/100) = 46 > MaxLeverage=20 -> refuse.
	// SL is just above the SL_TOO_TIGHT threshold (0.001) so that check passes.
	in := Inputs{Balance: 100, RiskPct: 5, Entry: 100, SL: 99.89, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 100, ContractSize: 0.01}
	_, err := Compute(in)
	if !errors.Is(err, ErrLeverageExceedsMax) {
		t.Fatalf("got %v want ErrLeverageExceedsMax", err)
	}
}

func TestRefusalContractsZero(t *testing.T) {
	// risk_amount = 0.1, sl_pct = 0.02, notional = 5 USDT.
	// contracts = floor(5 / 50000 / 0.001) = floor(0.1) = 0 -> refuse.
	in := Inputs{Balance: 100, RiskPct: 0.1, Entry: 50000, SL: 49000, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 100, ContractSize: 0.001}
	_, err := Compute(in)
	if !errors.Is(err, ErrContractsZero) {
		t.Fatalf("got %v want ErrContractsZero", err)
	}
}
