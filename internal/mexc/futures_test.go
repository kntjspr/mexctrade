package mexc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(srv.URL, "k", "s")
	c.sleep = func(_ int) {}
	return c
}

func TestPing(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/contract/ping" {
			t.Errorf("path: %s", r.URL.Path)
		}
		io.WriteString(w, `{"success":true,"code":0,"data":1779000000000}`)
	})
	srvTime, err := c.Ping(context.Background())
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if srvTime != 1779000000000 {
		t.Errorf("got %d", srvTime)
	}
}

func TestGetAssets(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/private/account/assets" {
			t.Errorf("path: %s", r.URL.Path)
		}
		io.WriteString(w, `{"success":true,"code":0,"data":[{"currency":"USDT","availableCash":1000.5,"availableOpen":900.0,"positionMargin":50.0,"unrealized":3.0}]}`)
	})
	assets, err := c.GetAssets(context.Background())
	if err != nil {
		t.Fatalf("GetAssets: %v", err)
	}
	if len(assets) != 1 || assets[0].Currency != "USDT" || assets[0].AvailableCash != 1000.5 {
		t.Errorf("assets wrong: %+v", assets)
	}
}

func TestGetOpenPositions(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/private/position/open_positions") {
			t.Errorf("path: %s", r.URL.Path)
		}
		io.WriteString(w, `{"success":true,"code":0,"data":[{"symbol":"BTC_USDT","positionType":1,"holdVol":100,"holdAvgPrice":50000,"markPrice":50500,"leverage":10}]}`)
	})
	ps, err := c.GetOpenPositions(context.Background(), "")
	if err != nil {
		t.Fatalf("GetOpenPositions: %v", err)
	}
	if len(ps) != 1 || ps[0].Symbol != "BTC_USDT" {
		t.Errorf("positions: %+v", ps)
	}
}

func TestGetOpenOrders(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"success":true,"code":0,"data":[{"orderId":"123","symbol":"LAB_USDT","side":3,"orderType":1,"price":1.5,"vol":100,"state":2}]}`)
	})
	orders, err := c.GetOpenOrders(context.Background(), "")
	if err != nil {
		t.Fatalf("GetOpenOrders: %v", err)
	}
	if len(orders) != 1 || orders[0].OrderID != "123" {
		t.Errorf("orders: %+v", orders)
	}
}

func TestChangeLeverage(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/private/position/change_leverage" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["symbol"] != "BTC_USDT" || body["leverage"] != float64(10) {
			t.Errorf("body wrong: %+v", body)
		}
		io.WriteString(w, `{"success":true,"code":0,"data":null}`)
	})
	if err := c.ChangeLeverage(context.Background(), "BTC_USDT", 10, 1); err != nil {
		t.Fatalf("ChangeLeverage: %v", err)
	}
}

func TestPlaceOrder(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/private/order/submit" {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body PlaceOrderRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Symbol != "BTC_USDT" || body.Side != 1 || body.Type != 5 {
			t.Errorf("body wrong: %+v", body)
		}
		io.WriteString(w, `{"success":true,"code":0,"data":"orderid-99"}`)
	})
	req := PlaceOrderRequest{Symbol: "BTC_USDT", Side: 1, Type: 5, Vol: 10, Leverage: 5, OpenType: 1}
	id, err := c.PlaceOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if id != "orderid-99" {
		t.Errorf("id: %s", id)
	}
}

func TestCancelAll(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/private/order/cancel_all" {
			t.Errorf("path: %s", r.URL.Path)
		}
		io.WriteString(w, `{"success":true,"code":0,"data":null}`)
	})
	if err := c.CancelAll(context.Background(), "LAB_USDT"); err != nil {
		t.Fatalf("CancelAll: %v", err)
	}
}

func TestFetchContracts(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/contract/detail" {
			t.Errorf("path: %s", r.URL.Path)
		}
		io.WriteString(w, `{"success":true,"code":0,"data":[{"symbol":"BTC_USDT","contractSize":0.0001,"maxLeverage":100,"priceScale":1,"volScale":0,"state":0}]}`)
	})
	cs, err := c.fetchContracts()
	if err != nil {
		t.Fatalf("fetchContracts: %v", err)
	}
	if len(cs) != 1 || cs[0].Symbol != "BTC_USDT" {
		t.Errorf("contracts: %+v", cs)
	}
}
