import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Download, Loader2, Undo2, WandSparkles, X, ZoomIn } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

export function ImageEditDialog({
  image,
  defaultPrompt = "",
  isLoading = false,
  error = null,
  editCount = 0,
  canUndo = false,
  onConfirm,
  onUndo,
  onClose,
}) {
  const { t } = useLocale();
  const [prompt, setPrompt] = useState(defaultPrompt);
  const [enhance, setEnhance] = useState(true);
  const [lightboxOpen, setLightboxOpen] = useState(false);
  const textareaRef = useRef(null);

  useEffect(() => {
    textareaRef.current?.focus();
  }, []);

  useEffect(() => {
    const handleKeyDown = (event) => {
      if (event.key === "Escape") {
        if (lightboxOpen) {
          setLightboxOpen(false);
        } else if (!isLoading) {
          onClose?.();
        }
      }

      if (
        event.key === "Enter" &&
        (event.metaKey || event.ctrlKey) &&
        prompt.trim() &&
        !isLoading
      ) {
        onConfirm?.(prompt.trim(), enhance);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [enhance, isLoading, lightboxOpen, onClose, onConfirm, prompt]);

  const shortId = image?.id ? image.id.slice(0, 8) : "";

  const handleDownload = async () => {
    try {
      const response = await fetch(image.url);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `pictactic-${image.id}.png`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Download failed:", err);
    }
  };

  if (!image) return null;

  return createPortal(
    <>
      {lightboxOpen && (
        <div
          className="fixed inset-0 z-[80] flex items-center justify-center bg-black/95"
          onClick={() => setLightboxOpen(false)}
        >
          <img
            src={image.url}
            alt={`Full preview ${image.id}`}
            className="max-h-[92vh] max-w-[92vw] object-contain animate-fade-in select-none"
            draggable={false}
            onClick={(e) => e.stopPropagation()}
          />
          <button
            onClick={(e) => { e.stopPropagation(); setLightboxOpen(false); }}
            className="absolute right-4 top-4 flex h-10 w-10 items-center justify-center rounded-full bg-white/10 text-white/80 transition-colors hover:bg-white/20 hover:text-white backdrop-blur-sm"
          >
            <X className="h-5 w-5" />
          </button>
          <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-3">
            <button
              onClick={(e) => { e.stopPropagation(); handleDownload(); }}
              className="flex items-center gap-2 rounded-full bg-primary px-4 py-1.5 text-sm font-semibold text-primary-foreground transition-colors hover:brightness-110"
            >
              <Download className="h-4 w-4" />
              {t("modal.download")}
            </button>
          </div>
        </div>
      )}

      <div
        className="fixed inset-0 z-[70] flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm sm:p-6"
        onClick={() => {
          if (!isLoading) onClose?.();
        }}
      >
        <div
          className="flex h-[90vh] w-full max-w-6xl flex-col overflow-hidden rounded-3xl border border-border/50 bg-card/95 shadow-2xl"
          onClick={(event) => event.stopPropagation()}
        >
          <header className="flex items-center justify-between border-b border-border/40 px-5 py-3">
            <div>
              <h3 className="text-sm font-semibold text-foreground">{t("edit.dialogTitle")}</h3>
              <p className="mt-0.5 text-[11px] text-muted-foreground">
                {t("edit.dialogSubtitle")}
              </p>
            </div>
            <button
              onClick={onClose}
              disabled={isLoading}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
            >
              <X size={15} />
            </button>
          </header>

          <div className="grid min-h-0 flex-1 gap-0 lg:grid-cols-[1.4fr_0.8fr]">
            <section className="relative min-h-0 border-b border-border/40 bg-[linear-gradient(45deg,rgba(255,255,255,0.03)_25%,transparent_25%,transparent_75%,rgba(255,255,255,0.03)_75%),linear-gradient(45deg,rgba(255,255,255,0.03)_25%,transparent_25%,transparent_75%,rgba(255,255,255,0.03)_75%)] bg-[length:22px_22px] bg-[position:0_0,11px_11px] lg:border-b-0 lg:border-r">
              <div className="flex h-full flex-col">
                <div className="flex items-center justify-between px-4 py-2.5">
                  <span className="text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
                    {t("edit.canvasLabel")}
                  </span>
                  <span className="rounded-full border border-border/60 bg-secondary/60 px-2 py-0.5 text-[10px] text-muted-foreground">
                    #{shortId}
                  </span>
                </div>

                <div className="group/canvas relative min-h-0 flex-1 px-4 pb-4">
                  <img
                    src={image.url}
                    alt={`Editable ${image.id}`}
                    className="h-full w-full cursor-zoom-in rounded-2xl border border-border/40 bg-black/20 object-contain"
                    onClick={() => setLightboxOpen(true)}
                  />

                  <div className="pointer-events-none absolute bottom-7 right-7 flex items-center gap-1.5 rounded-full bg-black/40 px-2.5 py-1 text-[10px] text-white/60 opacity-0 backdrop-blur-sm transition-opacity group-hover/canvas:opacity-100">
                    <ZoomIn className="h-3 w-3" />
                    {t("modal.clickToZoom")}
                  </div>

                  {isLoading && (
                    <div className="absolute inset-4 flex items-center justify-center rounded-2xl bg-black/45 backdrop-blur-sm">
                      <div className="flex items-center gap-2 rounded-lg bg-black/50 px-3 py-2 text-xs text-white">
                        <Loader2 size={14} className="animate-spin" />
                        {t("edit.loading")}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </section>

            <aside className="flex min-h-0 flex-col p-4">
              <div className="mb-3 flex items-center justify-between">
                <span className="text-xs font-semibold text-foreground">{t("edit.title")}</span>
                <span className="text-[11px] text-muted-foreground">
                  {t("edit.historyCount", { count: editCount })}
                </span>
              </div>

              <textarea
                ref={textareaRef}
                value={prompt}
                onChange={(event) => setPrompt(event.target.value)}
                placeholder={t("edit.promptPlaceholder")}
                disabled={isLoading}
                rows={7}
                className="min-h-[140px] w-full resize-none rounded-xl border border-border/40 bg-background px-3 py-2 text-sm text-foreground outline-none transition-shadow placeholder:text-muted-foreground focus:ring-2 focus:ring-primary/30 disabled:opacity-60"
              />

              <label className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
                <input
                  type="checkbox"
                  checked={enhance}
                  onChange={(event) => setEnhance(event.target.checked)}
                  disabled={isLoading}
                  className="h-3.5 w-3.5 rounded border-border accent-primary"
                />
                {t("edit.enhanceToggle")}
              </label>

              {error ? (
                <div className="mt-3 rounded-lg border border-rose-500/30 bg-rose-500/10 px-3 py-2 text-xs text-rose-200">
                  {error}
                </div>
              ) : null}

              <div className="mt-auto space-y-2 pt-4">
                <button
                  onClick={onUndo}
                  disabled={!canUndo || isLoading}
                  className="flex w-full items-center justify-center gap-2 rounded-xl border border-border/60 bg-secondary/70 py-2.5 text-xs font-semibold text-foreground transition-colors hover:bg-secondary disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <Undo2 size={14} />
                  {t("edit.undo")}
                </button>

                <button
                  onClick={handleDownload}
                  disabled={isLoading}
                  className="flex w-full items-center justify-center gap-2 rounded-xl border border-border/60 bg-secondary/70 py-2.5 text-xs font-semibold text-foreground transition-colors hover:bg-secondary disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <Download size={14} />
                  {t("modal.download")}
                </button>

                <button
                  onClick={() => onConfirm?.(prompt.trim(), enhance)}
                  disabled={!prompt.trim() || isLoading}
                  className="flex w-full items-center justify-center gap-2 rounded-xl bg-primary py-2.5 text-xs font-semibold text-primary-foreground transition-all hover:brightness-105 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {isLoading ? <Loader2 size={14} className="animate-spin" /> : <WandSparkles size={14} />}
                  {isLoading ? t("edit.loading") : t("edit.confirm")}
                </button>

                <button
                  onClick={onClose}
                  disabled={isLoading}
                  className="w-full rounded-xl border border-border/60 py-2.5 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {t("edit.cancel")}
                </button>
              </div>
            </aside>
          </div>
        </div>
      </div>
    </>,
    document.body
  );
}
