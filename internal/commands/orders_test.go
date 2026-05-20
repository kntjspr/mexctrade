package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kntjspr/mexctrade/internal/mexc"
)

func TestOrdersJSON(t *testing.T) {
	out := &bytes.Buffer{}
	c := &Ctx{
		API: &stubAPI{orders: []mexc.Order{{
			OrderID: "id1", Symbol: "LAB_USDT", Side: 3, Type: 1, Price: 1.5, Vol: 100, State: 2,
		}}},
		Stdout: out, Stderr: &bytes.Buffer{}, JSON: true,
	}
	if code := Orders(context.Background(), c); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var got []OrderOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got[0].Side != "open short" || got[0].Type != "limit" {
		t.Errorf("got %+v", got)
	}
}
