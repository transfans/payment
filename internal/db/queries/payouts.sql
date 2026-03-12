-- name: InsertPayout :one
INSERT INTO payouts (creator_id, amount, status, completed_at)
VALUES ($1, $2, 'completed', now())
RETURNING *;

-- name: ListPayoutsByCreator :many
SELECT * FROM payouts
WHERE creator_id = $1
ORDER BY completed_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPayoutsByCreator :one
SELECT COUNT(*) FROM payouts
WHERE creator_id = $1;
