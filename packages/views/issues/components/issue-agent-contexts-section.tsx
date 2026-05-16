"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";
import { issueAgentContextsOptions } from "@multica/core/issues/queries";
import { useActorName } from "@multica/core/workspace/hooks";
import { ActorAvatar } from "../../common/actor-avatar";
import { ContextMeterBar } from "./context-meter";
import { useT } from "../../i18n";

interface Props {
  issueId: string;
}

// Right-sidebar collapsible section listing every agent that has run a task
// on this issue, ordered by max context window occupancy (server-sorted).
// Each row shows the agent's avatar, name, latest task status, and a
// ContextMeterBar showing "max prompt+cache_read tokens reached / model
// context window". Historical view sourced from
// GET /api/issues/{id}/agent-contexts. For live in-flight per-task context
// see useTaskContextStore (separate concern).
export function IssueAgentContextsSection({ issueId }: Props) {
  const { t } = useT("issues");
  const { getActorName } = useActorName();
  const [open, setOpen] = useState(true);
  const { data: rows = [] } = useQuery(issueAgentContextsOptions(issueId));

  if (!rows.length) return null;

  return (
    <div>
      <button
        className={`flex w-full items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors mb-2 hover:bg-accent/70 ${open ? "" : "text-muted-foreground hover:text-foreground"}`}
        onClick={() => setOpen(!open)}
      >
        {t(($) => $.detail.section_agent_contexts)}
        <ChevronRight className={`!size-3 shrink-0 stroke-[2.5] text-muted-foreground transition-transform ${open ? "rotate-90" : ""}`} />
      </button>
      {open && (
        <div className="flex flex-col gap-1.5 pl-2">
          {rows.map((r) => (
            <div key={r.agent_id} className="flex items-center gap-2 text-xs">
              <ActorAvatar actorType="agent" actorId={r.agent_id} size={18} enableHoverCard />
              <span className="truncate font-medium">{getActorName("agent", r.agent_id)}</span>
              <span className="shrink-0 text-[11px] text-muted-foreground">{r.latest_task_status}</span>
              <ContextMeterBar
                usedTokens={r.max_context_tokens_seen}
                maxTokens={r.max_context_tokens}
                className="ml-auto"
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
