# Runtimes and repos source map

Evidence layer for `SKILL.md`. Line numbers are current-tree anchors; re-derive
when code moves.

## Verification

```bash
go test ./internal/service -run TestRuntimesAndReposSkillCoversClaimAndCheckoutChain
go test ./internal/service -run TestBuiltinSkillsConformToTemplate
```

## Runtime CLI and routes

- `server/cmd/multica/cmd_runtime.go` registers `runtime list`, `usage`,
  `activity`, `update`, and `delete`.
- `runtime list` reads `/api/runtimes` and prints `id`, `name`, `runtime_mode`,
  `provider`, `status`, and `last_seen_at`.
- `runtime update` posts to `/api/runtimes/{runtime-id}/update`; with `--wait`
  it polls update status.
- `runtime delete` deletes `/api/runtimes/{runtime-id}`; with `--cascade`, it
  first reads the `runtime_has_active_agents` conflict payload and posts those
  ids to `/api/runtimes/{runtime-id}/archive-agents-and-delete`.
- `server/cmd/multica/cmd_repo.go` registers `repo checkout <url> [--ref]`.
- `repo checkout` requires `MULTICA_DAEMON_PORT`, sends `workspace_id`,
  `workdir`, `ref`, `agent_name`, and `task_id` to local daemon
  `/repo/checkout`, then prints the checked-out path.
- `server/cmd/server/router.go` registers daemon APIs under `/api/daemon`,
  including workspace repos and task claim.

## Runtime blank agents

- `server/internal/handler/runtime.go:73-89` calls `UpsertRuntimeBlankAgent` and
  publishes an agent event whenever a runtime registration/update needs its
  blank agent ensured.
- `server/internal/handler/runtime.go:91-96` maps runtime visibility to agent
  visibility: public runtime becomes workspace-visible agent; all other runtime
  visibility values become private agent visibility.
- `server/pkg/db/queries/agent.sql:702-724` defines `GetRuntimeBlankAgentByRuntime`
  and `UpsertRuntimeBlankAgent`. The upsert writes empty instructions, env,
  args, MCP, model, and thinking level, and sets `kind='runtime_blank'`.
- `server/migrations/122_project_tree_trash_agent_kind_origin.up.sql:17-36` adds
  `agent.kind`, restricts it to `configured` or `runtime_blank`, and creates the
  unique runtime-blank index on `runtime_id`.
- `server/internal/handler/agent.go:34-72` exposes `kind`, `origin_type`, and
  `origin_id` on agent resources.
- `server/internal/handler/agent.go:1152-1158` rejects generic agent writes for
  `kind=runtime_blank`; update, env, and skill handlers call this guard.
- `server/internal/service/task.go:696-741` requires a runtime blank agent for
  `EnqueueQuickCreateAgentTask` and queues the task on that blank agent's
  `agent_id` and `runtime_id`.

## Daemon claim and task execution

- `server/internal/handler/daemon.go:1158-1160` documents that task claim returns
  the agent's current name and skills from the DB.
- `server/internal/handler/daemon.go:1720-1727` maps `quick_create_agent` context
  onto the daemon task response fields used by the daemon prompt and env.
- `server/internal/handler/daemon.go:1788-1834` mints task tokens. Standard
  tasks use the runtime owner; quick-create-agent tasks use the requester from
  task context so AI-created agents are owned by the requesting human.
- `server/internal/handler/daemon.go:1737-1764` enforces workspace isolation for
  issue, chat, autopilot, quick-create, and quick-create-agent tasks before
  minting the task token.
- `server/internal/daemon/types.go:77-83` carries quick-create and
  quick-create-agent fields in the daemon task type.
- `server/internal/daemon/prompt.go:17-32` routes quick-create-agent tasks to the
  agent-design prompt before normal quick-create or issue prompts.
- `server/internal/daemon/prompt.go:41-76` tells the runtime blank agent to run
  exactly one `multica agent create --output json` command for the selected
  runtime.
- `server/internal/daemon/daemon.go:3344-3353` injects task-scoped Multica env
  (`MULTICA_TOKEN`, server URL, daemon port, workspace id, agent id, task id).
- `server/internal/daemon/daemon.go:3374-3376` injects
  `MULTICA_QUICK_CREATE_AGENT_TASK_ID` for quick-create-agent tasks.
- `server/internal/daemon/execenv/runtime_config.go` injects task/project/repo
  context into agent workdirs, including quick-create-like task variants.
- `server/internal/daemon/daemon.go` claims tasks, prepares workdirs, launches
  provider CLIs, and reports completion.

## Repos and project context

- The runtime brief lists repos available to the task. Workspace repo metadata
  comes from workspace data; `github_repo` and `local_directory` project
  resources are durable project context.
- `local_directory` resources include a daemon id and local path, so they carry
  local-machine assumptions.
- Project resources are written into `.multica/project/resources.json` for agent
  workdirs by `server/internal/daemon/execenv/runtime_config.go`.
