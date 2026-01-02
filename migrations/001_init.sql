CREATE TABLE IF NOT EXISTS pools (
  chain_id BIGINT NOT NULL,
  pool_address TEXT NOT NULL,
  token0 TEXT NOT NULL,
  token1 TEXT NOT NULL,
  fee INTEGER NOT NULL,
  tick_spacing INTEGER NOT NULL,
  first_seen_block BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, pool_address)
);

CREATE TABLE IF NOT EXISTS pool_window_metrics (
  chain_id BIGINT NOT NULL,
  pool_address TEXT NOT NULL,
  window_size_seconds INT NOT NULL,
  window_start_ts TIMESTAMPTZ NOT NULL,
  window_end_ts TIMESTAMPTZ NOT NULL,
  swap_count BIGINT NOT NULL,
  volume0 NUMERIC NOT NULL,
  volume1 NUMERIC NOT NULL,
  fee0 NUMERIC NOT NULL,
  fee1 NUMERIC NOT NULL,
  fee_usd NUMERIC NULL,
  fee_rate0 NUMERIC NULL,
  fee_rate1 NUMERIC NULL,
  tvl0 NUMERIC NULL,
  tvl1 NUMERIC NULL,
  tvl_usd NUMERIC NULL,
  apr NUMERIC NULL,
  fee_method TEXT NOT NULL,
  tvl_method TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (chain_id, pool_address, window_size_seconds, window_start_ts)
);

CREATE TABLE IF NOT EXISTS indexer_state (
  name TEXT PRIMARY KEY,
  last_processed_ts BIGINT NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
