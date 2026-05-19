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
	if out.Contracts != 399 {
		t.Errorf("Contracts: got %d want 399", out.Contracts)
	}
}

func TestComputeShortLimit(t *testing.T) {
	in := Inputs{
		Balance: 500, RiskPct: 1,
		Entry: 200, SL: 210, Side: SideShort,
		MaxLeverage: 20, SymbolMaxLeverage: 100, ContractSize: 1,
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
	if out.Contracts != 0 {
		t.Errorf("Contracts: got %d want 0 (100 USDT / 200 entry / 1 cs = 0.5 -> floor 0)", out.Contracts)
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
	// notional 4000 vs balance 100 -> raw_leverage 40 > max 20 -> refuse
	in := Inputs{Balance: 100, RiskPct: 2, Entry: 100, SL: 99.95, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 100, ContractSize: 0.01}
	// First SL_TOO_TIGHT will fire because 0.05% < 0.1%, swap to tight-but-allowed:
	in.SL = 99.5
	_, err := Compute(in)
	if !errors.Is(err, ErrLeverageExceedsMax) {
		t.Fatalf("got %v want ErrLeverageExceedsMax", err)
	}
}

func TestRefusalContractsZero(t *testing.T) {
	// notional 1 USDT / entry 50000 / contract_size 0.001 -> 0.02 -> floor 0
	in := Inputs{Balance: 100, RiskPct: 0.5, Entry: 50000, SL: 49500, Side: SideLong, MaxLeverage: 20, SymbolMaxLeverage: 100, ContractSize: 0.001}
	_, err := Compute(in)
	if !errors.Is(err, ErrContractsZero) {
		t.Fatalf("got %v want ErrContractsZero", err)
	}
}
