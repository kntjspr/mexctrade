package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kntjspr/mexctrade/internal/config"
	"github.com/kntjspr/mexctrade/internal/mexc"
)

// MexcAPI is the subset of mexc.Client that command code depends on.
// Defining the interface in the consumer package lets tests inject stubs
// without importing httptest into command tests.
type MexcAPI interface {
	Ping(ctx context.Context) (int64, error)
	GetAssets(ctx context.Context) ([]mexc.Asset, error)
	GetOpenPositions(ctx context.Context, symbol string) ([]mexc.Position, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]mexc.Order, error)
	ChangeLeverage(ctx context.Context, symbol string, leverage, openType int) error
	PlaceOrder(ctx context.Context, req mexc.PlaceOrderRequest) (string, error)
	CancelAll(ctx context.Context, symbol string) error
	CancelOrders(ctx context.Context, orderIDs []string) error
}

// Resolver is the subset of mexc.Resolver commands depend on.
type Resolver interface {
	Resolve(input string) (*mexc.Contract, error)
}

// Ctx is the runtime context passed to every command.
type Ctx struct {
	Cfg      *config.Config
	API      MexcAPI
	Resolver Resolver
	Stdout   io.Writer
	Stderr   io.Writer
	JSON     bool
	DryRun   bool
}

// ExitCode is returned by every command. main.go translates to os.Exit.
type ExitCode int

const (
	ExitOK       ExitCode = 0
	ExitUsage    ExitCode = 1
	ExitNetwork  ExitCode = 2
	ExitAuth     ExitCode = 3
	ExitRefused  ExitCode = 4
	ExitUnknown  ExitCode = 5
	ExitInternal ExitCode = 6
)

// BuildCtx is the production constructor. CLI calls this once.
// Performs clock-skew check against the MEXC server.
func BuildCtx(cfgPath string, jsonOut, dryRun bool) (*Ctx, error) {
	if cfgPath == "" {
		cfgPath = os.ExpandEnv("$HOME/.config/mexctrade/config.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	if dryRun {
		cfg.DryRun = true
	}
	client := mexc.New(cfg.BaseURL, cfg.APIKey, cfg.APISecret)
	resolver := mexc.NewResolver(client)

	srvMs, err := client.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ping MEXC: %w", err)
	}
	localMs := time.Now().UnixMilli()
	if abs64(srvMs-localMs) > 1000 {
		return nil, fmt.Errorf("clock skew %dms vs MEXC server, fix NTP", srvMs-localMs)
	}

	return &Ctx{
		Cfg:      cfg,
		API:      client,
		Resolver: resolver,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		JSON:     jsonOut,
		DryRun:   cfg.DryRun,
	}, nil
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
