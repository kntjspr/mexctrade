package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kntjspr/mexctrade/internal/config"
	"github.com/kntjspr/mexctrade/internal/mexc"
)

// stubResolver returns a fixed contract for all symbols, useful when the test
// already knows which contract should resolve.
type stubResolver struct {
	c   *mexc.Contract
	err error
}

func (s *stubResolver) Resolve(_ string) (*mexc.Contract, error) { return s.c, s.err }

// stubAPIPlace records ChangeLeverage / PlaceOrder calls so tests can assert
// the exact request shape the place command builds.
type stubAPIPlace struct {
	stubAPI
	leverageCalls int
	placeCalls    int
	lastPlace     mexc.PlaceOrderRequest
	placeOut      string
}

func (s *stubAPIPlace) ChangeLeverage(_ context.Context, _ string, _, _ int) error {
	s.leverageCalls++
	return nil
}
func (s *stubAPIPlace) PlaceOrder(_ context.Context, req mexc.PlaceOrderRequest) (string, error) {
	s.placeCalls++
	s.lastPlace = req
	return s.placeOut, nil
}

func newPlaceCtx(t *testing.T, json, dryRun bool, contract *mexc.Contract, assets []mexc.Asset) (*Ctx, *bytes.Buffer, *stubAPIPlace) {
	t.Helper()
	api := &stubAPIPlace{
		stubAPI:  stubAPI{assets: assets},
		placeOut: "order-1",
	}
	out := &bytes.Buffer{}
	return &Ctx{
		API:      api,
		Resolver: &stubResolver{c: contract},
		Stdout:   out,
		Stderr:   &bytes.Buffer{},
		JSON:     json,
		DryRun:   dryRun,
		Cfg:      &config.Config{MaxLeverage: 20},
	}, out, api
}

func TestPlaceLongMarketDryRun(t *testing.T) {
	ctr := &mexc.Contract{Symbol: "BTC_USDT", ContractSize: 0.0001, MaxLeverage: 100}
	assets := []mexc.Asset{{Currency: "USDT", AvailableCash: 1000, AvailableOpen: 1000}}
	c, out, api := newPlaceCtx(t, true, true, ctr, assets)

	code := Place(context.Background(), c, PlaceArgs{
		Symbol: "BTC", Side: "long", Entry: "market", EntryPrice: 50000,
		SL: 49500, TP: 0, RiskPct: 2,
	})
	if code != ExitOK {
		t.Fatalf("exit %d, body: %s", code, out.String())
	}
	if api.placeCalls != 0 {
		t.Errorf("dry-run should not call PlaceOrder; got %d", api.placeCalls)
	}
	if api.leverageCalls != 0 {
		t.Errorf("dry-run should not call ChangeLeverage; got %d", api.leverageCalls)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["dry_run"] != true {
		t.Errorf("dry_run flag missing: %+v", result)
	}
	if result["would_call"] != "POST /api/v1/private/order/submit" {
		t.Errorf("would_call wrong: %v", result["would_call"])
	}
}

func TestPlaceLiveExecutes(t *testing.T) {
	ctr := &mexc.Contract{Symbol: "BTC_USDT", ContractSize: 0.0001, MaxLeverage: 100}
	assets := []mexc.Asset{{Currency: "USDT", AvailableCash: 1000, AvailableOpen: 1000}}
	c, _, api := newPlaceCtx(t, true, false, ctr, assets)

	code := Place(context.Background(), c, PlaceArgs{
		Symbol: "BTC", Side: "long", Entry: "market", EntryPrice: 50000,
		SL: 49500, RiskPct: 2,
	})
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if api.placeCalls != 1 {
		t.Errorf("expected 1 PlaceOrder call, got %d", api.placeCalls)
	}
	if api.lastPlace.Symbol != "BTC_USDT" {
		t.Errorf("symbol wrong: %s", api.lastPlace.Symbol)
	}
	if api.lastPlace.Side != 1 {
		t.Errorf("side wrong: %d (want 1 = open long)", api.lastPlace.Side)
	}
	if api.lastPlace.Type != 5 {
		t.Errorf("type wrong: %d (want 5 = market)", api.lastPlace.Type)
	}
	if api.lastPlace.OpenType != 1 {
		t.Errorf("openType wrong: %d (want 1 = cross)", api.lastPlace.OpenType)
	}
	if api.lastPlace.StopLossPrice != 49500 {
		t.Errorf("SL wrong: %f", api.lastPlace.StopLossPrice)
	}
}

func TestPlaceLimitOrderHasPrice(t *testing.T) {
	ctr := &mexc.Contract{Symbol: "BTC_USDT", ContractSize: 0.0001, MaxLeverage: 100}
	assets := []mexc.Asset{{Currency: "USDT", AvailableCash: 1000, AvailableOpen: 1000}}
	c, _, api := newPlaceCtx(t, true, false, ctr, assets)

	code := Place(context.Background(), c, PlaceArgs{
		Symbol: "BTC", Side: "short", Entry: "49000", EntryPrice: 0,
		SL: 49500, TP: 47000, RiskPct: 1,
	})
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if api.lastPlace.Type != 1 {
		t.Errorf("type wrong: %d (want 1 = limit)", api.lastPlace.Type)
	}
	if api.lastPlace.Price != 49000 {
		t.Errorf("price wrong: %f", api.lastPlace.Price)
	}
	if api.lastPlace.Side != 3 {
		t.Errorf("side wrong: %d (want 3 = open short)", api.lastPlace.Side)
	}
	if api.lastPlace.TakeProfitPrice != 47000 {
		t.Errorf("TP wrong: %f", api.lastPlace.TakeProfitPrice)
	}
}

func TestPlaceRefusesNoSL(t *testing.T) {
	ctr := &mexc.Contract{Symbol: "BTC_USDT", ContractSize: 0.0001, MaxLeverage: 100}
	assets := []mexc.Asset{{Currency: "USDT", AvailableCash: 1000, AvailableOpen: 1000}}
	c, _, _ := newPlaceCtx(t, true, false, ctr, assets)
	code := Place(context.Background(), c, PlaceArgs{
		Symbol: "BTC", Side: "long", Entry: "market", EntryPrice: 50000,
		SL: 0, RiskPct: 2,
	})
	if code != ExitRefused {
		t.Errorf("expected ExitRefused, got %d", code)
	}
}
