---
name: multica-projects-and-resources
description: "Use when creating, inspecting, updating, or debugging Multica projects and project resources. Covers durable project context, github_repo and local_directory resources, how resources affect future agent task context, when to bind repos, and when not to mutate resources."
user-invocable: false
allowed-tools: Bash(multica *)
---

# Multica Projects and Resources

## Quick start

Projects are durable context containers. Projects can be nested into an
unlimited-depth tree with `parent_project_id`; resources attached to any project
can affect future agent tasks for work in that project.

```bash
multica project list --output json
multica project get <project-id> --output json
multica project trash --output json
multica project resource list <project-id> --output json
```

Project resources are mutated through project resource commands/endpoints. Issue
comments do not create durable project resources.

## Core model

A project groups work and carries durable resources. A project may have a parent
project and child projects; the tree relationship is structural metadata, while
resources remain attached to one project. A resource is not just display
metadata; it is context later injected into task briefs and
`.multica/project/resources.json`.

`GET /api/projects` and `multica project list` show only active projects by
default. Each project response includes `parent_project_id`, `position`,
`child_count`, `deleted_at`, and `delete_expires_at`. Deleted projects live in
trash for 30 days and are listed through `GET /api/projects/trash` /
`multica project trash`.

Common resource types:

- `github_repo` — durable GitHub repo context, with `resource_ref.url` and optional `default_branch_hint`;
- `local_directory` — daemon-local path context, with `resource_ref.local_path`, `daemon_id`, and optional label.

## CLI

```bash
multica project list --output json
multica project get <project-id> --output json
multica project create --title "<title>" --parent <parent-project-id> --repo <github-url> --output json
multica project update <project-id> --title "<title>" --parent <new-parent-id> --position 10 --output json
multica project update <project-id> --clear-parent --output json
multica project status <project-id> in_progress --output json
multica project delete <project-id> --confirm
multica project trash --output json
multica project restore <project-id> --output json
multica project resource list <project-id> --output json
multica project resource add <project-id> --type github_repo --url <github-url> --output json
multica project resource add <project-id> --type local_directory --local-path <abs-path> --daemon-id <daemon-id> --output json
multica project resource update <project-id> <resource-id> --url <new-github-url> --output json
multica project resource remove <project-id> <resource-id> --output json
```

Use `--ref '<json>'` only for resource types or payloads not covered by shortcuts.
Use `--parent` to create/move a child project; use `--clear-parent` to move a
project back to the top level. The server rejects moves that would make a
project its own descendant.

## Delete and restore

Project deletion is a 30-day soft delete. A delete request against a non-empty
project tree returns HTTP `409` with `code=project_not_empty`, `child_count`, and
`issue_count` unless the caller confirms. In the CLI, re-run the delete with
`--confirm` after the user has explicitly approved deleting the whole tree:

```bash
multica project delete <project-id> --confirm
```

Confirmed deletion moves the selected project and all descendant projects to
trash as one batch. Issues are not deleted. During the 30-day trash window, issue
rows keep their `project_id`; restoring the project batch restores the original
tree and issues appear back under their projects automatically. After
`delete_expires_at`, the server sweeper hard-deletes the trashed projects; the
existing issue foreign key behavior clears affected `issue.project_id` values.

Use:

```bash
multica project trash --output json
multica project restore <deleted-project-id> --output json
```

`project restore` restores the deleted batch, so restoring any project from a
deleted tree restores the tree that was deleted together.

## When to add a resource

Add/update a project resource when the user asks for durable project context: "把这个 GitHub repo 绑到项目上", "以后都用这个 repo", "agent 总是拿不到这个项目的仓库", or "这个项目要在我的本地目录里跑".

Project resources are durable and affect future tasks. `multica repo checkout`
is task-local checkout state.

## Debugging wrong context

1. `multica project get <project-id> --output json`.
2. Check whether `parent_project_id` points at the expected container.
3. `multica project resource list <project-id> --output json`.
4. Check `github_repo.resource_ref.url`, `default_branch_hint`, and `local_directory.resource_ref.daemon_id`.
5. Updating resources or moving projects is a durable mutation. After an update,
   listing the project/resource is the verification path.
6. If resources match the expected task context, inspect runtime/repo checkout
   path next.

## Side effects

Project create/update/delete/status/restore and project resource
add/update/remove mutate durable workspace state and affect future tasks. Moving
a project changes the project tree for every user. Deleting a project is soft for
30 days but can still hide an entire project tree, so require explicit user
confirmation before passing `--confirm`. Ask before changing `local_directory`
unless the user explicitly requested that exact local path.

More source-backed details: `references/projects-and-resources-source-map.md`.
