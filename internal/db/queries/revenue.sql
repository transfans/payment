-- name: GetRevenue :one
SELECT COALESCE(SUM(amount), 0)::NUMERIC(10,2) AS total
FROM transactions
WHERE creator_id = $1
  AND ($2::TIMESTAMPTZ IS NULL OR created_at >= $2)
  AND ($3::TIMESTAMPTZ IS NULL OR created_at <= $3);

-- name: GetRevenueByTier :many
SELECT
    tier_id,
    COUNT(*)::INT               AS count,
    SUM(amount)::NUMERIC(10,2)  AS amount
FROM transactions
WHERE creator_id = $1
  AND ($2::TIMESTAMPTZ IS NULL OR created_at >= $2)
  AND ($3::TIMESTAMPTZ IS NULL OR created_at <= $3)
GROUP BY tier_id;
