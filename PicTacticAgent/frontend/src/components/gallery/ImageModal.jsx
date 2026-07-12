import { useEffect, useCallback, useState } from "react";
import { createPortal } from "react-dom";
import { X, Star, Download, Pencil, ChevronLeft, ChevronRight, ZoomIn } from "lucide-react";
import { ScoreBar } from "./ScoreBar";
import { useLocale } from "@/hooks/useLocale";

const scoreKeyMap = {
  prompt_alignment: "modal.score.promptAlignment",
  text_quality: "modal.score.textQuality",
  layout_rationality: "modal.score.layoutRationality",
  visual_attraction: "modal.score.visualAttraction",
  technical_integrity: "modal.score.technicalIntegrity",
  identity_preservation: "modal.score.identityPreservation",
};

export function ImageModal({ images, currentIndex, onNavigate, onClose, onEdit }) {
  const { t } = useLocale();
  const image = images[currentIndex];
  const [lightboxOpen, setLightboxOpen] = useState(false);

  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex < images.length - 1;

  const goPrev = useCallback(() => {
    if (hasPrev) onNavigate(currentIndex - 1);
  }, [hasPrev, currentIndex, onNavigate]);

  const goNext = useCallback(() => {
    if (hasNext) onNavigate(currentIndex + 1);
  }, [hasNext, currentIndex, onNavigate]);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === "Escape") {
        if (lightboxOpen) {
          setLightboxOpen(false);
        } else {
          onClose();
        }
      } else if (e.key === "ArrowLeft") {
        goPrev();
      } else if (e.key === "ArrowRight") {
        goNext();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose, goPrev, goNext, lightboxOpen]);

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
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-6"
      onClick={onClose}
    >
      {/* Lightbox: full-screen image preview with nav + download */}
      {lightboxOpen && (
        <div
          className="fixed inset-0 z-[60] flex items-center justify-center bg-black/95"
          onClick={() => setLightboxOpen(false)}
        >
          <img
            key={image.id}
            src={image.url}
            alt={`Full preview ${image.id}`}
            className="max-h-[92vh] max-w-[92vw] object-contain animate-fade-in select-none"
            draggable={false}
            onClick={(e) => e.stopPropagation()}
          />

          {/* Top bar: counter + close */}
          <div className="absolute top-4 left-1/2 -translate-x-1/2 flex items-center gap-3">
            {images.length > 1 && (
              <span className="rounded-full bg-white/10 px-3.5 py-1.5 text-sm tabular-nums text-white/80 backdrop-blur-sm">
                {currentIndex + 1} / {images.length}
              </span>
            )}
          </div>
          <button
            onClick={(e) => { e.stopPropagation(); setLightboxOpen(false); }}
            className="absolute right-4 top-4 flex h-10 w-10 items-center justify-center rounded-full bg-white/10 text-white/80 transition-colors hover:bg-white/20 hover:text-white backdrop-blur-sm"
          >
            <X className="h-5 w-5" />
          </button>

          {/* Nav arrows */}
          {hasPrev && (
            <button
              onClick={(e) => { e.stopPropagation(); goPrev(); }}
              className="absolute left-4 top-1/2 -translate-y-1/2 flex h-12 w-12 items-center justify-center rounded-full bg-white/10 text-white/80 transition-colors hover:bg-white/20 hover:text-white backdrop-blur-sm"
            >
              <ChevronLeft className="h-6 w-6" />
            </button>
          )}
          {hasNext && (
            <button
              onClick={(e) => { e.stopPropagation(); goNext(); }}
              className="absolute right-4 top-1/2 -translate-y-1/2 flex h-12 w-12 items-center justify-center rounded-full bg-white/10 text-white/80 transition-colors hover:bg-white/20 hover:text-white backdrop-blur-sm"
            >
              <ChevronRight className="h-6 w-6" />
            </button>
          )}

          {/* Bottom bar: score + download */}
          <div className="absolute bottom-4 left-1/2 -translate-x-1/2 flex items-center gap-3">
            <span className="rounded-full bg-white/10 px-3.5 py-1.5 text-sm font-medium text-white/80 backdrop-blur-sm">
              <Star className="mr-1 inline h-3.5 w-3.5 fill-primary text-primary align-[-2px]" />
              {(image.score ?? 0).toFixed(2)}
            </span>
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
        className="relative max-h-[92vh] w-full max-w-7xl overflow-hidden rounded-2xl bg-card ring-1 ring-border/30 shadow-2xl animate-fade-in-scale"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute right-3 top-3 z-10 flex h-8 w-8 items-center justify-center rounded-lg bg-black/40 backdrop-blur-sm text-white/80 transition-colors hover:bg-black/60 hover:text-white"
        >
          <X className="h-4 w-4" />
        </button>

        <div className="flex flex-col md:flex-row">
          {/* Image area with nav buttons */}
          <div className="group/img relative flex-1 bg-black/10">
            <img
              key={image.id}
              src={image.thumbnail_url || image.url}
              alt={`Generated ${image.id}`}
              className="h-full w-full object-contain max-h-[85vh] animate-fade-in cursor-zoom-in"
              onClick={() => setLightboxOpen(true)}
              onError={(e) => {
                if (image.url && !e.target.dataset.fallback) {
                  e.target.dataset.fallback = "1";
                  e.target.src = image.url;
                }
              }}
            />
            {/* Zoom hint */}
            <div className="absolute bottom-3 right-3 flex items-center gap-1.5 rounded-full bg-black/40 px-2.5 py-1 text-[10px] text-white/60 opacity-0 transition-opacity group-hover/img:opacity-100 backdrop-blur-sm pointer-events-none">
              <ZoomIn className="h-3 w-3" />
              {t("modal.clickToZoom")}
            </div>

            {/* Left nav button */}
            {hasPrev && (
              <button
                onClick={goPrev}
                className="absolute left-2 top-1/2 -translate-y-1/2 flex h-10 w-10 items-center justify-center rounded-full bg-black/40 text-white/70 opacity-0 transition-all hover:bg-black/60 hover:text-white group-hover/img:opacity-100 backdrop-blur-sm"
              >
                <ChevronLeft className="h-5 w-5" />
              </button>
            )}

            {/* Right nav button */}
            {hasNext && (
              <button
                onClick={goNext}
                className="absolute right-2 top-1/2 -translate-y-1/2 flex h-10 w-10 items-center justify-center rounded-full bg-black/40 text-white/70 opacity-0 transition-all hover:bg-black/60 hover:text-white group-hover/img:opacity-100 backdrop-blur-sm"
              >
                <ChevronRight className="h-5 w-5" />
              </button>
            )}
          </div>

          <div className="w-full p-6 md:w-80 md:overflow-y-auto md:max-h-[92vh]">
            {/* Score header */}
            <div className="mb-6 flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-primary/10">
                <Star className="h-5 w-5 fill-primary text-primary" />
              </div>
              <div>
                <div className="text-2xl font-bold tabular-nums text-foreground font-display">{(image.score ?? 0).toFixed(2)}</div>
                <div className="text-[10px] uppercase tracking-[0.1em] text-muted-foreground font-medium">{t("modal.overallScore")}</div>
              </div>
            </div>

            {/* Score details */}
            {image.score_details && (
              <div className="space-y-3 mb-6">
                <h4 className="text-[10px] font-semibold uppercase tracking-[0.1em] text-muted-foreground">{t("modal.breakdown")}</h4>
                {Object.entries(image.score_details)
                  .filter(([key, value]) => value != null && scoreKeyMap[key])
                  .map(([key, value]) => (
                    <ScoreBar key={key} label={t(scoreKeyMap[key])} value={value} />
                  ))}
              </div>
            )}

            {/* Meta */}
            <div className="space-y-2.5 border-t border-border/30 pt-4">
              {[
                { label: t("modal.round"), value: `#${image.round_number}` },
                { label: t("modal.provider"), value: image.provider },
              ].map(({ label, value }) => (
                <div key={label} className="flex justify-between text-xs">
                  <span className="text-muted-foreground">{label}</span>
                  <span className="font-medium text-foreground">{value}</span>
                </div>
              ))}
            </div>

            {/* Edit */}
            {onEdit && (
              <button
                onClick={() => onEdit(image)}
                className="mt-5 flex w-full items-center justify-center gap-2 rounded-lg bg-secondary py-2 text-xs font-semibold text-secondary-foreground transition-all hover:brightness-110"
              >
                <Pencil className="h-3.5 w-3.5" />
                {t("modal.edit")}
              </button>
            )}

            {/* Download */}
            <button
              onClick={handleDownload}
              className={`${onEdit ? "mt-2" : "mt-5"} flex w-full items-center justify-center gap-2 rounded-lg bg-primary py-2 text-xs font-semibold text-primary-foreground transition-all hover:brightness-110`}
            >
              <Download className="h-3.5 w-3.5" />
              {t("modal.download")}
            </button>
          </div>
        </div>

        {/* Position indicator */}
        {images.length > 1 && (
          <div className="absolute bottom-3 left-1/2 -translate-x-1/2 rounded-full bg-black/50 px-3 py-1 text-xs tabular-nums text-white/80 backdrop-blur-sm">
            {currentIndex + 1} / {images.length}
          </div>
        )}
      </div>
    </div>,
    document.body
  );
}
