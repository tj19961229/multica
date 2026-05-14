"use client";

import { useTaskContext } from "@multica/core/issues/stores/task-context-store";
import { formatTokenCount } from "@multica/core/format/tokens";
import { cn } from "@multica/ui/lib/utils";

interface Props {
  taskId: string;
  className?: string;
}

// Live "prompt size / max context window" meter for an in-flight task. Sourced
// from the realtime task:usage_update WS stream via useTaskContextStore. Returns
// null until the task posts its first turn — the parent should remain layout-
// stable in that gap (no skeleton; this is a hint, not a primary affordance).
export function ContextMeter({ taskId, className }: Props) {
  const entry = useTaskContext(taskId);
  if (!entry) return null;

  const used = entry.prompt_tokens + entry.cache_read_tokens;
  const max = entry.max_context_tokens || 200_000;
  const ratio = Math.min(used / max, 1);
  const pct = Math.round(ratio * 100);

  const barColor =
    ratio > 0.85 ? "bg-red-500/80"
    : ratio > 0.6 ? "bg-amber-500/80"
    : "bg-emerald-500/80";

  return (
    <div className={cn("flex items-center gap-2 text-xs", className)}>
      <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full transition-[width] duration-300", barColor)}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-muted-foreground tabular-nums">
        {formatTokenCount(used)} / {formatTokenCount(max)} · {pct}%
      </span>
    </div>
  );
}
