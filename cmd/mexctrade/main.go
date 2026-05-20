package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/kntjspr/mexctrade/internal/commands"
)

func main() {
	app := &cli.Command{
		Name:    "mexctrade",
		Usage:   "MEXC futures CLI for risk-managed trade execution",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "path to config.toml"},
			&cli.BoolFlag{Name: "json"},
			&cli.BoolFlag{Name: "dry-run"},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}},
		},
		Commands: []*cli.Command{
			{
				Name:  "portfolio",
				Usage: "show balance, available margin, open position count, uPnL",
				Action: func(ctx context.Context, c *cli.Command) error {
					return runCmd(ctx, c, commands.Portfolio)
				},
			},
			{
				Name:  "positions",
				Usage: "list currently open positions",
				Action: func(ctx context.Context, c *cli.Command) error {
					return runCmd(ctx, c, commands.Positions)
				},
			},
			{
				Name:  "orders",
				Usage: "list pending limit orders",
				Action: func(ctx context.Context, c *cli.Command) error {
					return runCmd(ctx, c, commands.Orders)
				},
			},
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCmd(ctx context.Context, c *cli.Command, fn func(context.Context, *commands.Ctx) commands.ExitCode) error {
	cmdCtx, err := commands.BuildCtx(
		c.Root().String("config"),
		c.Root().Bool("json"),
		c.Root().Bool("dry-run"),
	)
	if err != nil {
		return err
	}
	code := fn(ctx, cmdCtx)
	if code != commands.ExitOK {
		os.Exit(int(code))
	}
	return nil
}
