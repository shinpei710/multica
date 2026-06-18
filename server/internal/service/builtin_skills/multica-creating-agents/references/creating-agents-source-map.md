# Creating agents — source map

Evidence layer for `SKILL.md`. Every contract maps to `file:line` on the
current tree (branch `feat/builtin-skills`, latest `main` merged), the runtime
effect, and a safe read-only check. Line numbers were re-derived against this
tree — re-derive again if the files move, the surrounding context (not the
number) is the anchor.

## Verification

```bash
# Conformance eval for this skill (and the shared template invariants):
go test ./internal/service -run TestCreatingAgentsSkillCoversAgentCreationContracts
go test ./internal/service -run TestBuiltinSkillsConformToTemplate
```

## CLI entry points — `server/cmd/multica/cmd_agent.go`

| Contract | Line | Behavior | Safe check |
|---|---|---|---|
| Create flags: `name`, `description`, `instructions`, `instructions-stdin`, `instructions-file`, `runtime-id` | 159–164 | Registered create flags; `instructions-file` / `instructions-stdin` provide safe multi-line input; `name`/`runtime-id` enforced in `runAgentCreate` | `multica agent create --help` |
| `runtime-config`, `model`, `thinking-level`, `custom-args` flags | 165–168 | `model` help: "Prefer this over passing --model in --custom-args"; `thinking-level` is a thin pass-through (server validates the provider enum, empty = runtime default); `custom-args` help names codex/openclaw rejecting `--model` (CLI help only, not server-enforced) | `multica agent create --help` |
| Secret-safe env input: `custom-env`, `custom-env-stdin`, `custom-env-file` | 169–171 | `--custom-env` warns about shell history / `ps`; stdin and file modes keep secrets off the command line; mutually exclusive | `multica agent create --help` |
| Secret-safe MCP input: `mcp-config`, `mcp-config-stdin`, `mcp-config-file` (create) | 172–174 | Same three-channel pattern as `custom-env`; `--mcp-config` warns about shell history / `ps`; value must be a JSON object or `null` | `multica agent create --help` |
| MCP flags on `agent update` | 197–199 | Same three channels on update; `--mcp-config null` clears. Unlike `custom_env`, `mcp_config` IS settable via update | `multica agent update --help` |
| `thinking-level` flag on `agent update` | 187 | New reasoning/effort level; thin pass-through; `--thinking-level ""` clears to runtime default (mirrors `--model`) | `multica agent update --help` |
| `runAgentCreate` builds body + `POST /api/agents` | 419 | Only sets a body key when the flag `Changed`; posts to `/api/agents` (line 499) | read 419–500 |
| Body assembly: description/instructions/runtime-config/custom-args/custom-env/mcp-config/model/thinking-level | 438–491 | `resolveInstructions` gates inline/stdin/file instruction input; `resolveCustomEnv` and `resolveMcpConfig` gate their secret channels; `model` and `thinking_level` are `Changed`-gated pass-throughs; omitted flags are not sent | read 438–491 |
| Quick-create-agent origin stamping | 490–493 | When `MULTICA_QUICK_CREATE_AGENT_TASK_ID` exists, CLI adds `origin_type=quick_create_agent` and `origin_id=<task-id>` before posting to `/api/agents` | read 490–493 |
| `runAgentUpdate` sends `thinking_level` / `mcp_config` | 508 | `thinking_level` added when `--thinking-level` is `Changed` (556); `resolveMcpConfig` adds `mcp_config` (570); `PUT /api/agents/{id}` at 584; `custom_env` is intentionally not a flag here | read 508–585 |
| `resolveInstructions` helper | around 1010 | Mutually exclusive inline/stdin/file instruction input; stdin/file reject empty content so generated multi-line instructions can avoid shell quoting | read helper |
| `parseMcpConfig` / `resolveMcpConfig` helpers | around 1130 | Validator (object-or-`null`, content-free errors) + three-channel resolver, mirroring `parseCustomEnv`/`resolveCustomEnv` | read helpers |
| `agent skills set` = replace-all | 792 | `PUT /api/agents/{id}/skills` (810); `--skill-ids ''` clears all (798–799) | `multica agent skills set --help` |
| `agent skills add` = additive | 817 | `POST /api/agents/{id}/skills/add` (838); requires ≥1 id (823–828) | `multica agent skills add --help` |
| `agent skills list` | 760 | reads bindings, no side effect | `multica agent skills list --help` |
| `agent env get` | 894 | `GET /api/agents/{id}/env` | `multica agent env get --help` |
| `agent env set` | 929 | `PUT /api/agents/{id}/env` with full `custom_env` map (935, 949) | `multica agent env set --help` |

