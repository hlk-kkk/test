import { useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { X, Download, Film } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

export function VideoModal({ video, isOpen, onClose }) {
  const { t } = useLocale();

  const handleKeyDown = useCallback(
    (e) => {
      if (e.key === "Escape") {
        onClose();
      }
    },
    [onClose]
  );

  useEffect(() => {
    if (!isOpen) return;
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, handleKeyDown]);

  const handleDownload = async () => {
    try {
      const response = await fetch(video.url);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `pictactic-video-${video.id}.mp4`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Download failed:", err);
    }
  };

  if (!isOpen || !video) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-6 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="relative max-h-[92vh] w-full max-w-5xl animate-fade-in-scale overflow-hidden rounded-2xl bg-card shadow-2xl ring-1 ring-border/30"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute right-3 top-3 z-10 flex h-8 w-8 items-center justify-center rounded-lg bg-black/40 text-white/80 backdrop-blur-sm transition-colors hover:bg-black/60 hover:text-white"
        >
          <X className="h-4 w-4" />
        </button>

        <div className="flex flex-col md:flex-row">
          {/* Video player area */}
          <div className="flex-1 bg-black/10">
            <video
              src={video.url}
              poster={video.start_image || undefined}
              controls
              autoPlay
              playsInline
              className="h-full max-h-[85vh] w-full object-contain"
            />
          </div>

          {/* Info sidebar */}
          <div className="w-full p-6 md:w-72 md:max-h-[92vh] md:overflow-y-auto">
            {/* Video icon */}
            <div className="mb-6 flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10">
                <Film className="h-5 w-5 text-primary" />
              </div>
              <div>
                <div className="text-lg font-bold text-foreground font-display">
                  {video.duration != null ? `${video.duration}s` : "--"}
                </div>
                <div className="text-[10px] font-medium uppercase tracking-[0.1em] text-muted-foreground">
                  {t("video.duration")}
                </div>
              </div>
            </div>

            {/* Meta */}
            <div className="space-y-2.5 border-t border-border/30 pt-4">
              {[
                { label: t("modal.round"), value: video.round_number != null ? `#${video.round_number}` : "--" },
                { label: t("modal.provider"), value: video.provider || "--" },
                { label: t("video.resolution"), value: video.resolution || "--" },
              ].map(({ label, value }) => (
                <div key={label} className="flex justify-between text-xs">
                  <span className="text-muted-foreground">{label}</span>
                  <span className="font-medium text-foreground">{value}</span>
                </div>
              ))}
            </div>

            {/* Download */}
            <button
              onClick={handleDownload}
              className="mt-5 flex w-full items-center justify-center gap-2 rounded-lg bg-primary py-2 text-xs font-semibold text-primary-foreground transition-all hover:brightness-110"
            >
              <Download className="h-3.5 w-3.5" />
              {t("video.download")}
            </button>
          </div>
        </div>
      </div>
    </div>,
    document.body
  );
}
