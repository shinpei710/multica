import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { projectKeys } from "./queries";
import { useWorkspaceId } from "../hooks";
import { useRecentContextStore } from "../chat/recent-context-store";
import type { Project, CreateProjectRequest, UpdateProjectRequest, ListProjectsResponse } from "../types";

export function useCreateProject() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (data: CreateProjectRequest) => api.createProject(data),
    onSuccess: (newProject) => {
      qc.setQueryData<Project>(projectKeys.detail(wsId, newProject.id), newProject);
      qc.setQueryData<ListProjectsResponse>(projectKeys.list(wsId), (old) =>
        old && !old.projects.some((p) => p.id === newProject.id)
          ? { ...old, projects: [...old.projects, newProject], total: old.total + 1 }
          : old,
      );
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: projectKeys.list(wsId) });
      qc.invalidateQueries({ queryKey: projectKeys.trash(wsId) });
    },
  });
}

export function useUpdateProject() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdateProjectRequest) =>
      api.updateProject(id, data),
    onMutate: ({ id, ...data }) => {
      qc.cancelQueries({ queryKey: projectKeys.list(wsId) });
      const prevList = qc.getQueryData<ListProjectsResponse>(projectKeys.list(wsId));
      const prevDetail = qc.getQueryData<Project>(projectKeys.detail(wsId, id));
      qc.setQueryData<ListProjectsResponse>(projectKeys.list(wsId), (old) =>
        old ? { ...old, projects: old.projects.map((p) => (p.id === id ? { ...p, ...data } : p)) } : old,
      );
      qc.setQueryData<Project>(projectKeys.detail(wsId, id), (old) =>
        old ? { ...old, ...data } : old,
      );
      return { prevList, prevDetail, id };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prevList) qc.setQueryData(projectKeys.list(wsId), ctx.prevList);
      if (ctx?.prevDetail) qc.setQueryData(projectKeys.detail(wsId, ctx.id), ctx.prevDetail);
    },
    onSettled: (_data, _err, vars) => {
      qc.invalidateQueries({ queryKey: projectKeys.detail(wsId, vars.id) });
      qc.invalidateQueries({ queryKey: projectKeys.list(wsId) });
    },
  });
}

export function useDeleteProject() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: ({ id, confirm = false }: { id: string; confirm?: boolean }) =>
      api.deleteProject(id, confirm),
    onMutate: async ({ id, confirm }) => {
      if (!confirm) return { prevList: undefined, id, optimistic: false };
      await qc.cancelQueries({ queryKey: projectKeys.list(wsId) });
      const prevList = qc.getQueryData<ListProjectsResponse>(projectKeys.list(wsId));
      qc.setQueryData<ListProjectsResponse>(projectKeys.list(wsId), (old) => {
        if (!old) return old;
        const removed = collectProjectTreeIds(old.projects, id);
        return {
          ...old,
          projects: old.projects.filter((p) => !removed.has(p.id)),
          total: Math.max(0, old.total - removed.size),
        };
      });
      qc.removeQueries({ queryKey: projectKeys.detail(wsId, id) });
      return { prevList, id, optimistic: true };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prevList) qc.setQueryData(projectKeys.list(wsId), ctx.prevList);
    },
    onSuccess: (_data, vars) => {
      useRecentContextStore.getState().forgetContext(wsId, { type: "project", id: vars.id });
    },
    onSettled: (_data, _err, vars) => {
      qc.invalidateQueries({ queryKey: projectKeys.list(wsId) });
      qc.invalidateQueries({ queryKey: projectKeys.trash(wsId) });
      qc.invalidateQueries({ queryKey: projectKeys.detail(wsId, vars.id) });
    },
  });
}

export function useRestoreProject() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (id: string) => api.restoreProject(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: projectKeys.list(wsId) });
      qc.invalidateQueries({ queryKey: projectKeys.trash(wsId) });
    },
  });
}

function collectProjectTreeIds(projects: Project[], rootId: string): Set<string> {
  const ids = new Set<string>([rootId]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const project of projects) {
      if (project.parent_project_id && ids.has(project.parent_project_id) && !ids.has(project.id)) {
        ids.add(project.id);
        changed = true;
      }
    }
  }
  return ids;
}
