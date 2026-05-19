# mexctrade — Design

**Status**: Draft
**Date**: 2026-05-20
**Author**: kntjspr

## Goal

A Go CLI that places MEXC futures trades with risk-managed position sizing, intended to be invoked by an openclaw trader-agent that's parsing trading signals (text + chart screenshots) from a Discord channel. Output is human-readable by default, JSON when `--json` is set so the agent can consume it.

## Position in the larger system

```
LARRY G's discord channel
       │
       ▼ (dcli listen, separate repo)
   trading-bridge (separate repo)
       │
       ▼ (openclaw agent --agent trader)
  trader-agent (config in ~/.openclaw/agents/trader)
       │
       ▼ (subprocess)
    mexctrade  ← THIS PROJECT
       │
       ▼ (REST, signed)
   MEXC futures API
```

mexctrade has no awareness of Discord, the agent, or the bridge. It is a pure trading executor: take inputs, place orders, report outcome.

## Non-goals (YAGNI)

- websocket streaming (price feeds, position updates)
- multi-account / multi-exchange abstraction
- trailing stops, breakeven moves, partial closes
- `modify` command (deferred until paper-trading proves the rest)
- daily-loss circuit breaker (user opted out; cross-margin makes this risky and the user accepted that)
- backtesting harness
- TUI

## Stack

- Go 1.25, stdlib http/crypto, `urfave/cli/v3` for commands, `pelletier/go-toml/v2` for config
- direct REST against `https://contract.mexc.com/api/v1/...` — no SDK wrapper

## Project layout

```
mexctrade/
├── go.mod
├── cmd/mexctrade/main.go              # entry, wires CLI to commands
├── internal/
│   ├── mexc/
│   │   ├── client.go                  # signed REST client (HMAC-SHA256)
│   │   ├── futures.go                 # endpoint methods
│   │   ├── symbols.go                 # contract detail cache
│   │   └── types.go                   # req/resp structs
│   ├── config/config.go               # load TOML, env overrides
│   ├── risk/sizing.go                 # position sizing math (pure functions)
│   ├── output/
│   │   ├── pretty.go
│   │   └── json.go
│   └── commands/
│       ├── portfolio.go
│       ├── positions.go
│       ├── orders.go
│       ├── place.go
│       ├── cancel.go
│       └── close.go
├── docs/superpowers/specs/2026-05-20-mexctrade-design.md
├── README.md
└── .gitignore
```

## Commands (v1)

| command | purpose |
|---|---|
| `mexctrade portfolio` | balance, available margin, unrealized PnL |
| `mexctrade positions` | currently open positions (symbol, side, size, entry, mark, uPnL) |
| `mexctrade orders` | pending limit orders (id, symbol, side, price, qty) |
| `mexctrade place --symbol X --side long\|short --entry market\|<price> --sl <price> --tp <price> --risk-pct 2` | open a new position |
| `mexctrade cancel --symbol X [--order-id ID]` | cancel pending limit orders on a symbol |
| `mexctrade close --symbol X` | market-close an open position |

Global flags: `--json`, `--dry-run`, `--verbose`, `--config <path>`.

## Symbol resolution

LARRY G's signals say `BTC`, `BILL`, `XAUT`, etc. MEXC futures uses `BTC_USDT`, `BILL_USDT`. mexctrade:

1. On first use, fetches `/api/v1/contract/detail` and caches all symbols to `~/.cache/mexctrade/contracts.json` (TTL 24h).
2. Accepts both short (`BTC`) and full (`BTC_USDT`). Short gets resolved by appending `_USDT` and looking up in the cache.
3. If the resolved symbol isn't found, exits with `UNKNOWN_SYMBOL` and suggests closest matches (Levenshtein distance ≤ 2).
4. Full-form input is never modified.

## Risk math

```
inputs:
  balance          : USDT available margin (from MEXC)
  risk_pct         : --risk-pct flag (default 2, max 5)
  entry            : market price OR user-supplied limit price
  sl               : --sl price
  max_leverage     : config, default 20

step 1 — compute risk and required notional:
  risk_amount      = balance × (risk_pct / 100)
  sl_distance_pct  = |entry - sl| / entry          # fractional, 0.01 = 1%
  notional_usdt    = risk_amount / sl_distance_pct

step 2 — run refusal checks (see next section). If any fire, exit 4. Never silently cap leverage to bring a refused trade into bounds — that violates the user's stated risk intent.

step 3 — compute final order params (only reached if refusal checks pass):
  raw_leverage     = ceil(notional_usdt / balance)
  leverage         = min(raw_leverage, symbol_max_leverage)
  contracts        = floor(notional_usdt / entry / contract_size)
```

## Refusal conditions

`mexctrade place` exits non-zero with `{"error": "...", "code": "RISK_..."}` (exit code 4) if any of:

- `--sl` not provided
- `risk_pct > 5` or `risk_pct <= 0`
- `sl_distance_pct < 0.001` (sub-0.1% SL — vision likely misread)
- `sl` on wrong side: long with `sl >= entry`, short with `sl <= entry`
- TP on wrong side: long with `tp <= entry`, short with `tp >= entry` (if `--tp` provided)
- `raw_leverage > max_leverage`
- symbol not found in contract cache
- `contracts == 0` (position too small to express in valid contract increments)

