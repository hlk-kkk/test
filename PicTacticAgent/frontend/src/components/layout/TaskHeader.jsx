import { useState, useEffect } from "react";
import { Check, ChevronDown, ChevronUp, Copy, Plus, Sparkles } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";

const statusConfig = {
  idle: { chip: "bg-muted text-muted-foreground", label: "sidebar.status.pending", pulse: false },
  creating: { chip: "bg-primary/20 text-primary", label: "progress.initializing", pulse: true },
  running: { chip: "bg-primary/20 text-primary", label: "progress.generating", pulse: true },
  completing: { chip: "bg-emerald-500/20 text-emerald-300", label: "progress.completing", pulse: true },
  completed: { chip: "bg-emerald-500/20 text-emerald-300", label: "sidebar.status.completed", pulse: false },
  failed: { chip: "bg-destructive/20 text-destructive", label: "sidebar.status.failed", pulse: false },
  cancelled: { chip: "bg-muted text-muted-foreground", label: "sidebar.status.cancelled", pulse: false },
  loading: { chip: "bg-primary/20 text-primary", label: "progress.loading", pulse: true },
};

export function TaskHeader({ status, taskContext, onNewTask, selectedTaskId }) {
  const { t } = useLocale();
  const [expanded, setExpanded] = useState(false);
  const [copied, setCopied] = useState(false);

  const shouldShow = selectedTaskId || (status && status !== "idle");

  const prompt = taskContext?.prompt || "";
  const isLong = prompt.length > 96;

  // Reset expanded state when prompt changes
  useEffect(() => {
    setExpanded(false);
  }, [prompt, selectedTaskId]);

  if (!shouldShow) return null;

  const config = statusConfig[status] || statusConfig.idle;

  const handleCopy = async () => {
    if (!prompt) return;
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      const textarea = document.createElement("textarea");
      textarea.value = prompt;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="mb-5 w-full">
      <div className="brand-card flex flex-wrap items-center justify-between gap-4 p-4 sm:p-5">
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex items-center gap-2">
            <span className="brand-chip bg-secondary/70 text-foreground">
              <Sparkles className="h-3.5 w-3.5 text-primary" />
              Active Session
            </span>
            <span className={cn("inline-flex items-center gap-2 rounded-full px-2.5 py-1 text-[11px] font-medium", config.chip)}>
              <span className={cn("h-1.5 w-1.5 rounded-full", config.pulse ? "animate-pulse bg-current" : "bg-current/80")} />
              {t(config.label)}
            </span>
          </div>

          {prompt ? (
            <div className="flex items-start gap-2">
              <p className={cn(
                "flex-1 text-sm leading-relaxed text-foreground/90 transition-all",
                !expanded && isLong && "line-clamp-2"
              )}>
                {expanded || !isLong ? prompt : `${prompt.slice(0, 96)}...`}
              </p>
              <div className="flex shrink-0 items-center gap-1 pt-0.5">
                <button
                  onClick={handleCopy}
                  className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                  title={t("header.copyPrompt")}
                >
                  {copied ? (
                    <Check className="h-3.5 w-3.5 text-emerald-400" />
                  ) : (
                    <Copy className="h-3.5 w-3.5" />
                  )}
                </button>
                {isLong ? (
                  <button
                    onClick={() => setExpanded(!expanded)}
                    className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                    title={expanded ? t("header.collapsePrompt") : t("header.expandPrompt")}
                  >
                    {expanded ? (
                      <ChevronUp className="h-3.5 w-3.5" />
                    ) : (
                      <ChevronDown className="h-3.5 w-3.5" />
                    )}
                  </button>
                ) : null}
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{t("empty.readyText")}</p>
          )}

          {taskContext ? (
            <div className="flex flex-wrap items-center gap-2 text-[11px]">
              {taskContext.mode ? (
                <span className="rounded-full border border-border/70 bg-secondary/60 px-2 py-0.5 text-muted-foreground">
                  {taskContext.mode === "image_to_image" ? t("mode.imageEdit") : t("mode.textToImage")}
                </span>
              ) : null}
              {taskContext.aspectRatio ? (
                <span className="rounded-full border border-border/70 bg-secondary/60 px-2 py-0.5 text-muted-foreground">
                  {taskContext.aspectRatio}
                </span>
              ) : null}
              {taskContext.size ? (
                <span className="rounded-full border border-border/70 bg-secondary/60 px-2 py-0.5 text-muted-foreground">
                  {taskContext.size}
                </span>
              ) : null}
            </div>
          ) : null}
        </div>

        <button
          onClick={onNewTask}
          className="brand-btn-primary px-4 py-2.5 text-sm font-semibold"
        >
          <Plus size={14} />
          {t("sidebar.newTask")}
        </button>
      </div>
    </div>
  );
}
