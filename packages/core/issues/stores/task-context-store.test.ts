import { describe, it, expect, beforeEach } from "vitest";
import { useTaskContextStore } from "./task-context-store";

describe("useTaskContextStore", () => {
  beforeEach(() => useTaskContextStore.getState().clear());

  const samplePayload = {
    task_id: "t1",
    agent_id: "a1",
    issue_id: "i1",
    model: "claude-sonnet-4-6",
    prompt_tokens: 100,
    cache_read_tokens: 0,
    cache_write_tokens: 0,
    max_context_tokens: 200_000,
  };

  it("upserts context info by task_id", () => {
    useTaskContextStore.getState().set(samplePayload);
    expect(useTaskContextStore.getState().get("t1")?.prompt_tokens).toBe(100);
  });

  it("set overwrites prior entry for same task_id", () => {
    useTaskContextStore.getState().set(samplePayload);
    useTaskContextStore.getState().set({ ...samplePayload, prompt_tokens: 500 });
    expect(useTaskContextStore.getState().get("t1")?.prompt_tokens).toBe(500);
  });

  it("returns undefined for unknown task", () => {
    expect(useTaskContextStore.getState().get("nope")).toBeUndefined();
  });

  it("remove drops the entry", () => {
    useTaskContextStore.getState().set(samplePayload);
    useTaskContextStore.getState().remove("t1");
    expect(useTaskContextStore.getState().get("t1")).toBeUndefined();
  });

  it("remove on unknown task_id is a no-op", () => {
    expect(() => useTaskContextStore.getState().remove("nope")).not.toThrow();
  });

  it("entries Map ref changes on set so subscribers re-render", () => {
    const before = useTaskContextStore.getState().entries;
    useTaskContextStore.getState().set(samplePayload);
    const after = useTaskContextStore.getState().entries;
    expect(after).not.toBe(before);
  });

  it("clear empties all entries", () => {
    useTaskContextStore.getState().set(samplePayload);
    useTaskContextStore.getState().set({ ...samplePayload, task_id: "t2" });
    useTaskContextStore.getState().clear();
    expect(useTaskContextStore.getState().entries.size).toBe(0);
  });
});
