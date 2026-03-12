-- +goose Up
CREATE TABLE transactions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    fan_id      UUID        NOT NULL,
    creator_id  UUID        NOT NULL,
    tier_id     UUID        NOT NULL,
    amount      NUMERIC(10,2) NOT NULL,
    status      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE balances (
    creator_id   UUID          PRIMARY KEY,
    available    NUMERIC(10,2) NOT NULL DEFAULT 0,
    total_earned NUMERIC(10,2) NOT NULL DEFAULT 0,
    total_paid   NUMERIC(10,2) NOT NULL DEFAULT 0
);

CREATE TABLE payouts (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id   UUID          NOT NULL REFERENCES balances(creator_id),
    amount       NUMERIC(10,2) NOT NULL,
    status       TEXT          NOT NULL,
    completed_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE payouts;
DROP TABLE balances;
DROP TABLE transactions;
