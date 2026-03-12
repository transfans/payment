-- name: GetBalance :one
SELECT * FROM balances
WHERE creator_id = $1;

-- name: UpsertBalance :one
INSERT INTO balances (creator_id, available, total_earned, total_paid)
VALUES ($1, $2, $2, 0)
ON CONFLICT (creator_id) DO UPDATE
SET available    = balances.available    + EXCLUDED.available,
    total_earned = balances.total_earned + EXCLUDED.total_earned
RETURNING *;

-- name: DecrementBalance :one
UPDATE balances
SET available  = available  - $2,
    total_paid = total_paid + $2
WHERE creator_id = $1
  AND available >= $2
RETURNING *;
