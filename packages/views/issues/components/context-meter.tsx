"use client";

import { useTaskContext } from "@multica/core/issues/stores/task-context-store";
import { formatTokenCount } from "@multica/core/format/tokens";
import { cn } from "@multica/ui/lib/utils";

interface BarProps {
  usedTokens: number;
  maxTokens: number;
  className?: string;
}

// Pure progress bar. Color thresholds: <60% green, 60-85% amber, >85% red.
// Caller decides where the numbers come from (live store / server query).
export function ContextMeterBar({ usedTokens, maxTokens, className }: BarProps) {
  const max = maxTokens || 200_000;
  const ratio = Math.min(usedTokens / max, 1);
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
        {formatTokenCount(usedTokens)} / {formatTokenCount(max)} · {pct}%
      </span>
    </div>
  );
}

// Store-aware wrapper used by anything that wants live per-task context.
// Returns null when the store has no entry for this task (task not yet
// reported a turn). For static historical data prefer ContextMeterBar directly.
export function LiveContextMeter({ taskId, className }: { taskId: string; className?: string }) {
  const entry = useTaskContext(taskId);
  if (!entry) return null;
  return (
    <ContextMeterBar
      usedTokens={entry.prompt_tokens + entry.cache_read_tokens}
      maxTokens={entry.max_context_tokens}
      className={className}
    />
  );
}
