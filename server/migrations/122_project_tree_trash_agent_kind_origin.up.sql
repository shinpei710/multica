-- Project hierarchy and trash lifecycle.
ALTER TABLE project
    ADD COLUMN IF NOT EXISTS parent_project_id UUID REFERENCES project(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS position DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_by UUID REFERENCES "user"(id),
    ADD COLUMN IF NOT EXISTS delete_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_batch_id UUID;

CREATE INDEX IF NOT EXISTS idx_project_parent
    ON project(workspace_id, parent_project_id, position, created_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_project_deleted
    ON project(workspace_id, deleted_at DESC)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_project_delete_expires
    ON project(delete_expires_at)
    WHERE deleted_at IS NOT NULL;

-- Runtime-backed blank agents and agent quick-create provenance.
ALTER TABLE agent
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'configured'
        CHECK (kind IN ('configured', 'runtime_blank')),
    ADD COLUMN IF NOT EXISTS origin_type TEXT
        CHECK (origin_type IN ('quick_create_agent')),
    ADD COLUMN IF NOT EXISTS origin_id UUID;

ALTER TABLE agent DROP CONSTRAINT IF EXISTS agent_workspace_name_unique;
DROP INDEX IF EXISTS agent_workspace_name_unique;
CREATE UNIQUE INDEX IF NOT EXISTS agent_workspace_name_unique
    ON agent(workspace_id, name)
    WHERE kind = 'configured';

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_runtime_blank_runtime
    ON agent(runtime_id)
    WHERE kind = 'runtime_blank' AND runtime_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_agent_origin
    ON agent(origin_type, origin_id)
    WHERE origin_type IS NOT NULL;
