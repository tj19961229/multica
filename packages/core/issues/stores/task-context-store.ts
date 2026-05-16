import { create } from "zustand";
import type { TaskUsageUpdatePayload } from "../../types/events";

// One row of in-flight context-window state for a running task. updated_at is
// kept for future stale-detection / "no update in 30s → fade" UX; not used yet.
interface TaskContextEntry extends TaskUsageUpdatePayload {
  updated_at: number;
}

interface State {
  entries: Map<string, TaskContextEntry>;
  set: (payload: TaskUsageUpdatePayload) => void;
  remove: (taskId: string) => void;
  get: (taskId: string) => TaskContextEntry | undefined;
  clear: () => void;
}

// Ephemeral client-only store — never persisted. Reset on page refresh by design
// (the next turn will repopulate it). Distinct from issue-level TanStack Query
// cache which holds cumulative billing-style task_usage data.
export const useTaskContextStore = create<State>((set, get) => ({
  entries: new Map(),
  set: (p) =>
    set((s) => {
      const next = new Map(s.entries);
      next.set(p.task_id, { ...p, updated_at: Date.now() });
      return { entries: next };
    }),
  remove: (taskId) =>
    set((s) => {
      if (!s.entries.has(taskId)) return s;
      const next = new Map(s.entries);
      next.delete(taskId);
      return { entries: next };
    }),
  get: (taskId) => get().entries.get(taskId),
  clear: () => set({ entries: new Map() }),
}));

// Selector hook for a single task — usable directly in React components.
// Returns a stable reference per task until the entry mutates.
export function useTaskContext(taskId: string) {
  return useTaskContextStore((s) => s.entries.get(taskId));
}
