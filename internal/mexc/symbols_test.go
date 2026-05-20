package mexc

import (
	goerrs "errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func newResolver(t *testing.T, contracts []Contract) *Resolver {
	t.Helper()
	dir := t.TempDir()
	r := &Resolver{
		cachePath: filepath.Join(dir, "contracts.json"),
		ttl:       24 * time.Hour,
		fetch: func() ([]Contract, error) {
			return contracts, nil
		},
	}
	return r
}

func TestResolveShortForm(t *testing.T) {
	r := newResolver(t, []Contract{
		{Symbol: "BTC_USDT", ContractSize: 0.0001, MaxLeverage: 100},
		{Symbol: "ETH_USDT", ContractSize: 0.01, MaxLeverage: 75},
	})
	c, err := r.Resolve("BTC")
	if err != nil {
		t.Fatalf("Resolve BTC: %v", err)
	}
	if c.Symbol != "BTC_USDT" {
		t.Errorf("got %s want BTC_USDT", c.Symbol)
	}
}

func TestResolveFullForm(t *testing.T) {
	r := newResolver(t, []Contract{{Symbol: "BTC_USDT", ContractSize: 0.0001, MaxLeverage: 100}})
	c, err := r.Resolve("BTC_USDT")
	if err != nil {
		t.Fatalf("Resolve full: %v", err)
	}
	if c.Symbol != "BTC_USDT" {
		t.Errorf("got %s", c.Symbol)
	}
}

func TestResolveCaseInsensitive(t *testing.T) {
	r := newResolver(t, []Contract{{Symbol: "BTC_USDT"}})
	c, err := r.Resolve("btc")
	if err != nil || c.Symbol != "BTC_USDT" {
		t.Fatalf("case insensitive failed: %v %v", err, c)
	}
}

func TestResolveUnknownReturnsSuggestion(t *testing.T) {
	r := newResolver(t, []Contract{
		{Symbol: "BTC_USDT"},
		{Symbol: "LAB_USDT"},
	})
	_, err := r.Resolve("LBA") // typo of LAB
	if err == nil {
		t.Fatalf("expected error")
	}
	var unkErr *UnknownSymbolError
	if !errorsAs(err, &unkErr) {
		t.Fatalf("expected UnknownSymbolError, got %T: %v", err, err)
	}
	if len(unkErr.Suggestions) == 0 || unkErr.Suggestions[0] != "LAB_USDT" {
		t.Errorf("suggestions wrong: %v", unkErr.Suggestions)
	}
}

func TestCacheHitSkipsFetch(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "contracts.json")
	// Pre-seed cache
	os.WriteFile(cachePath, []byte(`{"fetchedAt":`+nowTS()+`,"contracts":[{"symbol":"BTC_USDT","contractSize":0.0001,"maxLeverage":100,"priceScale":1,"volScale":0,"state":0}]}`), 0o644)
	fetchCalled := false
	r := &Resolver{
		cachePath: cachePath,
		ttl:       24 * time.Hour,
		fetch: func() ([]Contract, error) {
			fetchCalled = true
			return nil, nil
		},
	}
	c, err := r.Resolve("BTC")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Symbol != "BTC_USDT" {
		t.Errorf("got %s", c.Symbol)
	}
	if fetchCalled {
		t.Errorf("cache hit should not fetch")
	}
}

// Test helpers.
func errorsAs(err error, target any) bool { return goerrs.As(err, target) }
func nowTS() string                       { return strconv.FormatInt(time.Now().Unix(), 10) }
