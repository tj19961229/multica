import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ContextMeterBar, LiveContextMeter } from "./context-meter";

const mockEntry = vi.fn();
vi.mock("@multica/core/issues/stores/task-context-store", () => ({
  useTaskContext: (id: string) => mockEntry(id),
}));

describe("ContextMeterBar (pure)", () => {
  it("renders X / Y · Z% from props", () => {
    render(<ContextMeterBar usedTokens={12_500} maxTokens={200_000} />);
    expect(screen.getByText(/12\.5k/)).toBeInTheDocument();
    expect(screen.getByText(/200/)).toBeInTheDocument();
    expect(screen.getByText(/6%/)).toBeInTheDocument();
  });

  it("uses amber color between 60% and 85%", () => {
    const { container } = render(
      <ContextMeterBar usedTokens={140_000} maxTokens={200_000} />,
    );
    expect(container.querySelector('[class*="bg-amber"]')).toBeTruthy();
  });

  it("uses red color above 85%", () => {
    const { container } = render(
      <ContextMeterBar usedTokens={180_000} maxTokens={200_000} />,
    );
    expect(container.querySelector('[class*="bg-red"]')).toBeTruthy();
  });

  it("falls back to 200k window when maxTokens is 0", () => {
    render(<ContextMeterBar usedTokens={20_000} maxTokens={0} />);
    // 20k / 200k = 10%
    expect(screen.getByText(/10%/)).toBeInTheDocument();
  });
});

describe("LiveContextMeter (store-aware)", () => {
  it("renders nothing when store has no entry", () => {
    mockEntry.mockReturnValueOnce(undefined);
    const { container } = render(<LiveContextMeter taskId="t1" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders ContextMeterBar with prompt+cache_read sum from store entry", () => {
    mockEntry.mockReturnValueOnce({
      task_id: "t1", agent_id: "a1", issue_id: "i1", model: "x",
      prompt_tokens: 1_000, cache_read_tokens: 9_000, cache_write_tokens: 0,
      max_context_tokens: 100_000, updated_at: Date.now(),
    });
    render(<LiveContextMeter taskId="t1" />);
    // 1k + 9k = 10k → 10% of 100k
    expect(screen.getByText(/10\.0k|10k/)).toBeInTheDocument();
    expect(screen.getByText(/10%/)).toBeInTheDocument();
  });
});
