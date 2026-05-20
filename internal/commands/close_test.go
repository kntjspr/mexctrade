package commands

import (
	"bytes"
	"context"
	"testing"

	"github.com/kntjspr/mexctrade/internal/mexc"
)

type stubAPIClose struct {
	stubAPI
	placeCalls int
	lastPlace  mexc.PlaceOrderRequest
}

func (s *stubAPIClose) PlaceOrder(_ context.Context, req mexc.PlaceOrderRequest) (string, error) {
	s.placeCalls++
	s.lastPlace = req
	return "close-id", nil
}

func TestCloseLong(t *testing.T) {
	api := &stubAPIClose{
		stubAPI: stubAPI{positions: []mexc.Position{{
			Symbol: "BTC_USDT", PositionType: 1, HoldVol: 100,
		}}},
	}
	out := &bytes.Buffer{}
	c := &Ctx{
		API:      api,
		Resolver: &stubResolver{c: &mexc.Contract{Symbol: "BTC_USDT"}},
		Stdout:   out, Stderr: &bytes.Buffer{}, JSON: true,
	}
	code := Close(context.Background(), c, CloseArgs{Symbol: "BTC"})
	if code != ExitOK {
		t.Fatalf("exit %d body: %s", code, out.String())
	}
	if api.placeCalls != 1 {
		t.Errorf("expected 1 PlaceOrder, got %d", api.placeCalls)
	}
	if api.lastPlace.Side != 2 {
		t.Errorf("side wrong: %d (want 2 = close long)", api.lastPlace.Side)
	}
	if api.lastPlace.Vol != 100 {
		t.Errorf("vol wrong: %d", api.lastPlace.Vol)
	}
	if !api.lastPlace.ReduceOnly {
		t.Errorf("ReduceOnly must be true")
	}
}

func TestCloseNoPosition(t *testing.T) {
	api := &stubAPIClose{}
	c := &Ctx{
		API:      api,
		Resolver: &stubResolver{c: &mexc.Contract{Symbol: "BTC_USDT"}},
		Stdout:   &bytes.Buffer{}, Stderr: &bytes.Buffer{}, JSON: true,
	}
	code := Close(context.Background(), c, CloseArgs{Symbol: "BTC"})
	if code != ExitRefused {
		t.Errorf("expected ExitRefused (no position to close), got %d", code)
	}
}
