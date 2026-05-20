package mexc

import (
	"context"
	"fmt"
	"net/http"
)

// Ping returns the server time in ms.
func (c *Client) Ping(ctx context.Context) (int64, error) {
	var env Envelope[int64]
	if err := c.do(ctx, http.MethodGet, "/api/v1/contract/ping", nil, nil, &env); err != nil {
		return 0, err
	}
	return env.Data, nil
}

func (c *Client) GetAssets(ctx context.Context) ([]Asset, error) {
	var env Envelope[[]Asset]
	if err := c.do(ctx, http.MethodGet, "/api/v1/private/account/assets", nil, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) GetOpenPositions(ctx context.Context, symbol string) ([]Position, error) {
	params := map[string]string{}
	if symbol != "" {
		params["symbol"] = symbol
	}
	var env Envelope[[]Position]
	if err := c.do(ctx, http.MethodGet, "/api/v1/private/position/open_positions", params, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) GetOpenOrders(ctx context.Context, symbol string) ([]Order, error) {
	params := map[string]string{}
	if symbol != "" {
		params["symbol"] = symbol
	}
	var env Envelope[[]Order]
	if err := c.do(ctx, http.MethodGet, "/api/v1/private/order/list/open_orders", params, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// ChangeLeverage sets leverage for a symbol. openType: 1=cross, 2=isolated.
func (c *Client) ChangeLeverage(ctx context.Context, symbol string, leverage, openType int) error {
	body := map[string]any{
		"symbol":   symbol,
		"leverage": leverage,
		"openType": openType,
	}
	var env Envelope[any]
	return c.do(ctx, http.MethodPost, "/api/v1/private/position/change_leverage", nil, body, &env)
}

// PlaceOrder returns the orderId on success.
func (c *Client) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (string, error) {
	var env Envelope[string]
	if err := c.do(ctx, http.MethodPost, "/api/v1/private/order/submit", nil, req, &env); err != nil {
		return "", err
	}
	if !env.Success {
		return "", fmt.Errorf("MEXC code %d: %s", env.Code, env.Message)
	}
	return env.Data, nil
}

func (c *Client) CancelAll(ctx context.Context, symbol string) error {
	body := map[string]any{"symbol": symbol}
	var env Envelope[any]
	return c.do(ctx, http.MethodPost, "/api/v1/private/order/cancel_all", nil, body, &env)
}

func (c *Client) CancelOrders(ctx context.Context, orderIDs []string) error {
	var env Envelope[any]
	return c.do(ctx, http.MethodPost, "/api/v1/private/order/cancel", nil, orderIDs, &env)
}

// fetchContracts is unexported — called from the symbol Resolver only.
func (c *Client) fetchContracts() ([]Contract, error) {
	var env Envelope[[]Contract]
	if err := c.do(context.Background(), http.MethodGet, "/api/v1/contract/detail", nil, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}