## MEXC API

| concern | choice |
|---|---|
| base URL | `https://contract.mexc.com` |
| auth | HMAC-SHA256. Headers: `ApiKey`, `Request-Time`, `Signature`. Signature = `HMAC_SHA256(secret, api_key + request_time + (sorted_query_string OR raw_body))`. Hex-encoded lower case. |
| time sync | `GET /api/v1/contract/ping` at startup. Compare `Date` header vs local. Abort if skew > 1s. |
| margin mode | cross (`openType=1`) |
| position mode | one-way (`positionMode=1`) |
| rate limits | 20 req / 2s. Single in-flight call per command — cannot breach. |

### Place order

```
POST /api/v1/private/order/submit
{
  "symbol":             "BTC_USDT",
  "side":               1=open long | 3=open short,
  "type":               1=limit | 5=market,
  "openType":           1,
  "positionMode":       1,
  "leverage":           <computed>,
  "vol":                <contracts>,
  "price":              <entry>  // only if type=1
  "stopLossPrice":      <sl>,
  "takeProfitPrice":    <tp>     // only if --tp provided
}
```

Before submit: `POST /api/v1/private/position/change_leverage` if requested leverage differs from current symbol setting.

### Cancel

`POST /api/v1/private/order/cancel_all` filtered by `symbol`, or `POST /api/v1/private/order/cancel` with `orderId` list if `--order-id` provided.

### Close

```
1. GET /api/v1/private/position/open_positions?symbol=BTC_USDT
2. submit reduce-only market order, opposite side, vol = position size:
   POST /api/v1/private/order/submit
   { symbol, side: 2 (close long) or 4 (close short), type: 5, vol, openType: 1 }
```

## Dry-run

`--dry-run` (or `dry_run = true` in config) skips order placement only. All reads happen against the live API:

- balance is real
- positions/orders show real state
- sizing math runs on real numbers

But `place`, `cancel`, `close` short-circuit before any state-mutating endpoint, and emit:

```json
{
  "dry_run": true,
  "would_call": "POST /api/v1/private/order/submit",
  "request_body": { ... full payload ... },
  "computed": { "leverage": 5, "contracts": 12, "notional_usdt": 247.5, ... }
}
```

## Output

Default pretty:

```
$ mexctrade portfolio
Balance:           $1,243.50 USDT
Available margin:  $987.20 USDT
Open positions:    3 (uPnL: +$24.10)
```

JSON mode (`--json`):

```json
{
  "balance_usdt":           1243.50,
  "available_margin_usdt":  987.20,
  "positions_count":        3,
  "unrealized_pnl_usdt":    24.10
}
```

Every command in `--json` mode emits a single JSON object on stdout. Errors in `--json` mode also emit JSON: `{"error": "...", "code": "...", "exit": N}`.

## Exit codes

| code | meaning |
|---|---|
| 0 | success |
| 1 | usage error (bad flags, missing args) |
| 2 | network / transient failure after retries |
| 3 | auth / config error (bad key, futures not enabled, clock skew, missing config) |
| 4 | refused — risk check failed OR MEXC business error (insufficient balance, leverage too high) |
| 5 | unknown symbol |
| 6 | unexpected / internal error |

## Config

Loaded from (in order, later wins):

1. `~/.config/mexctrade/config.toml` (default; mode 0600)
2. `--config <path>` override
3. env vars `MEXC_API_KEY`, `MEXC_API_SECRET`, `MEXCTRADE_DRY_RUN`, `MEXCTRADE_MAX_LEVERAGE`

```toml
api_key       = "..."
api_secret    = "..."
base_url      = "https://contract.mexc.com"
dry_run       = true                                    # safe default for v1
max_leverage  = 20
```

mexctrade refuses to start if `config.toml` exists with mode other than 0600.

## Errors → exit codes

| condition | exit |
|---|---|
| network timeout / 5xx (after 3 retries with backoff) | 2 |
| 429 (after retry_after-based retries up to 30s) | 2 |
| MEXC 401 / code 1002 / signature invalid | 3 |
| MEXC code 30005 (futures not authorized) | 3 |
| clock skew > 1s | 3 |
| MEXC code 600 (insufficient balance) | 4 |
| MEXC code 36101 (leverage exceeds limit) | 4 |
| any RISK_* refusal | 4 |
| unknown symbol | 5 |

## Testing

| file | scope |
|---|---|
| `internal/risk/sizing_test.go` | table-driven: long/short, market/limit, edge cases (huge SL, tiny SL, leverage cap, all refusal conditions) |
| `internal/mexc/client_test.go` | HMAC signing — feed MEXC's documented example into the signer, assert byte-identical output |
| `internal/mexc/futures_test.go` | `net/http/httptest`-backed stubs; assert request shape (headers, body, query) for each endpoint method |
| `internal/commands/*_test.go` | command exec with stubbed mexc.Client; assert stdout JSON shape + exit codes |
| `integration_test.go` (build tag `integration`) | optional manual smoke against the real key |

No live MEXC calls in CI.

## Open questions

None — all decided in brainstorming.
