-- name: GetSlippageChart :many
SELECT 
    time_bucket(@interval::interval, time)::timestamptz AS bucket,
    symbol,
    exchange,
    size,
    AVG(slippage_bid) as avg_slippage_bid,
    AVG(slippage_ask) as avg_slippage_ask
FROM slippage
WHERE time >= NOW() - @timeframe::interval
    AND exchange = ANY(@exchanges::text[])
    AND symbol = @symbol
    AND size = ANY(@sizes::int[])
GROUP BY symbol, exchange, bucket, size
ORDER BY symbol, exchange, size, bucket ASC;

-- name: GetDepthChart :many
SELECT 
    time_bucket(@interval::interval, time)::timestamptz AS bucket,
    symbol,
    exchange,
    AVG(depth_bid) as avg_depth_bid,
    AVG(depth_ask) as avg_depth_ask
FROM snapshots
WHERE time >= NOW() - @timeframe::interval
    AND exchange = ANY(@exchanges::text[])
    AND symbol = @symbol
GROUP BY symbol, exchange, bucket
ORDER BY symbol, exchange, bucket ASC;