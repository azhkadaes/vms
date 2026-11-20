-- +migrate Up
-- Limiter buckets (D29-D33), keyed by a keyed digest, never a raw IP or account.
CREATE TABLE support.rate_limit_buckets (
    bucket_key CHAR(64) PRIMARY KEY CHECK (bucket_key ~ '^[0-9a-f]{64}$'),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
GRANT SELECT, INSERT, UPDATE, DELETE ON support.rate_limit_buckets TO vms_api;

-- +migrate Down
REVOKE SELECT, INSERT, UPDATE, DELETE ON support.rate_limit_buckets FROM vms_api;
DROP TABLE support.rate_limit_buckets;
