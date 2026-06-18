import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { useWorkspaceId } from "../hooks";
import { workspaceKeys } from "../workspace/queries";
import type { QuickCreateAgentRequest } from "../types";

export function useQuickCreateAgent() {
  const qc = useQueryClient();
  const wsId = useWorkspaceId();
  return useMutation({
    mutationFn: (data: QuickCreateAgentRequest) => api.quickCreateAgent(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: workspaceKeys.agents(wsId) });
    },
  });
}
