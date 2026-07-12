import { useState } from "react";
import { Download, Maximize2, Pencil, Star } from "lucide-react";
import { cacheImageAspectRatio, getImageAspectRatio } from "./aspectRatio";
import { ImageEditOverlay } from "./ImageEditOverlay";

export function ImageCard({ image, rank, onClick, aspectRatio = "16:9", evaluating = false, onEdit, isEditLoading = false }) {
  const displaySrc = image.thumbnail_url || image.url;
  const [imgSrc, setImgSrc] = useState(displaySrc);
  const [resolvedAspectRatio, setResolvedAspectRatio] = useState(() =>
    getImageAspectRatio(image, aspectRatio)
  );

  const handleImageLoad = (event) => {
    const { naturalWidth, naturalHeight } = event.currentTarget;
    if (!naturalWidth || !naturalHeight) return;
    const naturalAspectRatio =
      cacheImageAspectRatio(image?.url, naturalWidth, naturalHeight) ||
      `${naturalWidth} / ${naturalHeight}`;
    setResolvedAspectRatio((prev) => (prev === naturalAspectRatio ? prev : naturalAspectRatio));
  };

  const handleImgError = () => {
    if (imgSrc !== image.url) {
      setImgSrc(image.url);
    }
  };

  const handleDownload = async (e) => {
    e.stopPropagation();
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
    } catch (error) {
      console.error("Download failed:", error);
    }
  };

  return (
    <div
      className={`group relative cursor-zoom-in overflow-hidden rounded-xl border bg-card shadow-xl transition-all hover:shadow-2xl hover:border-primary/30 hover:-translate-y-1 ${
        evaluating ? "border-primary/35" : "border-border/30"
      }`}
      onClick={onClick}
    >
      <div style={{ aspectRatio: resolvedAspectRatio }}>
        <img
          src={imgSrc}
          alt={`Generated ${image.id}`}
          onLoad={handleImageLoad}
          onError={handleImgError}
          loading="lazy"
          decoding="async"
          className="h-full w-full object-cover transition-transform duration-700 group-hover:scale-105"
        />
      </div>

      {evaluating && (
        <div className="pointer-events-none absolute inset-0 overflow-hidden">
          <div className="absolute inset-0 bg-black/20" />
          <div className="absolute inset-y-0 -left-1/3 w-1/2 animate-shimmer opacity-80" />
        </div>
      )}

      <ImageEditOverlay visible={isEditLoading} />

      {/* Top overlay - Score badge */}
      <div className="absolute inset-x-0 top-0 flex items-start justify-between p-2.5">
        {/* Rank badge */}
        {rank !== undefined && rank < 3 && (
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-[11px] font-bold text-primary-foreground shadow-sm">
            {rank + 1}
          </div>
        )}
        {(rank === undefined || rank >= 3) && <div />}

        {/* Score */}
        <div className="flex items-center gap-1 rounded-md bg-black/40 backdrop-blur-md px-2 py-1 text-[11px] font-semibold tabular-nums text-white">
          <Star className="h-3 w-3 fill-primary text-primary" />
          {(image.score ?? 0).toFixed(2)}
        </div>
      </div>

      {/* Bottom overlay - on hover */}
      <div className="absolute inset-x-0 bottom-0 p-3 bg-gradient-to-t from-black/80 via-black/40 to-transparent opacity-0 group-hover:opacity-100 transition-opacity flex justify-between items-end translate-y-2 group-hover:translate-y-0 duration-300">
        <span className="text-[10px] font-mono text-zinc-300">#{rank !== undefined ? rank + 1 : image.id}</span>
        <div className="flex gap-1">
          {/* Edit button */}
          {onEdit && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onEdit(image);
              }}
              className="p-1.5 bg-white/10 backdrop-blur-md rounded-md hover:bg-white/20 text-white transition-colors"
            >
              <Pencil size={12} />
            </button>
          )}
          {/* Preview button */}
          <button
            onClick={(e) => {
              e.stopPropagation();
              onClick?.();
            }}
            className="p-1.5 bg-white/10 backdrop-blur-md rounded-md hover:bg-white/20 text-white transition-colors"
          >
            <Maximize2 size={12} />
          </button>
          {/* Download button */}
          <button
            onClick={handleDownload}
            className="p-1.5 bg-primary/90 backdrop-blur-md rounded-md hover:bg-primary text-primary-foreground font-medium transition-colors"
          >
            <Download size={12} />
          </button>
        </div>
      </div>
    </div>
  );
}
