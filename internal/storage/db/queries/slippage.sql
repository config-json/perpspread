-- name: SetSlippages :copyfrom
INSERT INTO slippage (time, exchange, symbol, size, slippage_bid, slippage_ask)
VALUES ($1, $2, $3, $4, $5, $6);