Note: the CLI no longer exposes `--from-template`. The agent-template backend
still exists (registry `server/internal/agenttmpl/`, handler `agent_template.go`,
routes `GET /api/agent-templates` and `POST /api/agents/from-template`, plus the
`packages/core` client/query wrappers) but is currently orphaned plumbing with no
live caller: the removed CLI flag was its only non-test consumer, and onboarding
does NOT use it — `packages/views/onboarding/steps/step-agent.tsx` builds four
hardcoded local presets (i18n-resolved) and creates via plain `POST /api/agents`
(`createAgent`), never `POST /api/agents/from-template`. Do not treat the template
API as a supported agent-creation path. This skill teaches manual `agent create`
only.

## Create handler — `server/internal/handler/agent.go`

| Contract | Line | Behavior |
|---|---|---|
| `maxAgentDescriptionLength = 255` | 31 | Cap is 255 **Unicode code points** (comment: counted via `utf8.RuneCountInString`, matches Postgres `char_length`) |
| `AgentResponse` omits plaintext `custom_env` and exposes agent provenance | 34–72 | Exposes only `has_custom_env` (53) and `custom_env_key_count` (54); also returns `kind` (64), `origin_type` (65), and `origin_id` (66) |
| `CreateAgentRequest` fields | 682–704 | `description`, `instructions`, `runtime_config`, `custom_env`, `custom_args`, `model`, `thinking_level`, `origin_type`, and `origin_id` (plus name/avatar/visibility/mcp_config/max_concurrent_tasks) |
| `name` required | 742–745 | 400 "name is required" |
| `description` ≤ 255 code points | 746–749 | `utf8.RuneCountInString(req.Description) > maxAgentDescriptionLength` → 400 |
| `runtime_id` required | 750–753 | `if req.RuntimeID == ""` → 400 "runtime_id is required" |
| `visibility` default | 754–756 | `if req.Visibility == "" { req.Visibility = "private" }` — access-control field, not the runtime prompt |
| `max_concurrent_tasks` default | 757–759 | `if req.MaxConcurrentTasks == 0 { req.MaxConcurrentTasks = 6 }` — scheduler cap |
| `runtime_id` must resolve in workspace | 770–777 | parsed + `GetAgentRuntimeForWorkspace`; unknown → 400 "invalid runtime_id" |
| Runtime access gate | 779–786 | private runtimes can only be used by their owner or a workspace admin |
| `thinking_level` provider-level validation | 788–795 | `!agent.IsKnownThinkingValue(runtime.Provider, req.ThinkingLevel)` → 400; per-model gaps deferred to daemon |
| `origin_type` / `origin_id` validation | 797–835 | only accepts `origin_type=quick_create_agent`; requires `origin_id`; actor must be a task-scoped agent; `origin_id` must equal current `X-Task-ID`; task agent must match `X-Agent-ID`; task context type must be `quick_create_agent` |
| Defaults: `{}` config/env, `[]` args | 847–860 | `RuntimeConfig`→`{}`, `CustomEnv`→`{}`, `CustomArgs`→`[]` when nil, before insert |
| `mcp_config` null-skip on create | 862–865 | raw JSON copied through unless the body value is the literal `null` |
| `CreateAgent` insert params | 867–886 | persists runtime_config, instructions, custom_env, custom_args, model, thinking_level, mcp_config, visibility, max_concurrent_tasks, origin_type, and origin_id |
| `mcp_config` redacted on read | 55, 660–676 | `redactMcpConfig` sets `McpConfigRedacted=true`; reads redact for agent actors, workspace always-redact, and users lacking secret access |
| Runtime blank agent write guard | 1152–1158 | `kind=runtime_blank` returns 400 for managed agent writes; update, env, and skill handlers call this guard |
| `UpdateAgent` rejects `custom_env` | 1184–1187 | if `custom_env` present in body → 400 "use PUT /api/agents/{id}/env (or `multica agent env set`)" |
| `UpdateAgent` persists / clears `mcp_config` | 1218–1222, 1334–1335 | Tri-state from the raw body: key omitted → no change; literal `null` → `ClearAgentMcpConfig`; object → replace. No 400 like `custom_env` — `mcp_config` IS updatable here |
| `description` ≤ 255 on update too | 1195–1198 | same cap re-checked on update |
| `QuickCreateAgentRequest` and response | 924–934 | request fields are `prompt`, `runtime_id`, `visibility`, `model`, and `thinking_level`; response is accepted `task_id` |
| `QuickCreateAgent` handler | 936–1019 | validates prompt/runtime/visibility/runtime access/runtime online/quick-create-agent daemon support/thinking level, ensures the runtime blank agent, and enqueues a quick-create-agent task |
| Quick-create-agent daemon gate | `server/pkg/agent/version.go`, `server/internal/handler/issue.go` | `MinQuickCreateAgentCLIVersion` is stricter than the issue quick-create minimum because this flow requires quick-create-agent task payloads, `MULTICA_QUICK_CREATE_AGENT_TASK_ID`, and `--instructions-file` |

## Runtime blank and AI-created-agent flow

