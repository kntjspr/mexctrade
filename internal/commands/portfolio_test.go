package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kntjspr/mexctrade/internal/mexc"
)

// stubAPI is the shared MexcAPI stub for command tests.
type stubAPI struct {
	assets    []mexc.Asset
	positions []mexc.Position
	orders    []mexc.Order
	assetsErr error
}

func (s *stubAPI) Ping(_ context.Context) (int64, error) { return 0, nil }
func (s *stubAPI) GetAssets(_ context.Context) ([]mexc.Asset, error) {
	return s.assets, s.assetsErr
}
func (s *stubAPI) GetOpenPositions(_ context.Context, _ string) ([]mexc.Position, error) {
	return s.positions, nil
}
func (s *stubAPI) GetOpenOrders(_ context.Context, _ string) ([]mexc.Order, error) {
	return s.orders, nil
}
func (s *stubAPI) ChangeLeverage(_ context.Context, _ string, _, _ int) error             { return nil }
func (s *stubAPI) PlaceOrder(_ context.Context, _ mexc.PlaceOrderRequest) (string, error) { return "", nil }
func (s *stubAPI) CancelAll(_ context.Context, _ string) error                            { return nil }
func (s *stubAPI) CancelOrders(_ context.Context, _ []string) error                      { return nil }

func TestPortfolioJSON(t *testing.T) {
	out := &bytes.Buffer{}
	c := &Ctx{
		API: &stubAPI{
			assets: []mexc.Asset{{
				Currency: "USDT", AvailableCash: 800, AvailableOpen: 700,
				PositionMargin: 200, UnrealizedPNL: 5,
			}},
			positions: []mexc.Position{{Symbol: "BTC_USDT"}, {Symbol: "ETH_USDT"}},
		},
		Stdout: out, Stderr: &bytes.Buffer{},
		JSON: true,
	}
	if code := Portfolio(context.Background(), c); code != ExitOK {
		t.Fatalf("exit code: %d", code)
	}
	var got PortfolioOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.BalanceUSDT != 1000 || got.AvailableMarginUSDT != 700 || got.PositionsCount != 2 || got.UnrealizedPNLUSDT != 5 {
		t.Errorf("got %+v", got)
	}
}

func TestPortfolioAuthError(t *testing.T) {
	c := &Ctx{
		API:    &stubAPI{assetsErr: mexc.ErrAuth},
		Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
		JSON: true,
	}
	if code := Portfolio(context.Background(), c); code != ExitAuth {
		t.Errorf("got %d want %d", code, ExitAuth)
	}
}
