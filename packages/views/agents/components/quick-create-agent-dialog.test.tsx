import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithI18n } from "../../test/i18n";
import { QuickCreateAgentDialog } from "./quick-create-agent-dialog";

const mockQuickCreateAgent = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: readonly unknown[] }) => {
    if (queryKey[0] === "runtimes") {
      return {
        data: [
          {
            id: "runtime-1",
            workspace_id: "ws-1",
            daemon_id: "daemon-1",
            name: "Hermes",
            runtime_mode: "local",
            provider: "hermes",
            launch_header: "hermes",
            status: "online",
            device_info: "Mac",
            metadata: {},
            owner_id: "user-1",
            visibility: "private",
            last_seen_at: null,
            created_at: "2026-01-01T00:00:00Z",
            updated_at: "2026-01-01T00:00:00Z",
          },
        ],
        isLoading: false,
      };
    }
    if (queryKey[0] === "members") {
      return {
        data: [
          { user_id: "user-1", name: "Ada", role: "owner" },
        ],
      };
    }
    return { data: [] };
  },
}));

vi.mock("@multica/core/agents", () => ({
  useQuickCreateAgent: () => ({
    mutateAsync: mockQuickCreateAgent,
    isPending: false,
  }),
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (selector?: (state: { user: { id: string } }) => unknown) =>
    selector ? selector({ user: { id: "user-1" } }) : { user: { id: "user-1" } },
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "ws-1",
}));

vi.mock("@multica/core/runtimes", () => ({
  runtimeListOptions: () => ({ queryKey: ["runtimes"] }),
}));

vi.mock("@multica/core/workspace/queries", () => ({
  memberListOptions: () => ({ queryKey: ["members"] }),
}));

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

beforeEach(() => {
  mockQuickCreateAgent.mockReset();
  mockQuickCreateAgent.mockResolvedValue({ task_id: "task-1" });
});

describe("QuickCreateAgentDialog", () => {
  it("submits the quick-create agent request", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithI18n(<QuickCreateAgentDialog onClose={onClose} />);

    await user.type(
      screen.getByPlaceholderText(/Create a frontend QA agent/i),
      "Create a QA agent",
    );
    fireEvent.change(screen.getByLabelText("Visibility"), {
      target: { value: "private" },
    });
    await user.type(screen.getByLabelText("Model"), "opus");

    await user.click(screen.getByRole("button", { name: "Create with AI" }));

    await waitFor(() => {
      expect(mockQuickCreateAgent).toHaveBeenCalledWith({
        prompt: "Create a QA agent",
        runtime_id: "runtime-1",
        visibility: "private",
        model: "opus",
        thinking_level: undefined,
      });
    });
    expect(onClose).toHaveBeenCalled();
  });
});
