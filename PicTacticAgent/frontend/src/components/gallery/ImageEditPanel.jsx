import { useState, useRef, useEffect } from "react";
import { X, Loader2 } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

export function ImageEditPanel({ defaultPrompt = "", onConfirm, onCancel, isLoading = false }) {
  const [prompt, setPrompt] = useState(defaultPrompt);
  const [enhance, setEnhance] = useState(true);
  const textareaRef = useRef(null);
  const { t } = useLocale();

  useEffect(() => {
    textareaRef.current?.focus();
  }, []);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === "Escape" && !isLoading) {
        onCancel?.();
      }
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey) && prompt.trim() && !isLoading) {
        onConfirm?.(prompt.trim(), enhance);
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [prompt, enhance, isLoading, onCancel, onConfirm]);

  return (
    <div className="mt-2 rounded-xl border border-border/40 bg-card p-3 shadow-lg animate-slide-up">
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs font-semibold text-foreground">{t("edit.title")}</span>
        <button
          onClick={onCancel}
          disabled={isLoading}
          className="p-1 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors disabled:opacity-50"
        >
          <X size={14} />
        </button>
      </div>

      <textarea
        ref={textareaRef}
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        placeholder={t("edit.promptPlaceholder")}
        disabled={isLoading}
        rows={2}
        className="w-full resize-none rounded-lg border border-border/30 bg-background px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary/40 disabled:opacity-50"
      />

      <div className="mt-2 flex items-center justify-between">
        <label className="flex items-center gap-1.5 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={enhance}
            onChange={(e) => setEnhance(e.target.checked)}
            disabled={isLoading}
            className="h-3.5 w-3.5 rounded border-border accent-primary"
          />
          <span className="text-[11px] text-muted-foreground">{t("edit.enhanceToggle")}</span>
        </label>

        <div className="flex gap-1.5">
          <button
            onClick={onCancel}
            disabled={isLoading}
            className="rounded-md px-3 py-1.5 text-[11px] font-medium text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors disabled:opacity-50"
          >
            {t("edit.cancel")}
          </button>
          <button
            onClick={() => onConfirm?.(prompt.trim(), enhance)}
            disabled={!prompt.trim() || isLoading}
            className="flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-[11px] font-semibold text-primary-foreground transition-all hover:brightness-110 disabled:opacity-50"
          >
            {isLoading && <Loader2 size={12} className="animate-spin" />}
            {isLoading ? t("edit.loading") : t("edit.confirm")}
          </button>
        </div>
      </div>
    </div>
  );
}
