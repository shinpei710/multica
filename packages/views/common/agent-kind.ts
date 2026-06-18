import type { Agent } from "@multica/core/types";

export function isRuntimeBlankAgent(agent: Pick<Agent, "kind">): boolean {
  return (agent.kind ?? "configured") === "runtime_blank";
}

export function splitRuntimeBlankAgents<T extends Pick<Agent, "kind">>(
  agents: T[],
): { configured: T[]; runtimes: T[] } {
  const configured: T[] = [];
  const runtimes: T[] = [];
  for (const agent of agents) {
    if (isRuntimeBlankAgent(agent)) {
      runtimes.push(agent);
    } else {
      configured.push(agent);
    }
  }
  return { configured, runtimes };
}
