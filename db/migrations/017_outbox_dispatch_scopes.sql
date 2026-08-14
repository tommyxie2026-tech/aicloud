-- S3D: global operational scheduling metadata for RLS-preserving Outbox dispatch.
--
-- The table intentionally contains Tenant/Project scope only. It must never
-- contain Task IDs, Outbox IDs, event payloads, model/tool/user data, or any
-- other tenant business record.

CREATE TABLE IF NOT EXISTS outbox_dispatch_scopes (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, project_id),
    CHECK (tenant_id <> ''),
    CHECK (project_id <> '')
);

REVOKE INSERT, UPDATE, DELETE, TRUNCATE ON TABLE outbox_dispatch_scopes FROM PUBLIC;

CREATE OR REPLACE FUNCTION aicloud_upsert_outbox_dispatch_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
BEGIN
    INSERT INTO public.outbox_dispatch_scopes (
        tenant_id,
        project_id,
        first_seen_at,
        last_seen_at
    ) VALUES (
        NEW.tenant_id,
        NEW.project_id,
        COALESCE(NEW.created_at, NOW()),
        COALESCE(NEW.created_at, NOW())
    )
    ON CONFLICT (tenant_id, project_id) DO UPDATE
       SET last_seen_at = GREATEST(
           public.outbox_dispatch_scopes.last_seen_at,
           EXCLUDED.last_seen_at
       );
    RETURN NEW;
END;
$$;

REVOKE ALL ON FUNCTION aicloud_upsert_outbox_dispatch_scope() FROM PUBLIC;

DROP TRIGGER IF EXISTS trg_aicloud_outbox_dispatch_scope ON outbox_messages;
CREATE TRIGGER trg_aicloud_outbox_dispatch_scope
AFTER INSERT ON outbox_messages
FOR EACH ROW
EXECUTE FUNCTION aicloud_upsert_outbox_dispatch_scope();

-- Backfill scheduling metadata for existing committed Outbox rows. This runs as
-- the migration owner; runtime dispatchers still receive no cross-tenant
-- payload-reading capability from this table.
INSERT INTO outbox_dispatch_scopes (
    tenant_id,
    project_id,
    first_seen_at,
    last_seen_at
)
SELECT
    tenant_id,
    project_id,
    MIN(created_at),
    MAX(created_at)
FROM outbox_messages
WHERE tenant_id <> '' AND project_id <> ''
GROUP BY tenant_id, project_id
ON CONFLICT (tenant_id, project_id) DO UPDATE
   SET first_seen_at = LEAST(
           outbox_dispatch_scopes.first_seen_at,
           EXCLUDED.first_seen_at
       ),
       last_seen_at = GREATEST(
           outbox_dispatch_scopes.last_seen_at,
           EXCLUDED.last_seen_at
       );
