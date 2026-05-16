import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { IssueAgentContextsSection } from "./issue-agent-contexts-section";

const mockData = vi.fn();

vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>("@tanstack/react-query");
  return {
    ...actual,
    useQuery: () => ({ data: mockData(), isLoading: false, isError: false }),
  };
});

vi.mock("@multica/core/workspace/hooks", () => ({
  useActorName: () => ({
    getActorName: (_t: string, id: string) =>
      id === "agent-a" ? "Atlas" : id === "agent-b" ? "Forge" : id,
    getActorAvatarUrl: () => undefined,
  }),
}));

vi.mock("../../common/actor-avatar", () => ({
  ActorAvatar: ({ actorId }: { actorId: string }) => <span data-testid={`avatar-${actorId}`} />,
}));

vi.mock("../../i18n", () => ({
  useT: () => ({
    t: (selector: (s: { detail: Record<string, string> }) => string) =>
      selector({ detail: { section_agent_contexts: "Agent contexts" } }),
  }),
}));

describe("IssueAgentContextsSection", () => {
  it("renders nothing when rows is empty", () => {
    mockData.mockReturnValueOnce([]);
    const { container } = render(<IssueAgentContextsSection issueId="i1" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders one row per agent with name + status + bar (default open, mirrors Token usage)", () => {
    mockData.mockReturnValue([
      {
        agent_id: "agent-a",
        max_context_tokens_seen: 12_500,
        latest_task_id: "t1",
        latest_task_status: "running",
        latest_model: "claude-sonnet-4-6",
        max_context_tokens: 200_000,
      },
      {
        agent_id: "agent-b",
        max_context_tokens_seen: 4_000,
        latest_task_id: "t2",
        latest_task_status: "completed",
        latest_model: "claude-sonnet-4-6",
        max_context_tokens: 200_000,
      },
    ]);
    render(<IssueAgentContextsSection issueId="i1" />);
    // Section button visible
    expect(screen.getByText("Agent contexts")).toBeInTheDocument();
    // Default open (mirrors Token usage useState(true)): rows are visible
    expect(screen.getByText("Atlas")).toBeInTheDocument();
    expect(screen.getByText("Forge")).toBeInTheDocument();
    expect(screen.getByText(/running/)).toBeInTheDocument();
    expect(screen.getByText(/completed/)).toBeInTheDocument();
    expect(screen.getByTestId("avatar-agent-a")).toBeInTheDocument();
    // Bar renders formatted token count
    expect(screen.getAllByText(/12\.5k/i).length).toBeGreaterThan(0);
    // Collapse via header click
    fireEvent.click(screen.getByText("Agent contexts"));
    expect(screen.queryByText("Atlas")).toBeNull();
  });
});
