import { useEffect, useState, useCallback } from "react";
import { createPortal } from "react-dom";
import { X, ChevronLeft, ChevronRight, Download } from "lucide-react";

export function TemplatePreviewModal({ template, onClose }) {
  const images = template.all_image_urls?.length
    ? template.all_image_urls
    : template.preview_image_url
      ? [template.preview_image_url]
      : [];

  const [currentIndex, setCurrentIndex] = useState(0);

  const hasPrev = currentIndex > 0;
  const hasNext = currentIndex < images.length - 1;

  const goPrev = useCallback(() => {
    if (hasPrev) setCurrentIndex((i) => i - 1);
  }, [hasPrev]);

  const goNext = useCallback(() => {
    if (hasNext) setCurrentIndex((i) => i + 1);
  }, [hasNext]);

  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === "Escape") onClose();
      else if (e.key === "ArrowLeft") goPrev();
      else if (e.key === "ArrowRight") goNext();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose, goPrev, goNext]);

  const handleDownload = async () => {
    try {
      const response = await fetch(images[currentIndex]);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `template-${template.id}-${currentIndex + 1}.jpg`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Download failed:", err);
    }
  };

  if (!images.length) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/90 backdrop-blur-sm"
      onClick={onClose}
    >
      {/* Image */}
      <img
        src={images[currentIndex]}
        alt={template.title}
        className="max-h-[90vh] max-w-[90vw] object-contain animate-fade-in select-none"
        draggable={false}
        onClick={(e) => e.stopPropagation()}
      />

      {/* Top bar: title + counter */}
      <div className="absolute top-4 left-1/2 -translate-x-1/2 flex items-center gap-3">
        <span className="rounded-full bg-white/10 px-4 py-1.5 text-sm text-white/90 backdrop-blur-sm font-medium truncate max-w-[60vw]">
          {template.title}
          {images.length > 1 && (
            <span className="ml-2 text-white/60 tabular-nums">
              {currentIndex + 1} / {images.length}
            </span>
          )}
        </span>
      </div>

      {/* Close button */}
      <button
        onClick={(e) => { e.stopPropagation(); onClose(); }}
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

      {/* Bottom bar: download */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2">
        <button
          onClick={(e) => { e.stopPropagation(); handleDownload(); }}
          className="flex items-center gap-2 rounded-full bg-white/10 px-4 py-2 text-sm text-white/80 transition-colors hover:bg-white/20 hover:text-white backdrop-blur-sm"
        >
          <Download className="h-4 w-4" />
        </button>
      </div>
    </div>,
    document.body
  );
}
