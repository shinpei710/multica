DROP INDEX IF EXISTS idx_agent_origin;
DROP INDEX IF EXISTS idx_agent_runtime_blank_runtime;
DROP INDEX IF EXISTS agent_workspace_name_unique;
DELETE FROM agent WHERE kind = 'runtime_blank';
ALTER TABLE agent ADD CONSTRAINT agent_workspace_name_unique UNIQUE (workspace_id, name);
ALTER TABLE agent
    DROP COLUMN IF EXISTS origin_id,
    DROP COLUMN IF EXISTS origin_type,
    DROP COLUMN IF EXISTS kind;

DROP INDEX IF EXISTS idx_project_delete_expires;
DROP INDEX IF EXISTS idx_project_deleted;
DROP INDEX IF EXISTS idx_project_parent;
ALTER TABLE project
    DROP COLUMN IF EXISTS deleted_batch_id,
    DROP COLUMN IF EXISTS delete_expires_at,
    DROP COLUMN IF EXISTS deleted_by,
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS position,
    DROP COLUMN IF EXISTS parent_project_id;
