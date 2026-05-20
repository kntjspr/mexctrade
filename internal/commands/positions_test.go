package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kntjspr/mexctrade/internal/mexc"
)

func TestPositionsJSON(t *testing.T) {
	out := &bytes.Buffer{}
	c := &Ctx{
		API: &stubAPI{positions: []mexc.Position{{
			Symbol: "BTC_USDT", PositionType: 1, HoldVol: 100,
			HoldAvgPrice: 50000, MarkPrice: 50500, UnrealizedPNL: 50, Leverage: 10,
		}}},
		Stdout: out, Stderr: &bytes.Buffer{}, JSON: true,
	}
	if code := Positions(context.Background(), c); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var got []PositionOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].Side != "long" || got[0].EntryPrice != 50000 {
		t.Errorf("got %+v", got)
	}
}
