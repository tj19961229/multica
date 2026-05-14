import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ContextMeter } from "./context-meter";

const mockEntry = vi.fn();
vi.mock("@multica/core/issues/stores/task-context-store", () => ({
  useTaskContext: (id: string) => mockEntry(id),
}));

describe("ContextMeter", () => {
  it("renders nothing when no entry", () => {
    mockEntry.mockReturnValueOnce(undefined);
    const { container } = render(<ContextMeter taskId="t1" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders X / Y · Z% when entry exists", () => {
    mockEntry.mockReturnValueOnce({
      task_id: "t1", agent_id: "a1", issue_id: "i1", model: "claude-sonnet-4-6",
      prompt_tokens: 12_500, cache_read_tokens: 0, cache_write_tokens: 0,
      max_context_tokens: 200_000, updated_at: Date.now(),
    });
    render(<ContextMeter taskId="t1" />);
    expect(screen.getByText(/12\.5k/)).toBeInTheDocument();
    expect(screen.getByText(/200\.0k|200k/)).toBeInTheDocument();
    expect(screen.getByText(/6%/)).toBeInTheDocument();
  });

  it("includes cache_read_tokens in the used total", () => {
    mockEntry.mockReturnValueOnce({
      task_id: "t1", agent_id: "a1", issue_id: "i1", model: "x",
      prompt_tokens: 1_000, cache_read_tokens: 9_000, cache_write_tokens: 0,
      max_context_tokens: 100_000, updated_at: Date.now(),
    });
    render(<ContextMeter taskId="t1" />);
    // 1k + 9k = 10k used out of 100k → 10%
    expect(screen.getByText(/10\.0k|10k/)).toBeInTheDocument();
    expect(screen.getByText(/10%/)).toBeInTheDocument();
  });

  it("uses amber color between 60% and 85%", () => {
    mockEntry.mockReturnValueOnce({
      task_id: "t1", agent_id: "a1", issue_id: "i1", model: "x",
      prompt_tokens: 140_000, cache_read_tokens: 0, cache_write_tokens: 0,
      max_context_tokens: 200_000, updated_at: Date.now(),
    });
    const { container } = render(<ContextMeter taskId="t1" />);
    expect(container.querySelector('[class*="bg-amber"]')).toBeTruthy();
  });

  it("uses red color above 85%", () => {
    mockEntry.mockReturnValueOnce({
      task_id: "t1", agent_id: "a1", issue_id: "i1", model: "x",
      prompt_tokens: 180_000, cache_read_tokens: 0, cache_write_tokens: 0,
      max_context_tokens: 200_000, updated_at: Date.now(),
    });
    const { container } = render(<ContextMeter taskId="t1" />);
    expect(container.querySelector('[class*="bg-red"]')).toBeTruthy();
  });
});
