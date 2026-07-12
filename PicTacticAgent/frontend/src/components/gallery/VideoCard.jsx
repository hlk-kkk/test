import { useRef, useCallback } from "react";
import { Download, Film, Maximize2, Play } from "lucide-react";

export function VideoCard({ video, aspectRatio = "16:9", rank, onClick }) {
  const videoRef = useRef(null);

  const handleMouseEnter = useCallback(() => {
    if (videoRef.current) {
      videoRef.current.play().catch(() => {
        // Autoplay may be blocked, ignore
      });
    }
  }, []);

  const handleMouseLeave = useCallback(() => {
    if (videoRef.current) {
      videoRef.current.pause();
      videoRef.current.currentTime = 0;
    }
  }, []);

  const handleDownload = async (e) => {
    e.stopPropagation();
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
    } catch (error) {
      console.error("Download failed:", error);
    }
  };

  const cssRatio = aspectRatio.replace(":", " / ");

  return (
    <div
      className="group relative cursor-pointer overflow-hidden rounded-xl border border-border/30 bg-card shadow-xl transition-all hover:shadow-2xl hover:border-primary/30 hover:-translate-y-1"
      onClick={() => onClick?.(video)}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
      <div style={{ aspectRatio: cssRatio }} className="relative">
        <video
          ref={videoRef}
          src={video.url}
          poster={video.start_image || undefined}
          muted
          loop
          playsInline
          preload="metadata"
          className="h-full w-full object-cover transition-transform duration-700 group-hover:scale-105"
        />

        {/* Play icon overlay (hidden on hover when video plays) */}
        <div className="absolute inset-0 flex items-center justify-center bg-black/20 transition-opacity group-hover:opacity-0">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-black/50 backdrop-blur-sm">
            <Play className="h-4 w-4 text-white" fill="white" />
          </div>
        </div>
      </div>

      {/* Top overlay - Round badge and duration */}
      <div className="absolute inset-x-0 top-0 flex items-start justify-between p-2.5">
        {/* Round badge */}
        {rank !== undefined && rank < 3 ? (
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-primary text-[11px] font-bold text-primary-foreground shadow-sm">
            {rank + 1}
          </div>
        ) : (
          <div />
        )}

        {/* Duration badge */}
        {video.duration != null && (
          <div className="flex items-center gap-1 rounded-md bg-black/40 px-2 py-1 text-[11px] font-semibold tabular-nums text-white backdrop-blur-md">
            <Film className="h-3 w-3" />
            {video.duration}s
          </div>
        )}
      </div>

      {/* Bottom overlay - on hover */}
      <div className="absolute inset-x-0 bottom-0 flex translate-y-2 items-end justify-between bg-gradient-to-t from-black/80 via-black/40 to-transparent p-3 opacity-0 transition-opacity duration-300 group-hover:translate-y-0 group-hover:opacity-100">
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] font-mono text-zinc-300">
            {video.provider || ""}
          </span>
          {video.round_number != null && (
            <span className="rounded bg-white/10 px-1.5 py-0.5 text-[10px] text-zinc-300">
              R{video.round_number}
            </span>
          )}
        </div>
        <div className="flex gap-1">
          {/* Preview button */}
          <button
            onClick={(e) => {
              e.stopPropagation();
              onClick?.(video);
            }}
            className="rounded-md bg-white/10 p-1.5 text-white backdrop-blur-md transition-colors hover:bg-white/20"
          >
            <Maximize2 size={12} />
          </button>
          {/* Download button */}
          <button
            onClick={handleDownload}
            className="rounded-md bg-primary/90 p-1.5 font-medium text-primary-foreground backdrop-blur-md transition-colors hover:bg-primary"
          >
            <Download size={12} />
          </button>
        </div>
      </div>
    </div>
  );
}
