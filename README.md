# mexctrade

MEXC futures CLI for risk-managed trade execution. Intended to be invoked by an upstream agent that parses trading signals.

## Install

    go install github.com/kntjspr/mexctrade/cmd/mexctrade@latest

## Configure

    mkdir -p ~/.config/mexctrade
    cat > ~/.config/mexctrade/config.toml <<EOF
    api_key       = "..."
    api_secret    = "..."
    dry_run       = true
    max_leverage  = 20
    EOF
    chmod 600 ~/.config/mexctrade/config.toml

## Commands

    mexctrade portfolio
    mexctrade positions
    mexctrade orders
    mexctrade place --symbol BTC --side long --entry market --sl 105000 --tp 112000 --risk-pct 2
    mexctrade cancel --symbol LAB
    mexctrade close --symbol BTC

Global flags: `--json`, `--dry-run`, `--verbose`, `--config`.

See `docs/superpowers/specs/2026-05-20-mexctrade-design.md` for the full design.