| Contract | Line | Behavior |
|---|---|---|
| Runtime blank agent response fields | `server/internal/handler/runtime.go:73–89` | `ensureRuntimeBlankAgent` upserts the blank agent and publishes an agent event |
| Runtime visibility mapping | `server/internal/handler/runtime.go:91–96` | runtime `public` maps to agent `workspace`; other values map to `private` |
| Runtime blank upsert SQL | `server/pkg/db/queries/agent.sql:702–724` | unique per runtime; empty instructions/env/args/MCP/model/thinking; `kind='runtime_blank'` |
| Quick-create-agent task context | `server/internal/service/task.go:608–624` | context stores prompt, requester/workspace, selected runtime, visibility, model, and thinking level |
| Enqueue quick-create-agent task | `server/internal/service/task.go:696–741` | requires `kind=runtime_blank`; queues with blank agent id and runtime id; wakes daemon |
| Daemon prompt dispatch | `server/internal/daemon/prompt.go:17–32` | quick-create-agent prompt wins before normal quick-create / issue prompts |
| Daemon agent-design prompt | `server/internal/daemon/prompt.go:41–76` | instructs exactly one `multica agent create --output json`, uses `--instructions-file` for generated multi-line instructions, passes selected runtime/visibility/model/thinking, and forbids retry |
| Env marker injection | `server/internal/daemon/daemon.go:3374–3376` | injects `MULTICA_QUICK_CREATE_AGENT_TASK_ID=<task-id>` into the task process |
| Completion lookup | `server/internal/service/task.go:2504–2554` | success path finds the new agent by `origin_type=quick_create_agent` and `origin_id=<task-id>`, then writes `agent_create_done` inbox |
| Failure notification | `server/internal/service/task.go:2556–2593` | failure/missing agent path writes `agent_create_failed` inbox |

## Env endpoint — `server/internal/handler/agent_env.go`

| Contract | Line | Behavior |
|---|---|---|
| `authorizeAgentEnv` gate | 66 | loads agent, then applies the two checks below |
| Agent actors denied | 80–84 | `if actorType == "agent"` → 403 "agents may not access env management endpoints" (MUL-2600 impersonation guard) |
| Owner/admin only | 86 | `requireWorkspaceRole(..., "owner", "admin")` |

## Routes — `server/cmd/server/router.go`

| Contract | Line | Behavior |
|---|---|---|
| `GET /env` | 603 | `h.GetAgentEnv` (plaintext read, gated) |
| `PUT /env` | 604 | `h.UpdateAgentEnv` (full-map overwrite, gated) |

## Claim-time injection — `server/internal/handler/daemon.go`

| Contract | Line | Behavior |
|---|---|---|
| Fresh agent re-read on claim | 1109–1111 | `GetAgent(task.AgentID)` — claim uses persisted fields, not create output |
| Workspace skills FIRST | 1115 | `skills := h.TaskService.LoadAgentSkills(...)` |
| Built-ins appended | 1116 | `skills = append(skills, h.TaskService.BuiltinSkills()...)` |
| Runtime payload | 1130–1143 | `TaskAgentData` carries `Instructions`, `Skills`, `CustomEnv`, `CustomArgs`, `Model`, `ThinkingLevel`, `McpConfig` (1130–1131, 1140) — confirms these are runtime-consumed; `description`, `visibility`, and `max_concurrent_tasks` are absent (not runtime-prompt fields) |

## Skill loading — `server/internal/service/task.go`

| Contract | Line | Behavior |
|---|---|---|
| `LoadAgentSkills` | 1685 | `ListAgentSkills` + per-skill `ListSkillFiles` → content + supporting files for execution |

## Built-in skills — `server/internal/service/builtin_skills.go`

| Contract | Line | Behavior |
|---|---|---|
| `go:embed builtin_skills` | 10–11 | skills embedded at compile time |
| `loadBuiltinSkill` | 45 | reads `<name>/SKILL.md` (47) + walks sibling files into `Files` (56–68) |

## Persisted columns — `server/pkg/db/generated/agent.sql.go`

| Contract | Line | Behavior |
|---|---|---|
| `CreateAgent` INSERT | generated file | columns include `runtime_config, runtime_id, instructions, custom_env, custom_args, mcp_config, model, thinking_level, origin_type, origin_id` |
| `CreateAgentParams` | generated file | typed params include `RuntimeConfig []byte`, `Instructions string`, `CustomEnv []byte`, `CustomArgs []byte`, `Model pgtype.Text`, `ThinkingLevel pgtype.Text`, `OriginType pgtype.Text`, `OriginID pgtype.UUID` |
| `UpdateAgent` SET | 2552–2566 | COALESCE updates of `runtime_config, instructions, custom_env, custom_args, model, thinking_level` — note `custom_env` is COALESCE-guarded but the handler rejects it before this query runs |
| `UpdateAgentCustomEnv` (called by the `UpdateAgentEnv` handler) | 2652 | `SET custom_env = $2` — the only write path for env values |
