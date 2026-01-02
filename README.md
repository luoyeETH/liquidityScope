# liquidityScope

liquidityScope is an open-source data pipeline for BSC Uniswap V3 / PancakeSwap V3 style pools. It ingests raw logs, decodes V3 pool events, and aggregates pool-level fee/TVL metrics into Postgres for research and analytics.

## Pipeline Overview

1. Step1: Pull raw logs to JSONL.
2. Step2: Decode Swap/Mint/Burn/Collect into typed events.
3. Step3: Aggregate into time windows and upsert metrics to Postgres.

## Key Features

- BSC log ingestion with batching, retry, checkpoint, and deterministic JSONL output.
- V3 pool event decoding (Swap/Mint/Burn/Collect) with pool metadata cache.
- Token metadata cache for decimals/symbol/name with safe fallbacks.
- Windowed metrics (volume, fees, TVL snapshots, fee rates, APR estimate).
- Idempotent Postgres upserts for incremental and recompute workflows.

## Requirements

- Go 1.22+
- BSC RPC endpoint (HTTP). Archive RPC is recommended for historical TVL accuracy.
- Postgres 14+ (local Docker example below).

## Build

```bash
go build ./cmd/indexer
```

## Quickstart

### Step1: Ingest Logs

```bash
./indexer run --rpc https://... --from 36000000 --to 36010000 \
  --address 0xPool1,0xPool2 \
  --topic0 0xSwapSig,0xMintSig \
  --batch-size 2000 \
  --out ./data/logs.jsonl \
  --checkpoint ./data/checkpoint.json
```

### Step2: Decode V3 Events

```bash
./indexer decode --rpc https://... --in ./data/logs.jsonl \
  --out ./data/typed_events.jsonl \
  --errors ./data/decode_errors.jsonl \
  --topic0-map 0xYourCollectSig=Collect \
  --include-live-meta=false
```

Notes:
- `topic0-map` allows mapping extra topic0 signatures to Swap/Mint/Burn/Collect for fork compatibility.
- `include-live-meta` attempts to read `slot0()` and `liquidity()` at the log block (archive RPC required for historical accuracy).
- Decode failures are appended to `decode_errors.jsonl`.

### Step3: Aggregate Windows

```bash
./indexer aggregate --rpc https://... \
  --in ./data/typed_events.jsonl \
  --window 5m \
  --pg-dsn "postgres://postgres:postgres@127.0.0.1:5432/liquidity?sslmode=disable" \
  --batch-size 1000 \
  --state-file ./data/aggregate_state.json
```

Notes:
- `--recompute-from` accepts unix seconds or RFC3339 (e.g. `1700000000` or `2024-01-01T00:00:00Z`).
- If `--state-file` is omitted, progress is stored in `indexer_state` (name `aggregator:<window_seconds>`).
- Fees are approximated from the fee tier and input-side amount (`fee_method=approx_from_feeTier`).

## Deployment (Local)

### Start Postgres (Docker)

```bash
docker run --name liquidity-pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=liquidity -p 5432:5432 -d postgres:16
```

### Apply Migrations

```bash
psql "postgres://postgres:postgres@127.0.0.1:5432/liquidity?sslmode=disable" -f ./migrations/001_init.sql
```

### Run the Pipeline

```bash
./indexer run ...
./indexer decode ...
./indexer aggregate ...
```

## Configuration

Configuration is loaded from, in order of precedence: flags, environment variables, then `config.yaml`. Use `--config` to point to a custom file.

Supported env vars (prefix `INDEXER_`):

- `INDEXER_RPC`
- `INDEXER_FROM`
- `INDEXER_TO`
- `INDEXER_ADDRESS` (comma-separated)
- `INDEXER_TOPIC0` (comma-separated)
- `INDEXER_BATCH_SIZE`
- `INDEXER_OUT`
- `INDEXER_CHECKPOINT`
- `INDEXER_CHECKPOINT_ENABLED`
- `INDEXER_MAX_RETRIES`
- `INDEXER_RETRY_BACKOFF` (e.g. `500ms`)
- `INDEXER_LOG_LEVEL` (debug/info/warn/error)
- `INDEXER_IN`
- `INDEXER_ERRORS`
- `INDEXER_TOPIC0_MAP` (comma-separated key=value)
- `INDEXER_INCLUDE_LIVE_META`
- `INDEXER_WINDOW`
- `INDEXER_PG_DSN`
- `INDEXER_STATE_FILE`
- `INDEXER_RECOMPUTE_FROM`

Example `config.yaml`:

```yaml
rpc: https://bsc-dataseed.binance.org
from: 36000000
to: 36010000
address:
  - 0x0000000000000000000000000000000000000000
topic0:
  - 0x0000000000000000000000000000000000000000000000000000000000000000
batch-size: 2000
out: ./data/logs.jsonl
checkpoint: ./data/checkpoint.json
checkpoint-enabled: true
max-retries: 5
retry-backoff: 500ms
log-level: info
```

## Output Schemas

### LogRecord (JSONL)

- `chain_id`
- `block_number`
- `block_hash`
- `tx_hash`
- `tx_index`
- `log_index`
- `address`
- `topics` (array of `0x` strings)
- `data` (`0x` hex)
- `removed`
- `timestamp`
- `ingested_at`

### TypedEvent (JSONL)

- `chain_id`
- `block_number`
- `block_hash`
- `tx_hash`
- `log_index`
- `address` (pool)
- `event_name` (Swap/Mint/Burn/Collect)
- `timestamp`
- `decoded` (event payload, big integers as strings)
- `pool_meta` (token0/token1/fee/tick_spacing)
- `raw` (topic0/data)

### Postgres Metrics

`pool_window_metrics` stores per-pool window aggregates:

- `window_start_ts` / `window_end_ts`
- `swap_count`, `volume0`, `volume1`
- `fee0`, `fee1`, `fee_usd`, `fee_rate0`, `fee_rate1`
- `tvl0`, `tvl1`, `tvl_usd`, `apr` (null unless a single-token APR estimate is available)
- `fee_method`, `tvl_method`

## Assumptions and Accuracy (v1)

- Fee uses a deterministic approximation from fee tier and input-side amount.
- TVL uses `balanceOf(pool)` at the last block of the window and falls back to latest if archive state is not available.
- USD fields are nullable until a decentralized price source is added.

## Roadmap

1. Precise fees using feeGrowthGlobal and tick feeGrowthOutside.
2. Accurate historical TVL with archive RPC (or state reconstruction).
3. On-chain price integration for USD conversions.
4. Tick-level liquidity attribution for LP analytics.
5. ClickHouse or TimescaleDB storage for large-scale backfills.

## Tests

```bash
go test ./...
```
