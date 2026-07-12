import { useState } from "react";
import ReactMarkdown from "react-markdown";
import { X, Zap, ChevronDown, ChevronUp, AlertTriangle, RefreshCw, Film } from "lucide-react";
import { cn } from "@/lib/utils";
import { ImageCard } from "@/components/gallery/ImageCard";
import { ImageSkeleton } from "@/components/gallery/ImageSkeleton";
import { ImageModal } from "@/components/gallery/ImageModal";
import { VideoCard } from "@/components/gallery/VideoCard";
import { VideoModal } from "@/components/gallery/VideoModal";
import { MasonryGrid } from "@/components/gallery/MasonryGrid";
import { useLocale } from "@/hooks/useLocale";

function StatusIndicator() {
  return (
    <span className="relative flex h-2 w-2">
      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary opacity-60" />
      <span className="relative inline-flex h-2 w-2 rounded-full bg-primary" />
    </span>
  );
}

function ErrorIndicator() {
  return (
    <span className="relative flex h-2 w-2">
      <span className="relative inline-flex h-2 w-2 rounded-full bg-red-500" />
    </span>
  );
}

function VideoProgressBar({ progress }) {
  const { t } = useLocale();
  if (!progress?.current_video_progress) return null;

  return (
    <div className="mb-4 rounded-xl border border-border/50 bg-secondary/30 p-3">
      <div className="mb-1.5 flex items-center justify-between text-xs">
        <span className="flex items-center gap-1.5 text-muted-foreground">
          <Film className="h-3 w-3" />
          {t("video.polling", { index: (progress.current_video_index || 0) + 1 })}
        </span>
        <span className="font-mono text-primary">
          {t("video.progress", {
            percent: Math.round(progress.current_video_progress),
            eta: progress.current_video_eta || "--",
          })}
        </span>
      </div>
      <div className="h-1.5 overflow-hidden rounded-full bg-secondary/80">
        <div
          className="h-full rounded-full bg-gradient-to-r from-primary via-amber-400 to-primary transition-all duration-500 ease-out"
          style={{ width: `${Math.max(3, progress.current_video_progress)}%` }}
        />
      </div>
    </div>
  );
}

