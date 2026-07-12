import ReactMarkdown from "react-markdown";
import { cn } from "@/lib/utils";
import { X, Zap } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

function StatusIndicator({ status }) {
  const isActive = status === "creating" || status === "running";
  const color =
    status === "completed" ? "bg-emerald-500" :
    status === "failed" ? "bg-red-500" :
    status === "cancelled" ? "bg-muted-foreground" :
    "bg-primary";

  return (
    <span className="relative flex h-2 w-2">
      {isActive && (
        <span className={cn("absolute inline-flex h-full w-full animate-ping rounded-full opacity-60", color)} />
      )}
      <span className={cn("relative inline-flex h-2 w-2 rounded-full", color)} />
    </span>
  );
}

function PromptBlock({ icon, label, content, labelClassName = "text-muted-foreground" }) {
  return (
    <div className="mt-3 rounded-lg bg-secondary/50 p-3 ring-1 ring-border/20">
      <div className={cn("mb-1.5 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-[0.1em]", labelClassName)}>
        {icon}
        {label}
      </div>
      <div className="prose-prompt text-xs text-foreground/80 leading-relaxed">
        <ReactMarkdown>{content}</ReactMarkdown>
      </div>
    </div>
  );
}

export function ProgressDisplay({ progress, status, error, onCancel }) {
  const { t } = useLocale();

  if (status === "idle") return null;

  const getStatusText = () => {
    if (error) return error;
    if (progress) return progress.message;
    switch (status) {
      case "creating": return t("progress.initializing");
      case "running": return t("progress.generating");
      case "completed": return t("progress.completed");
      case "failed": return t("progress.failed");
      case "cancelled": return t("progress.cancelled");
      default: return "";
    }
  };

  const progressPercent = progress
    ? Math.min(100, (((progress.current_round - 1) * progress.images_generated + progress.images_in_current_round) / (progress.max_rounds * (progress.images_generated || 5))) * 100)
    : 0;

  const isRunning = status === "creating" || status === "running";

  return (
    <div className="w-full rounded-xl bg-card ring-1 ring-border/30 p-4 animate-fade-in-scale">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <StatusIndicator status={status} />
          <span className="text-sm font-medium text-foreground">{getStatusText()}</span>
        </div>
        <div className="flex items-center gap-3">
          {progress && isRunning && (
            <div className="flex items-center gap-3 text-[11px] tabular-nums text-muted-foreground">
              <span>R{progress.current_round}/{progress.max_rounds}</span>
              <span className="text-border">|</span>
              <span>{t("progress.imgs", { count: progress.images_generated })}</span>
              <span className="text-border">|</span>
              <span className="text-primary font-semibold">{(progress.current_best_score ?? 0).toFixed(2)}</span>
            </div>
          )}
          {isRunning && onCancel && (
            <button
              onClick={onCancel}
              className="flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {isRunning && (
        <div className="mt-3">
          <div className="h-1 overflow-hidden rounded-full bg-secondary">
            <div
              className="h-full rounded-full bg-gradient-to-r from-primary to-primary/60 transition-all duration-700 ease-out"
              style={{ width: `${Math.max(3, progressPercent)}%` }}
            />
          </div>
        </div>
      )}

      {progress?.enhanced_prompt && (
        <PromptBlock
          icon={<Zap className="h-3 w-3" />}
          label={t("progress.enhancedPrompt")}
          content={progress.enhanced_prompt}
          labelClassName="text-primary"
        />
      )}
    </div>
  );
}
