"use client";

import { useMemo, useState } from "react";
import { Bot, Loader2, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { useQuery } from "@tanstack/react-query";
import { useQuickCreateAgent } from "@multica/core/agents";
import { ApiError } from "@multica/core/api";
import { useAuthStore } from "@multica/core/auth";
import { useWorkspaceId } from "@multica/core/hooks";
import { runtimeListOptions } from "@multica/core/runtimes";
import { memberListOptions } from "@multica/core/workspace/queries";
import type { AgentVisibility } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import { Input } from "@multica/ui/components/ui/input";
import { Label } from "@multica/ui/components/ui/label";
import { Textarea } from "@multica/ui/components/ui/textarea";
import { RuntimePicker } from "./runtime-picker";
import { useT } from "../../i18n";

const MIN_QUICK_CREATE_AGENT_CLI_VERSION = "0.2.22";

export function QuickCreateAgentDialog({ onClose }: { onClose: () => void }) {
  const { t } = useT("agents");
  const wsId = useWorkspaceId();
  const currentUserId = useAuthStore((s) => s.user?.id ?? null);
  const { data: runtimes = [], isLoading: runtimesLoading } = useQuery(
    runtimeListOptions(wsId),
  );
  const { data: members = [] } = useQuery(memberListOptions(wsId));
  const quickCreate = useQuickCreateAgent();
  const [selectedRuntimeId, setSelectedRuntimeId] = useState("");
  const [visibility, setVisibility] = useState<AgentVisibility>("workspace");
  const [model, setModel] = useState("");
  const [thinkingLevel, setThinkingLevel] = useState("");
  const [prompt, setPrompt] = useState("");

  const selectedRuntime = useMemo(
    () => runtimes.find((runtime) => runtime.id === selectedRuntimeId) ?? null,
    [runtimes, selectedRuntimeId],
  );

  const submit = async () => {
    const trimmedPrompt = prompt.trim();
    if (!trimmedPrompt || !selectedRuntime) return;
    try {
      await quickCreate.mutateAsync({
        prompt: trimmedPrompt,
        runtime_id: selectedRuntime.id,
        visibility,
        model: model.trim() || undefined,
        thinking_level: thinkingLevel.trim() || undefined,
      });
      toast.success(t(($) => $.quick_create.toast_sent));
      onClose();
    } catch (err) {
      toast.error(formatQuickCreateAgentError(err, t));
    }
  };

  return (
    <Dialog open onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="h-4 w-4" />
            {t(($) => $.quick_create.title)}
          </DialogTitle>
          <DialogDescription>{t(($) => $.quick_create.description)}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="quick-create-agent-prompt" className="text-xs text-muted-foreground">
              {t(($) => $.quick_create.prompt_label)}
            </Label>
            <Textarea
              id="quick-create-agent-prompt"
              autoFocus
              value={prompt}
              onChange={(event) => setPrompt(event.target.value)}
              placeholder={t(($) => $.quick_create.prompt_placeholder)}
              className="min-h-32 resize-none"
            />
          </div>
          <RuntimePicker
            runtimes={runtimes}
            runtimesLoading={runtimesLoading}
            members={members}
            currentUserId={currentUserId}
            selectedRuntimeId={selectedRuntimeId}
            onSelect={setSelectedRuntimeId}
          />
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="space-y-1.5">
              <Label htmlFor="quick-create-agent-visibility" className="text-xs text-muted-foreground">
                {t(($) => $.quick_create.visibility_label)}
              </Label>
              <select
                id="quick-create-agent-visibility"
                value={visibility}
                onChange={(event) => setVisibility(event.target.value as AgentVisibility)}
                className="h-9 w-full rounded-md border bg-background px-2 text-sm"
              >
                <option value="workspace">{t(($) => $.visibility.workspace.label)}</option>
                <option value="private">{t(($) => $.visibility.private.label)}</option>
              </select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="quick-create-agent-model" className="text-xs text-muted-foreground">
                {t(($) => $.quick_create.model_label)}
              </Label>
              <Input
                id="quick-create-agent-model"
                value={model}
                onChange={(event) => setModel(event.target.value)}
                placeholder={t(($) => $.quick_create.model_placeholder)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="quick-create-agent-thinking" className="text-xs text-muted-foreground">
                {t(($) => $.quick_create.thinking_label)}
              </Label>
              <Input
                id="quick-create-agent-thinking"
                value={thinkingLevel}
                onChange={(event) => setThinkingLevel(event.target.value)}
                placeholder={t(($) => $.quick_create.thinking_placeholder)}
              />
            </div>
          </div>
          <div className="flex items-start gap-2 rounded-md border bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
            <Bot className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>{t(($) => $.quick_create.runtime_hint)}</span>
          </div>
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={onClose} disabled={quickCreate.isPending}>
            {t(($) => $.quick_create.cancel)}
          </Button>
          <Button
            onClick={() => void submit()}
            disabled={!prompt.trim() || !selectedRuntime || quickCreate.isPending}
          >
            {quickCreate.isPending && <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />}
            {t(($) => $.quick_create.submit)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function formatQuickCreateAgentError(
  err: unknown,
  t: ReturnType<typeof useT<"agents">>["t"],
): string {
  if (err instanceof ApiError && err.body && typeof err.body === "object") {
    const body = err.body as {
      code?: string;
      reason?: string;
      current_version?: string;
      min_version?: string;
    };
    if (body.code === "agent_unavailable") {
      return body.reason || t(($) => $.quick_create.error_agent_unavailable_fallback);
    }
    if (body.code === "daemon_version_unsupported") {
      return t(($) => $.quick_create.error_daemon_version, {
        current: body.current_version || "unknown",
        min: body.min_version || MIN_QUICK_CREATE_AGENT_CLI_VERSION,
      });
    }
  }
  return err instanceof Error && err.message
    ? err.message
    : t(($) => $.quick_create.toast_failed);
}
