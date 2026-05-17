-- name: GetSnapshotStats :many
SELECT
    symbol,
    exchange,
    AVG(spread) AS avg_spread,
    AVG(depth_bid) AS avg_depth_bid,
    AVG(depth_ask) AS avg_depth_ask,
    ((AVG(depth_bid) + AVG(depth_ask)) / 2)::float AS avg_depth
FROM snapshots
WHERE time >= NOW() - @timeframe::interval
    AND exchange = ANY(@exchanges::text[])
    AND symbol = @symbol
GROUP BY symbol, exchange;

-- name: GetSlippageStats :many
SELECT
    symbol,
    exchange,
    size,
    AVG(slippage_bid) AS avg_slippage_bid,
    AVG(slippage_ask) AS avg_slippage_ask,
    ((AVG(slippage_bid) + AVG(slippage_ask)) / 2)::float AS avg_slippage
FROM slippage
WHERE time >= NOW() - @timeframe::interval
    AND exchange = ANY(@exchanges::text[])
    AND symbol = @symbol
GROUP BY symbol, exchange, size;
