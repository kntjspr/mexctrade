package mexc

type Asset struct {
	Currency        string  `json:"currency"`
	AvailableCash   float64 `json:"availableCash"`
	AvailableOpen   float64 `json:"availableOpen"`
	PositionMargin  float64 `json:"positionMargin"`
	Bonus           float64 `json:"bonus"`
	UnrealizedPNL   float64 `json:"unrealized"`
}

type Position struct {
	PositionID   int64   `json:"positionId"`
	Symbol       string  `json:"symbol"`
	PositionType int     `json:"positionType"` // 1=long, 2=short
	OpenType     int     `json:"openType"`     // 1=cross, 2=isolated
	State        int     `json:"state"`
	HoldVol      int64   `json:"holdVol"`
	HoldAvgPrice float64 `json:"holdAvgPrice"`
	MarkPrice    float64 `json:"markPrice"`
	UnrealizedPNL float64 `json:"realised"` // MEXC field name
	Leverage     int     `json:"leverage"`
}

type Order struct {
	OrderID       string  `json:"orderId"`
	Symbol        string  `json:"symbol"`
	Side          int     `json:"side"` // 1=open long, 2=close short, 3=open short, 4=close long
	Type          int     `json:"orderType"`
	Price         float64 `json:"price"`
	Vol           int64   `json:"vol"`
	State         int     `json:"state"` // 1=uninformed, 2=uncompleted, 3=completed, 4=cancelled
	StopLossPrice float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice float64 `json:"takeProfitPrice,omitempty"`
}

type Contract struct {
	Symbol         string  `json:"symbol"`
	DisplayName    string  `json:"displayName"`
	ContractSize   float64 `json:"contractSize"`
	MaxLeverage    int     `json:"maxLeverage"`
	PriceScale     int     `json:"priceScale"`
	VolScale       int     `json:"volScale"`
	State          int     `json:"state"`
}

// envelope wraps every MEXC response.
type Envelope[T any] struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data"`
}

type PlaceOrderRequest struct {
	Symbol          string  `json:"symbol"`
	Price           float64 `json:"price,omitempty"`
	Vol             int     `json:"vol"`
	Leverage        int     `json:"leverage"`
	Side            int     `json:"side"`
	Type            int     `json:"type"`     // 1=limit, 5=market
	OpenType        int     `json:"openType"` // 1=cross, 2=isolated
	PositionMode    int     `json:"positionMode,omitempty"`
	StopLossPrice   float64 `json:"stopLossPrice,omitempty"`
	TakeProfitPrice float64 `json:"takeProfitPrice,omitempty"`
	ReduceOnly      bool    `json:"reduceOnly,omitempty"`
}
