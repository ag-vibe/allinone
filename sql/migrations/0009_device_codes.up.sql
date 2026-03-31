BEGIN;

CREATE TABLE device_codes (
    id                 BIGSERIAL PRIMARY KEY,
    device_code_hash   BYTEA       NOT NULL UNIQUE,
    user_code_hash     BYTEA       NOT NULL UNIQUE,
    client_id          TEXT        NOT NULL,
    scope              TEXT,
    status             TEXT        NOT NULL DEFAULT 'pending',
    user_id            INTEGER,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at         TIMESTAMPTZ NOT NULL,
    last_poll_at       TIMESTAMPTZ,
    poll_interval_sec  INTEGER     NOT NULL DEFAULT 5,
    poll_count         INTEGER     NOT NULL DEFAULT 0,
    ip                 TEXT,
    user_agent         TEXT,

    CONSTRAINT device_codes_status_check
      CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'consumed'))
);

CREATE INDEX device_codes_expires_at_idx ON device_codes(expires_at);
CREATE INDEX device_codes_status_idx ON device_codes(status);
CREATE INDEX device_codes_user_id_idx ON device_codes(user_id);

COMMIT;
