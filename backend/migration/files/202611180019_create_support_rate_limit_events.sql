-- +migrate Up
-- Admitted events per bucket; removing a bucket removes its history.
CREATE TABLE support.rate_limit_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    bucket_key CHAR(64) NOT NULL REFERENCES support.rate_limit_buckets(bucket_key) ON DELETE CASCADE,
    occurred_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX rate_limit_events_bucket_time_idx ON support.rate_limit_events(bucket_key, occurred_at);
GRANT SELECT, INSERT, UPDATE, DELETE ON support.rate_limit_events TO vms_api;
GRANT USAGE ON SEQUENCE support.rate_limit_events_id_seq TO vms_api;

-- +migrate Down
REVOKE USAGE ON SEQUENCE support.rate_limit_events_id_seq FROM vms_api;
REVOKE SELECT, INSERT, UPDATE, DELETE ON support.rate_limit_events FROM vms_api;
DROP TABLE support.rate_limit_events;
