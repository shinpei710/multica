# Projects and resources source map

Evidence layer for `SKILL.md`. Line numbers are current-tree anchors; re-derive
when code moves.

## Verification

```bash
go test ./internal/service -run TestProjectsAndResourcesSkillCoversDurableContext
go test ./internal/service -run TestBuiltinSkillsConformToTemplate
```

## Project tree, trash, and API

- `server/internal/handler/project.go:20-43` defines `ProjectResponse` with
  `parent_project_id`, `position`, `deleted_at`, `delete_expires_at`,
  `child_count`, and `resource_count`.
- `server/internal/handler/project.go:82-95` validates that a parent project is
  active and belongs to the same workspace.
- `server/internal/handler/project.go:97-117` checks descendants recursively so
  moving a project under its own child is rejected.
- `server/internal/handler/project.go:119-151` defines create/update request
  fields, including `parent_project_id` and `position`.
- `server/internal/handler/project.go:154-217` lists active projects and fills
  issue, resource, and child counts. The backing query filters
  `deleted_at IS NULL`.
- `server/internal/handler/project.go:566-589` handles project moves and clears:
  non-empty `parent_project_id` means move under that active parent; `null` or
  empty means top level; cycles are rejected.
- `server/internal/handler/project.go:609-668` soft-deletes a project tree. A
  non-empty tree or a tree with linked issues returns `409` with
  `code=project_not_empty`, `child_count`, and `issue_count` unless
  `confirm=true` is present.
- `server/internal/handler/project.go:670-685` lists deleted projects through
  the trash endpoint.
- `server/internal/handler/project.go:688-722` restores the deleted batch that
  contains the requested project.
- `server/internal/handler/project.go:841-849` keeps project search on active
  projects only (`p.deleted_at IS NULL`) while selecting the new tree/trash
  columns.
- `server/cmd/server/router.go:776-791` exposes `/api/projects`,
  `/api/projects/trash`, `/api/projects/{id}`, `/api/projects/{id}/restore`,
  and `/api/projects/{id}/resources` routes.

## SQL and sweeper behavior

- `server/pkg/db/queries/project.sql:1-7` lists active projects only and orders
  by parent, position, and creation time.
- `server/pkg/db/queries/project.sql:9-14` lists trash rows where
  `deleted_at IS NOT NULL`.
- `server/pkg/db/queries/project.sql:28-33` inserts `parent_project_id` and
  `position`.
- `server/pkg/db/queries/project.sql:36-51` updates parent/position, using
  `parent_project_id_set` so omitted fields do not move the project.
- `server/pkg/db/queries/project.sql:53-72` recursively soft-deletes the
  selected project and descendants, setting `deleted_at`, `deleted_by`,
  `delete_expires_at = now() + interval '30 days'`, and one
  `deleted_batch_id`.
- `server/pkg/db/queries/project.sql:74-92` restores every project sharing the
  deleted batch id.
- `server/pkg/db/queries/project.sql:98-101` hard-deletes expired trash rows.
- `server/pkg/db/queries/project.sql:107-134` counts child projects and linked
  issues in a project tree.
- `server/migrations/122_project_tree_trash_agent_kind_origin.up.sql:1-15` adds
  project hierarchy and trash columns/indexes.
- `server/cmd/server/runtime_sweeper.go:76-103` runs the trash sweeper and calls
  `DeleteExpiredProjects` every runtime sweeper tick.
- The migration keeps `parent_project_id` as `REFERENCES project(id) ON DELETE
  CASCADE`. The existing issue/project foreign key is left in place, so issue
  rows are not soft-deleted with the project; they keep `project_id` during the
  trash window and follow the existing database behavior once hard delete runs.

## CLI entry points

- `server/cmd/multica/cmd_project.go:125-133` registers `project list`, `get`,
  `create`, `update`, `delete`, `trash`, `restore`, and `status`.
- `server/cmd/multica/cmd_project.go:149-158` adds `project create --parent` and
  `--position`; `--repo` still bundles `github_repo` resources at creation.
- `server/cmd/multica/cmd_project.go:193-202` adds `project update --parent`,
  `--clear-parent`, and `--position`.
- `server/cmd/multica/cmd_project.go:204-211` adds `project delete --confirm`,
  `project trash`, and `project restore` flags.
- `server/cmd/multica/cmd_project.go:342-351` resolves the create parent project
  id/prefix and sends `parent_project_id` plus optional `position`.
- `server/cmd/multica/cmd_project.go:438-455` rejects `--parent` with
  `--clear-parent`, sends a new parent id, or sends `null` to move to top level.
- `server/cmd/multica/cmd_project.go:483-507` turns `409 project_not_empty`
  into a CLI message instructing the user to re-run with `--confirm`.
- `server/cmd/multica/cmd_project.go:510-545` lists project trash with deleted
  and expiry dates.
- `server/cmd/multica/cmd_project.go:548-573` restores a deleted project tree.
- `server/cmd/multica/cmd_project.go:1060-1091` computes table indentation from
  `parent_project_id`.
- `server/cmd/multica/cmd_project.go:1101-1124` resolves deleted project ids or
  prefixes by reading `/api/projects/trash`.

## Project resources

- `server/cmd/multica/cmd_project.go` registers `project resource
  list/add/update/remove`.
- `project create --repo` attaches `github_repo` resources during project
  creation.
- `project resource add` supports shortcuts for `github_repo` (`--url`,
  `--default-branch-hint`) and `local_directory` (`--local-path`, `--daemon-id`,
  `--ref-label`), or generic `--ref '<json>'`.
- `project resource update` merges shortcut edits with existing `resource_ref` so
  a partial edit does not clobber required fields.
- `server/pkg/db/queries/project_resource.sql` is the CRUD query surface for
  `project_resource` rows.
- Project resources are written into `.multica/project/resources.json` for agent
  workdirs by `server/internal/daemon/execenv/runtime_config.go`.
