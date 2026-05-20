package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kntjspr/mexctrade/internal/mexc"
)

type stubAPICancel struct {
	stubAPI
	cancelCalls int
	lastSymbol  string
}

func (s *stubAPICancel) CancelAll(_ context.Context, symbol string) error {
	s.cancelCalls++
	s.lastSymbol = symbol
	return nil
}

func TestCancelLive(t *testing.T) {
	api := &stubAPICancel{}
	out := &bytes.Buffer{}
	c := &Ctx{
		API:      api,
		Resolver: &stubResolver{c: &mexc.Contract{Symbol: "LAB_USDT"}},
		Stdout:   out, Stderr: &bytes.Buffer{},
		JSON: true,
	}
	code := Cancel(context.Background(), c, CancelArgs{Symbol: "LAB"})
	if code != ExitOK {
		t.Fatalf("exit %d, body: %s", code, out.String())
	}
	if api.cancelCalls != 1 || api.lastSymbol != "LAB_USDT" {
		t.Errorf("cancel call wrong: %d %s", api.cancelCalls, api.lastSymbol)
	}
}

func TestCancelDryRun(t *testing.T) {
	api := &stubAPICancel{}
	out := &bytes.Buffer{}
	c := &Ctx{
		API:      api,
		Resolver: &stubResolver{c: &mexc.Contract{Symbol: "LAB_USDT"}},
		Stdout:   out, Stderr: &bytes.Buffer{},
		JSON:     true,
		DryRun:   true,
	}
	code := Cancel(context.Background(), c, CancelArgs{Symbol: "LAB"})
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if api.cancelCalls != 0 {
		t.Errorf("dry-run should not call API")
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["dry_run"] != true || result["symbol"] != "LAB_USDT" {
		t.Errorf("output wrong: %+v", result)
	}
}
