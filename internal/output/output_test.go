package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type portfolioRow struct {
	BalanceUSDT         float64 `json:"balance_usdt"`
	AvailableMarginUSDT float64 `json:"available_margin_usdt"`
	PositionsCount      int     `json:"positions_count"`
	UnrealizedPNLUSDT   float64 `json:"unrealized_pnl_usdt"`
}

func TestJSONPortfolio(t *testing.T) {
	out := &bytes.Buffer{}
	o := portfolioRow{BalanceUSDT: 1000, AvailableMarginUSDT: 900, PositionsCount: 2, UnrealizedPNLUSDT: 5.5}
	if err := WriteJSON(out, o); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var back portfolioRow
	if err := json.Unmarshal(out.Bytes(), &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.BalanceUSDT != 1000 || back.PositionsCount != 2 {
		t.Errorf("round-trip wrong: %+v", back)
	}
}

func TestJSONError(t *testing.T) {
	out := &bytes.Buffer{}
	if err := WriteJSONError(out, "RISK_NO_SL", "stop loss required", 4); err != nil {
		t.Fatalf("err: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, `"code":"RISK_NO_SL"`) {
		t.Errorf("missing code: %s", s)
	}
	if !strings.Contains(s, `"exit":4`) {
		t.Errorf("missing exit: %s", s)
	}
	if !strings.Contains(s, `"error":"stop loss required"`) {
		t.Errorf("missing error: %s", s)
	}
}

func TestPrettyPortfolio(t *testing.T) {
	out := &bytes.Buffer{}
	WritePortfolio(out, 1000, 900, 5.5, 2)
	s := out.String()
	for _, want := range []string{"Balance", "1,000.00", "Available", "900.00", "Positions", "uPnL"} {
		if !strings.Contains(s, want) {
			t.Errorf("pretty output missing %q in:\n%s", want, s)
		}
	}
}

func TestPrintLine(t *testing.T) {
	out := &bytes.Buffer{}
	PrintLine(out, "BTC_USDT", "long", 100, 50000.5)
	s := out.String()
	if !strings.Contains(s, "BTC_USDT") || !strings.Contains(s, "long") || !strings.Contains(s, "50000.5") {
		t.Errorf("PrintLine output wrong: %q", s)
	}
	if !strings.HasSuffix(s, "\n") {
		t.Errorf("PrintLine should end with newline: %q", s)
	}
}
