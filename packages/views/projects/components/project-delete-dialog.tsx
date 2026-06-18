"use client";

import { useState } from "react";
import { AlertCircle } from "lucide-react";
import { toast } from "sonner";
import { ApiError } from "@multica/core/api";
import { useDeleteProject } from "@multica/core/projects";
import type { Project } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@multica/ui/components/ui/dialog";
import { useT } from "../../i18n";

interface ProjectNotEmptyBody {
  code?: string;
  child_count?: number;
  issue_count?: number;
}

export function ProjectDeleteDialog({
  project,
  open,
  onOpenChange,
  onDeleted,
}: {
  project: Project;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
}) {
  const { t } = useT("projects");
  const deleteProject = useDeleteProject();
  const [needsTreeConfirm, setNeedsTreeConfirm] = useState(false);
  const [childCount, setChildCount] = useState(0);
  const [issueCount, setIssueCount] = useState(0);

  const close = () => {
    setNeedsTreeConfirm(false);
    setChildCount(0);
    setIssueCount(0);
    onOpenChange(false);
  };

  const submit = async () => {
    try {
      await deleteProject.mutateAsync({
        id: project.id,
        confirm: needsTreeConfirm,
      });
      toast.success(t(($) => $.detail.toast_project_deleted));
      close();
      onDeleted?.();
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        const body = err.body as ProjectNotEmptyBody | undefined;
        if (body?.code === "project_not_empty") {
          setChildCount(body.child_count ?? 0);
          setIssueCount(body.issue_count ?? 0);
          setNeedsTreeConfirm(true);
          return;
        }
      }
      toast.error(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) close();
        else onOpenChange(true);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t(($) => $.delete_dialog.title)}</DialogTitle>
          <DialogDescription>
            {needsTreeConfirm
              ? t(($) => $.delete_dialog.non_empty_description, {
                  child_count: childCount,
                  issue_count: issueCount,
                })
              : t(($) => $.delete_dialog.description)}
          </DialogDescription>
        </DialogHeader>
        {needsTreeConfirm && (
          <div className="flex gap-2 rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-xs text-amber-700 dark:text-amber-300">
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>{t(($) => $.delete_dialog.restore_hint)}</span>
          </div>
        )}
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={deleteProject.isPending}
            onClick={close}
          >
            {t(($) => $.delete_dialog.cancel)}
          </Button>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            disabled={deleteProject.isPending}
            onClick={() => void submit()}
          >
            {needsTreeConfirm
              ? t(($) => $.delete_dialog.confirm_tree)
              : t(($) => $.delete_dialog.confirm)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
