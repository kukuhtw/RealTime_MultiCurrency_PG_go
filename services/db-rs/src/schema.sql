-- accounts
CREATE TABLE IF NOT EXISTS wallet_accounts (
  account_id   VARCHAR PRIMARY KEY,
  balance_idr  BIGINT NOT NULL DEFAULT 0,
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- reservations (pending debit sementara)
-- reservations (pending debit sementara)
CREATE TABLE IF NOT EXISTS reservations (
  reservation_id    UUID PRIMARY KEY,
  idempotency_key   TEXT NOT NULL UNIQUE,
  sender_id         VARCHAR NOT NULL,
  receiver_id       VARCHAR NOT NULL,
  amount_idr        BIGINT NOT NULL CHECK (amount_idr > 0),
  currency_input    TEXT NOT NULL, -- <== tambahan
  status            TEXT NOT NULL CHECK (status IN ('PENDING','COMMITTED','ROLLEDBACK')),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- payments log (final)
-- payments log (final)
CREATE TABLE IF NOT EXISTS payments (
  payment_id       UUID PRIMARY KEY,
  idempotency_key  TEXT NOT NULL UNIQUE,
  sender_id        VARCHAR NOT NULL,
  receiver_id      VARCHAR NOT NULL,
  currency_input   TEXT NOT NULL,
  amount_idr       BIGINT NOT NULL,
  status           TEXT NOT NULL CHECK (status IN ('SUCCESS','FAILED')),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- payments log (final)
CREATE TABLE IF NOT EXISTS payments (
  payment_id       UUID PRIMARY KEY,
  idempotency_key  TEXT NOT NULL UNIQUE,
  sender_id        VARCHAR NOT NULL,
  receiver_id      VARCHAR NOT NULL,
  currency_input   TEXT NOT NULL,
  amount_idr       BIGINT NOT NULL,
  status           TEXT NOT NULL CHECK (status IN ('SUCCESS','FAILED')),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
