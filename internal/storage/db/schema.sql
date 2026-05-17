CREATE TABLE snapshots (
    time TIMESTAMPTZ NOT NULL,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    spread NUMERIC NOT NULL,
    depth_bid NUMERIC NOT NULL,
    depth_ask NUMERIC NOT NULL,
    PRIMARY KEY (symbol, exchange, time)
);

CREATE TABLE slippage (
    time TIMESTAMPTZ NOT NULL,
    exchange TEXT NOT NULL,
    symbol TEXT NOT NULL,
    size NUMERIC NOT NULL,
    slippage_bid NUMERIC NOT NULL,
    slippage_ask NUMERIC NOT NULL,
    PRIMARY KEY (symbol, exchange, size, time)
);

SELECT create_hypertable('snapshots', 'time');
SELECT create_hypertable('slippage', 'time');

ALTER TABLE snapshots SET (
    TIMESCALEDB.COMPRESS,
    TIMESCALEDB.COMPRESS_SEGMENTBY = 'symbol, exchange',
    TIMESCALEDB.COMPRESS_ORDERBY = 'time DESC'
);

ALTER TABLE slippage SET (
    TIMESCALEDB.COMPRESS,
    TIMESCALEDB.COMPRESS_SEGMENTBY = 'symbol, exchange, size',
    TIMESCALEDB.COMPRESS_ORDERBY = 'time DESC'
);

SELECT add_compression_policy('snapshots', INTERVAL '7 days');
SELECT add_compression_policy('slippage', INTERVAL '7 days');

SELECT add_retention_policy('snapshots', INTERVAL '90 days');
SELECT add_retention_policy('slippage', INTERVAL '90 days');