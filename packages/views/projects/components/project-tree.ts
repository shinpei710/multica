import type { Project } from "@multica/core/types";

export interface ProjectTreeRow {
  project: Project;
  depth: number;
}

export function defaultProjectTreeCompare(a: Project, b: Project): number {
  return (
    (a.position ?? 0) - (b.position ?? 0) ||
    a.title.localeCompare(b.title) ||
    Date.parse(a.created_at) - Date.parse(b.created_at)
  );
}

export function flattenProjectTree(
  projects: Project[],
  compare: (a: Project, b: Project) => number = defaultProjectTreeCompare,
  includeIds?: ReadonlySet<string>,
): ProjectTreeRow[] {
  const included = includeIds ?? new Set(projects.map((project) => project.id));
  const byId = new Map<string, Project>();
  for (const project of projects) {
    if (included.has(project.id)) byId.set(project.id, project);
  }

  const childrenByParent = new Map<string | null, Project[]>();
  for (const project of byId.values()) {
    const parentId = project.parent_project_id;
    const key = parentId && byId.has(parentId) ? parentId : null;
    const children = childrenByParent.get(key) ?? [];
    children.push(project);
    childrenByParent.set(key, children);
  }
  for (const children of childrenByParent.values()) {
    children.sort(compare);
  }

  const rows: ProjectTreeRow[] = [];
  const visited = new Set<string>();
  const walk = (project: Project, depth: number) => {
    if (visited.has(project.id)) return;
    visited.add(project.id);
    rows.push({ project, depth });
    for (const child of childrenByParent.get(project.id) ?? []) {
      walk(child, depth + 1);
    }
  };

  for (const root of childrenByParent.get(null) ?? []) {
    walk(root, 0);
  }

  for (const project of [...byId.values()].sort(compare)) {
    if (!visited.has(project.id)) walk(project, 0);
  }

  return rows;
}

export function addProjectAncestors(
  projects: Project[],
  ids: Iterable<string>,
): Set<string> {
  const byId = new Map(projects.map((project) => [project.id, project]));
  const out = new Set(ids);
  for (const id of [...out]) {
    let current = byId.get(id);
    const seen = new Set<string>();
    while (current?.parent_project_id && !seen.has(current.parent_project_id)) {
      seen.add(current.parent_project_id);
      const parent = byId.get(current.parent_project_id);
      if (!parent) break;
      out.add(parent.id);
      current = parent;
    }
  }
  return out;
}
