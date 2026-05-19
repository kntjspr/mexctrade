package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:    "mexctrade",
		Usage:   "MEXC futures CLI for risk-managed trade execution",
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "path to config.toml (default ~/.config/mexctrade/config.toml)"},
			&cli.BoolFlag{Name: "json", Usage: "emit machine-readable JSON"},
			&cli.BoolFlag{Name: "dry-run", Usage: "skip state-mutating calls"},
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "debug logging"},
		},
		Commands: []*cli.Command{
			// commands will be wired in later tasks
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
