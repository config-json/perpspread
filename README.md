# Perpspread
Golang service that aggregates order book data across perp exchanges and exposes depth and slippage analytics via REST API.
## Getting Started
1. Make sure you're running a Postgres database.
2. Copy `.env.example` to `.env` and customize it with your own values.
3. Run `sqlc generate`. This will add go functions for all database operations. If you don't have `sqlc` installed, run `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` first.
4. Run `go run ./cmd/api` and `go run ./cmd/reader` in parallel to start both services.
## Architecture
![[docs/architecture.png]]
### Reader
![[docs/reader.png]]
The reader service ingests order book data from multiple exchanges, normalizes it, computes derived metrics, and writes results to the database.
#### Base Reader
- Owns the common lifecycle for all exchange readers
- Supports two connection models: one WS per market or one WS for all markets with a sub/unsub protocol. The base type handles both via a shared interface.
#### Exchange Readers
- Override `baseReader` and implement protocol-specific logic: message parsing, symbol formatting and delta application against a local snapshot.
- Emit a normalized order book struct 
#### Multiplexer
- Runs one goroutine for each reader
- Aggregates all readers' outputs into a single buffered channel (cap 1000)
- Owns context cancellation for coordinated shutdown of all readers
#### Processor
- Takes normalized order book snapshots from the multiplexer's output
- Computes derived data (depth/slippage)
- Stateless per snapshot: each input produces one processed output independently
#### Limitations
- Hyperliquid: only best 20 bids/asks are returned from the public WSS. To access the full order book data, a non-validating node is needed.
### Database
- TimescaleDB hypertables for snapshots and slippage, partitioned by time and keyed on (symbol, exchange)
- 7-day compression, 90-day retention
- Throttler writes the latest snapshot per (symbol, exchange) every 15s rather than every update. Decoupled from the reader so high-frequency exchange updates don't pressure the DB.
## REST API
Base URL: `http://localhost:8000`
### `GET /ping`
Health check. Returns 200
### `GET /info/exchanges`
Returns the list of supported exchanges. Use these values for the `exchange` query parameter on other endpoints.
### `GET /info/markets`
Returns the list of supported markets. Use these values for the `:symbol` route parameter.
### `GET /stats/:symbol`
Returns aggregated order book stats for a specific market.
#### Query params
- `period` (required) - e.g., `5m`, `1h`, `1D`, `1W`, `1M`, `3M`
- `exchange` (optional, defaults to all)
### `GET /chart/slippage/:symbol`
Returns series of slippage snapshots over a certain period of time for a specific market.
#### Query params
- `period` (required)
- `exchange` (optional, defaults to all)
### `GET /chart/depth/:symbol`
Returns series of order book depth snapshots over a certain period of time for a specific market.
#### Query params
`period` (required)
`exchange` (optional, defaults to all)