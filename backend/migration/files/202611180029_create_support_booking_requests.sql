-- +migrate Up
CREATE TABLE support.booking_requests (
    audience VARCHAR(8) NOT NULL CHECK (audience IN ('public', 'staff')),
    owner_digest CHAR(64) NOT NULL CHECK (owner_digest ~ '^[0-9a-f]{64}$'),
    reference UUID NOT NULL,
    payload_digest CHAR(64) CHECK (payload_digest ~ '^[0-9a-f]{64}$'),
    email_digest CHAR(64) CHECK (email_digest ~ '^[0-9a-f]{64}$'),
    state VARCHAR(9) NOT NULL CHECK (state IN ('committed', 'closed')),
    id_kunjungan INTEGER REFERENCES vms.kunjungan(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    PRIMARY KEY (audience, owner_digest, reference),
    CHECK ((state = 'committed' AND id_kunjungan IS NOT NULL AND payload_digest IS NOT NULL)
        OR (state = 'closed' AND id_kunjungan IS NULL AND payload_digest IS NULL)),
    CHECK ((audience = 'public') = (email_digest IS NOT NULL))
);
CREATE INDEX booking_requests_visit_idx ON support.booking_requests(id_kunjungan);
GRANT SELECT, INSERT ON support.booking_requests TO vms_api;

-- +migrate Down
REVOKE SELECT, INSERT ON support.booking_requests FROM vms_api;
DROP TABLE support.booking_requests;