export function GeneratingState({
  progress,
  streamedImages,
  streamedVideos,
  config,
  onCancel,
  error,
  isCompleting = false,
  onRetry,
  taskContext,
  mediaType = "image",
}) {
  const { t } = useLocale();
  const [showEnhanced, setShowEnhanced] = useState(false);
  const [selectedImageIndex, setSelectedImageIndex] = useState(null);
  const [selectedVideo, setSelectedVideo] = useState(null);

  const isVideo = mediaType === "video";

  const getStatusText = () => {
    if (error) return error;
    if (isCompleting) return t("progress.completing");
    if (progress?.message) return progress.message;
    return isVideo ? t("video.generating") : t("progress.generating");
  };

  const progressPercent = (() => {
    if (!progress) return 0;

    if (isVideo) {
      const rounds = Math.max(1, Number(progress.max_rounds) || 1);
      if (isCompleting || progress.status === "completed") return 100;
      const videosGenerated = Math.max(0, Number(progress.videos_generated) || 0);
      const videosPerRound = Math.max(1, config?.videosPerRound || 2);
      const totalVideos = rounds * videosPerRound;
      return Math.min(99, (videosGenerated / totalVideos) * 100);
    }

    const rounds = Math.max(1, Number(progress.max_rounds) || 1);
    const imagesPerRound = Math.max(1, config.imagesPerRound || 1);
    const stepsPerRound = imagesPerRound + 1;
    const totalSteps = rounds * stepsPerRound;

    if (isCompleting || progress.status === "completed") return 100;

    const currentRound = Math.max(0, Number(progress.current_round) || 0);
    const generatedInRound = Math.min(
      imagesPerRound,
      Math.max(0, Number(progress.images_in_current_round) || 0)
    );

    let completedRounds = Math.max(0, currentRound - 1);
    let currentRoundSteps = generatedInRound;

    if (progress.status === "reviewing") {
      completedRounds = currentRound;
      currentRoundSteps = 0;
    } else if (progress.status === "enhancing" && currentRound > 0) {
      completedRounds = currentRound;
      currentRoundSteps = 0;
    }

    const doneSteps = completedRounds * stepsPerRound + currentRoundSteps;
    return Math.min(99, (doneSteps / totalSteps) * 100);
  })();

  const skeletonCount = isVideo
    ? Math.max(0, (config?.videosPerRound || 2) - (streamedVideos?.length || 0))
    : Math.max(0, config.imagesPerRound - streamedImages.length);

  const isReviewing = progress?.status === "reviewing";

  return (
    <div className="w-full max-w-6xl mx-auto animate-fade-in">
      <div className="flex items-center justify-between mb-4 px-1">
        <div className="flex items-center gap-3">
          {error ? <ErrorIndicator /> : <StatusIndicator />}
          <span className={cn(
            "text-sm font-medium",
            error ? "text-red-400" : "text-foreground"
          )}>{getStatusText()}</span>
        </div>
        <div className="flex items-center gap-4">
          {progress && (
            <div className="flex items-center gap-3 text-[11px] tabular-nums text-muted-foreground">
              <span className="font-mono">
                R{progress.current_round}/{progress.max_rounds}
              </span>
              {isVideo ? (
                <>
                  <span className="text-border">|</span>
                  <span>{t("video.totalVideos", { count: progress.videos_generated || 0 })}</span>
                </>
              ) : (
                <>
                  <span className="text-border">|</span>
                  <span>{t("progress.imgs", { count: progress.images_generated })}</span>
                  {progress.current_best_score > 0 && (
                    <>
                      <span className="text-border">|</span>
                      <span className="text-primary font-semibold">
                        {progress.current_best_score.toFixed(2)}
                      </span>
                    </>
                  )}
                </>
              )}
            </div>
          )}
          {onCancel && (
            <button
              onClick={onCancel}
              className="flex h-7 items-center gap-1.5 px-2.5 rounded-lg text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors border border-border/50"
            >
              <X className="h-3.5 w-3.5" />
              <span>{t("progress.cancel")}</span>
            </button>
          )}
        </div>
      </div>

      {taskContext && (
        <div className="flex items-start gap-3 mb-4 p-3 rounded-xl bg-secondary/50 border border-border/50">
          {taskContext.templateImages?.length > 0 && (
            <div className="flex gap-1.5 flex-shrink-0">
              {taskContext.templateImages.slice(0, 3).map((src, i) => (
                <img
                  key={i}
                  src={src}
                  alt=""
                  className="h-12 w-12 rounded-lg object-cover ring-1 ring-border/50"
                />
              ))}
              {taskContext.templateImages.length > 3 && (
                <div className="h-12 w-12 rounded-lg bg-secondary flex items-center justify-center text-xs text-muted-foreground ring-1 ring-border/50">
                  +{taskContext.templateImages.length - 3}
                </div>
              )}
            </div>
          )}

          {isVideo && taskContext.startImage && (
            <div className="flex gap-1.5 flex-shrink-0">
              <img
                src={taskContext.startImage}
                alt="Start frame"
                className="h-12 w-12 rounded-lg object-cover ring-1 ring-border/50"
              />
            </div>
          )}

          <div className="flex-1 min-w-0">
            <p className="text-sm text-muted-foreground line-clamp-2 leading-relaxed">
              {taskContext.prompt}
            </p>
            <div className="flex items-center gap-2 mt-1.5">
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-secondary text-muted-foreground">
                {isVideo
                  ? (taskContext.mode === "image_to_video" ? t("mode.imageToVideo") : t("mode.textToVideo"))
                  : (taskContext.mode === "image_to_image" ? t("mode.imageEdit") : t("mode.textToImage"))
                }
              </span>
              {isVideo && taskContext.provider && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-secondary text-muted-foreground capitalize">
                  {taskContext.provider}
                </span>
              )}
              {!isVideo && taskContext.aspectRatio && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-secondary text-muted-foreground">
                  {taskContext.aspectRatio}
                </span>
              )}
              {!isVideo && taskContext.size && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-secondary text-muted-foreground">
                  {taskContext.size}
                </span>
              )}
            </div>
          </div>
        </div>
      )}

      {!error && (
        <div className="mb-6">
          <div className="h-1.5 overflow-hidden rounded-full bg-secondary/80">
            <div
              className={cn(
                "h-full rounded-full transition-all duration-700 ease-out",
                "bg-gradient-to-r from-primary via-amber-400 to-primary"
              )}
              style={{ width: `${Math.max(3, progressPercent)}%` }}
            />
          </div>
        </div>
      )}

      {error ? (
        <div className="flex flex-col items-center justify-center py-12">
          <div className="mb-6 inline-flex h-20 w-20 items-center justify-center rounded-2xl bg-red-500/10 border border-red-500/20">
            <AlertTriangle className="h-10 w-10 text-red-400" />
          </div>
          <h3 className="text-lg font-medium text-foreground mb-2">
            {t("progress.failed")}
          </h3>
          <p className="text-sm text-muted-foreground max-w-md text-center mb-6">
            {error}
          </p>
          {onRetry && (
            <button
              onClick={onRetry}
              className="flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <RefreshCw className="h-4 w-4" />
              {t("progress.retry")}
            </button>
          )}
        </div>
      ) : (
        <>
          {progress?.enhanced_prompt && (
            <div className="mb-6">
              <button
                onClick={() => setShowEnhanced(!showEnhanced)}
                className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-[0.1em] text-primary hover:text-primary/80 transition-colors"
              >
                <Zap className="h-3 w-3" />
                {t("progress.enhancedPrompt")}
                {showEnhanced ? (
                  <ChevronUp className="h-3 w-3" />
                ) : (
                  <ChevronDown className="h-3 w-3" />
                )}
              </button>
              {showEnhanced && (
                <div className="mt-2 rounded-lg bg-secondary/30 p-3 ring-1 ring-border/20 animate-fade-in">
                  <div className="prose-prompt text-xs text-foreground/80 leading-relaxed">
                    <ReactMarkdown>{progress.enhanced_prompt}</ReactMarkdown>
                  </div>
                </div>
              )}
            </div>
          )}

          {isVideo && progress?.current_video_progress && (
            <VideoProgressBar progress={progress} />
          )}

          {isVideo ? (
            <>
              <MasonryGrid
                className="stagger-children columns-1 gap-4 sm:columns-2 lg:columns-3"
                itemClassName="mb-4"
              >
                {(streamedVideos || []).map((video) => (
                  <div key={video.id} className="animate-fade-in-scale">
                    <VideoCard
                      video={video}
                      aspectRatio={config?.aspectRatio || "16:9"}
                      onClick={(v) => setSelectedVideo(v)}
                    />
                  </div>
                ))}
                {Array.from({ length: skeletonCount }).map((_, i) => (
                  <ImageSkeleton
                    key={`skel-${i}`}
                    aspectRatio={config?.aspectRatio || "16:9"}
                  />
                ))}
              </MasonryGrid>

              {selectedVideo && (
                <VideoModal
                  video={selectedVideo}
                  isOpen={true}
                  onClose={() => setSelectedVideo(null)}
                />
              )}
            </>
          ) : (
            <>
              <MasonryGrid
                className="stagger-children columns-2 gap-4 md:columns-3 xl:columns-4"
                itemClassName="mb-4"
              >
                {streamedImages.map((img, i) => (
                  <div key={img.id} className="animate-fade-in-scale">
                    <ImageCard
                      image={img}
                      aspectRatio={config.aspectRatio}
                      evaluating={isReviewing}
                      onClick={() => setSelectedImageIndex(i)}
                    />
                  </div>
                ))}
                {Array.from({ length: skeletonCount }).map((_, i) => (
                  <ImageSkeleton
                    key={`skel-${i}`}
                    aspectRatio={config.aspectRatio}
                  />
                ))}
              </MasonryGrid>

              {selectedImageIndex !== null && (
                <ImageModal
                  images={streamedImages}
                  currentIndex={selectedImageIndex}
                  onNavigate={setSelectedImageIndex}
                  onClose={() => setSelectedImageIndex(null)}
                />
              )}
            </>
          )}

          <div className="mt-6 text-center">
            <span className="font-mono text-xs tracking-widest text-primary/80 uppercase">
              {isCompleting
                ? t("progress.completing")
                : isVideo
                  ? `${t("video.generating")} ${(streamedVideos || []).length} / ${config?.videosPerRound || 2}`
                  : isReviewing
                    ? t("progress.reviewing")
                    : `${t("progress.generating")} ${streamedImages.length} / ${config.imagesPerRound}`}
            </span>
          </div>
        </>
      )}
    </div>
  );
}
