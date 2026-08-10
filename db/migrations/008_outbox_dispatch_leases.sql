-- R6 Outbox dispatcher leasing.
--
-- Delivery remains at-least-once. These fields let a scoped dispatcher claim
-- work safely, recover abandoned delivering rows after a lease expires, and
-- retain bounded-retry/dead-letter evidence without mutating TaskEvent history.

ALTER TABLE outbox_messages ADD COLUMN IF NOT EXISTS lease_owner TEXT;
ALTER TABLE outbox_messages ADD COLUMN IF NOT EXISTS lease_expires_at TIMESTAMPTZ;
ALTER TABLE outbox_messages ADD COLUMN IF NOT EXISTS last_error TEXT;

ALTER TABLE outbox_messages DROP CONSTRAINT IF EXISTS outbox_lease_state_check;
ALTER TABLE outbox_messages ADD CONSTRAINT outbox_lease_state_check CHECK (
    (status = 'delivering' AND lease_owner IS NOT NULL AND lease_owner <> '' AND lease_expires_at IS NOT NULL)
    OR
    (status <> 'delivering' AND lease_owner IS NULL AND lease_expires_at IS NULL)
);

CREATE INDEX IF NOT EXISTS outbox_dispatch_lease_idx
    ON outbox_messages(status, available_at, lease_expires_at, created_at, outbox_id);
