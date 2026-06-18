-- name: ListProjects :many
SELECT * FROM project
WHERE workspace_id = $1
  AND deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::text IS NULL OR priority = sqlc.narg('priority'))
ORDER BY COALESCE(parent_project_id, '00000000-0000-0000-0000-000000000000'::uuid), position ASC, created_at DESC;

-- name: ListDeletedProjects :many
SELECT * FROM project
WHERE workspace_id = $1
  AND deleted_at IS NOT NULL
ORDER BY deleted_at DESC, title ASC;

-- name: GetProject :one
SELECT * FROM project
WHERE id = $1;

-- name: GetProjectInWorkspace :one
SELECT * FROM project
WHERE id = $1 AND workspace_id = $2;

-- name: GetActiveProjectInWorkspace :one
SELECT * FROM project
WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL;

-- name: CreateProject :one
INSERT INTO project (
    workspace_id, title, description, icon, status,
    lead_type, lead_id, priority, parent_project_id, position
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, sqlc.narg('parent_project_id'), COALESCE(sqlc.narg('position')::double precision, 0)
) RETURNING *;

-- name: UpdateProject :one
UPDATE project SET
    title = COALESCE(sqlc.narg('title'), title),
    description = sqlc.narg('description'),
    icon = sqlc.narg('icon'),
    status = COALESCE(sqlc.narg('status'), status),
    priority = COALESCE(sqlc.narg('priority'), priority),
    lead_type = sqlc.narg('lead_type'),
    lead_id = sqlc.narg('lead_id'),
    parent_project_id = CASE
        WHEN @parent_project_id_set::boolean THEN @parent_project_id::uuid
        ELSE parent_project_id
    END,
    position = COALESCE(sqlc.narg('position')::double precision, position),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SoftDeleteProjectTree :many
WITH RECURSIVE tree AS (
    SELECT p.id FROM project p WHERE p.id = $1 AND p.workspace_id = $2
    UNION ALL
    SELECT child.id
    FROM project child
    JOIN tree parent ON child.parent_project_id = parent.id
    WHERE child.workspace_id = $2
), batch AS (
    SELECT COALESCE(sqlc.narg('deleted_batch_id')::uuid, gen_random_uuid()) AS id
)
UPDATE project p
SET deleted_at = now(),
    deleted_by = $3,
    delete_expires_at = now() + INTERVAL '30 days',
    deleted_batch_id = batch.id,
    updated_at = now()
FROM tree, batch
WHERE p.id = tree.id
  AND p.deleted_at IS NULL
RETURNING p.*;

-- name: RestoreProjectTree :many
WITH target AS (
    SELECT p.deleted_batch_id
    FROM project p
    WHERE p.id = $1 AND p.workspace_id = $2 AND p.deleted_at IS NOT NULL
), tree AS (
    SELECT p.id
    FROM project p, target t
    WHERE p.workspace_id = $2 AND p.deleted_batch_id = t.deleted_batch_id
)
UPDATE project p
SET deleted_at = NULL,
    deleted_by = NULL,
    delete_expires_at = NULL,
    deleted_batch_id = NULL,
    updated_at = now()
FROM tree
WHERE p.id = tree.id
RETURNING p.*;

-- name: DeleteProject :exec
DELETE FROM project
WHERE id = $1 AND workspace_id = $2;

-- name: DeleteExpiredProjects :execrows
WITH RECURSIVE descendants(root_id, id) AS (
    SELECT p.id, child.id
    FROM project p
    JOIN project child ON child.parent_project_id = p.id
    WHERE p.deleted_at IS NOT NULL
      AND p.delete_expires_at < now()
    UNION ALL
    SELECT d.root_id, child.id
    FROM descendants d
    JOIN project child ON child.parent_project_id = d.id
)
DELETE FROM project p
WHERE p.deleted_at IS NOT NULL
  AND p.delete_expires_at < now()
  AND NOT EXISTS (
      SELECT 1
      FROM descendants d
      JOIN project child ON child.id = d.id
      WHERE d.root_id = p.id
        AND (child.deleted_at IS NULL OR child.delete_expires_at IS NULL OR child.delete_expires_at >= now())
  );

-- name: CountIssuesByProject :one
SELECT count(*) FROM issue
WHERE project_id = $1;

-- name: CountChildProjectsByProject :one
SELECT count(*) FROM project
WHERE parent_project_id = $1 AND deleted_at IS NULL;

-- name: CountIssuesInProjectTree :one
WITH RECURSIVE tree AS (
    SELECT p.id FROM project p WHERE p.id = $1
    UNION ALL
    SELECT child.id
    FROM project child
    JOIN tree parent ON child.parent_project_id = parent.id
)
SELECT count(*) FROM issue
WHERE project_id IN (SELECT id FROM tree);

-- name: GetProjectIssueStats :many
SELECT project_id,
       count(*)::bigint AS total_count,
       count(*) FILTER (WHERE status IN ('done', 'cancelled'))::bigint AS done_count
FROM issue
WHERE project_id = ANY(sqlc.arg('project_ids')::uuid[])
GROUP BY project_id;

-- name: GetProjectChildCounts :many
SELECT parent_project_id AS project_id, count(*)::bigint AS child_count
FROM project
WHERE parent_project_id = ANY(sqlc.arg('project_ids')::uuid[])
  AND deleted_at IS NULL
GROUP BY parent_project_id;
