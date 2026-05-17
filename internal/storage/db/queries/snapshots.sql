-- name: SetSnapshot :exec
INSERT INTO snapshots (time, exchange, symbol, spread, depth_bid, depth_ask)
VALUES ($1, $2, $3, $4, $5, $6);