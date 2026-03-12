-- name: InsertTransaction :one
INSERT INTO transactions (fan_id, creator_id, tier_id, amount, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListTransactionsByFan :many
SELECT * FROM transactions
WHERE fan_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountTransactionsByFan :one
SELECT COUNT(*) FROM transactions
WHERE fan_id = $1;
