import { useState, useCallback, useEffect, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { Download, Grid, Share2, Film, Loader2, Sun, Moon, Globe, Github } from "lucide-react";
import { TaskSidebar } from "@/components/layout/TaskSidebar";
import { TaskHeader } from "@/components/layout/TaskHeader";
import { GeneratingState } from "@/components/generation/GeneratingState";
import { FloatingControlPanel } from "@/components/generation/FloatingControlPanel";
import { ImageGallery } from "@/components/gallery/ImageGallery";
import { SaveAsTemplateButton } from "@/components/templates/SaveAsTemplateButton";
import { useGeneration } from "@/hooks/useGeneration";
import { useVideoGeneration } from "@/hooks/useVideoGeneration";
import { useTaskHistory } from "@/hooks/useTaskHistory";
import { useLocale } from "@/hooks/useLocale";
import { useImageEdit } from "@/hooks/useImageEdit";
import { useTheme } from "@/hooks/useTheme";
import { apiClient } from "@/lib/api";
import { toast } from "sonner";

export default function App() {
  const { t, locale, setLocale } = useLocale();
  const { theme, toggleTheme } = useTheme();
  const [searchParams, setSearchParams] = useSearchParams();
  const [mode, setMode] = useState("text_to_image");
  const [mediaType, setMediaType] = useState("image");
  const [providerCapabilities, setProviderCapabilities] = useState(null);

  useEffect(() => {
    apiClient.getProviderCapabilities().then(setProviderCapabilities);
  }, []);

  const disabledModes = useMemo(() => {
    if (!providerCapabilities) return [];
    const supported = providerCapabilities.supported_modes || [];
    const all = ["text_to_image", "image_to_image"];
    return all.filter((item) => !supported.includes(item));
  }, [providerCapabilities]);

  const [templateImages, setTemplateImages] = useState([]);
  const [config, setConfig] = useState({
    maxRounds: 3,
    imagesPerRound: 5,
    qualityThreshold: 0.7,
    aspectRatio: "16:9",
    size: "2K",
    enhancePrompt: true,
    enableAiScoring: true,
  });

  // Image generation hook
  const {
    status: imageStatus,
    progress: imageProgress,
    result: imageResult,
    error: imageError,
    generationConfig: imageGenerationConfig,
    streamedImages,
    taskContext: imageTaskContext,
    startGeneration,
    cancelGeneration: cancelImageGeneration,
    loadResult: loadImageResult,
    resumeTask: resumeImageTask,
    reset: resetImage,
  } = useGeneration();

  // Video generation hook
  const {
    status: videoStatus,
    progress: videoProgress,
    result: videoResult,
    error: videoError,
    videoConfig: videoGenerationConfig,
    streamedVideos,
    taskContext: videoTaskContext,
    startVideoGeneration,
    resumeVideoTask,
    cancelVideoGeneration,
    loadVideoResult,
    reset: resetVideo,
  } = useVideoGeneration();

  // Active state based on media type
  const isVideo = mediaType === "video";
  const status = isVideo ? videoStatus : imageStatus;
  const progress = isVideo ? videoProgress : imageProgress;
  const error = isVideo ? videoError : imageError;
  const generationConfig = isVideo ? videoGenerationConfig : imageGenerationConfig;
  const taskContext = isVideo ? videoTaskContext : imageTaskContext;

  const {
    tasks: historyTasks,
    selectedTaskId,
    loading: historyLoading,
    selectTask,
    clearSelection,
    deleteTask,
    refresh: refreshHistory,
  } = useTaskHistory();

  const imageEdit = useImageEdit(imageTaskContext?.taskId);

  const handleSelectHistoryTask = useCallback(
    (taskId) => {
      selectTask(taskId);
      const task = historyTasks.find((item) => item.task_id === taskId);
      if (!task) return;

      const runningStatuses = ["pending", "enhancing", "generating", "reviewing", "queued", "processing"];
      const isVideoTask = task.task_type === "video";

      if (isVideoTask) {
        // Handle video tasks
        setMediaType("video");
        if (runningStatuses.includes(task.status)) {
          resumeVideoTask(taskId, task);
          return;
        }
        loadVideoResult(taskId);
      } else {
        // Handle image tasks
        setMediaType("image");
        if (runningStatuses.includes(task.status)) {
          resumeImageTask(taskId, task);
          return;
        }
        loadImageResult(taskId);
      }
    },
    [selectTask, loadImageResult, resumeImageTask, loadVideoResult, resumeVideoTask, historyTasks, setMediaType]
  );

  const handleNewTask = useCallback(() => {
    resetImage();
    resetVideo();
    clearSelection();
    setTemplateImages([]);
    setMode("text_to_image");
  }, [resetImage, resetVideo, clearSelection]);

  const handleDeleteTask = useCallback(async (taskId) => {
    try {
      await deleteTask(taskId);
      if (taskId === selectedTaskId) {
        resetImage();
        resetVideo();
      }
      toast.success(t("sidebar.deleteSuccess"));
    } catch {
      toast.error(t("sidebar.deleteFailed"));
    }
  }, [deleteTask, selectedTaskId, resetImage, resetVideo, t]);

  useEffect(() => {
    if (imageStatus === "completed" || imageStatus === "failed" ||
        videoStatus === "completed" || videoStatus === "failed") {
      refreshHistory();
    }
  }, [imageStatus, videoStatus, refreshHistory]);

  const [initialPrompt, setInitialPrompt] = useState("");

  useEffect(() => {
    const taskId = searchParams.get("resumeTask");
    if (taskId) {
      searchParams.delete("resumeTask");
      setSearchParams(searchParams, { replace: true });
      resumeImageTask(taskId);
      return;
    }
    const templatePrompt = searchParams.get("templatePrompt");
    if (templatePrompt) {
      searchParams.delete("templatePrompt");
      setSearchParams(searchParams, { replace: true });
      setInitialPrompt(templatePrompt);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleModeChange = (newMode) => {
    if (disabledModes.includes(newMode)) return;
    setMode(newMode);
    if (newMode === "text_to_image") setTemplateImages([]);
  };

  const handleMediaTypeChange = (newMediaType) => {
    setMediaType(newMediaType);
    if (newMediaType === "video") {
      setMode("text_to_video");
    } else {
      setMode("text_to_image");
    }
  };

  const handleGenerate = async (prompt) => {
    const currentTemplateImages = mode === "image_to_image" ? templateImages : [];
    setTemplateImages([]);

    await startGeneration({
      prompt,
      template_images: currentTemplateImages,
      max_rounds: config.maxRounds,
      images_per_round: config.imagesPerRound,
      quality_threshold: config.qualityThreshold,
      aspect_ratio: config.aspectRatio,
      size: config.size,
      enhance_prompt: config.enhancePrompt,
      enable_ai_scoring: config.enableAiScoring,
      provider: providerCapabilities?.name || undefined,
    });
  };

  const handleVideoGenerate = async (request) => {
    await startVideoGeneration(request);
  };

  const isGenerating = status === "creating" || status === "running" || status === "completing";
  const canGenerate = mode === "text_to_image" || templateImages.length > 0;
  const hasResults = isVideo
    ? videoResult && (videoResult.videos?.length > 0 || videoResult.rounds?.length > 0)
    : imageResult && (imageResult.best_images?.length > 0 || imageResult.rounds?.length > 0);

  const shouldShowRounds = useMemo(() => {
    if (isVideo) {
      if (!videoResult?.rounds?.length) return false;
      return videoResult.rounds.length > 1;
    }
    if (!imageResult?.rounds?.length) return false;
    if (!imageResult?.best_images?.length) return true;
    if (imageResult.rounds.length > 1) return true;

    const round = imageResult.rounds[0];
    if (!round?.images?.length) return false;
    if (round.images.length !== imageResult.best_images.length) return true;

    const bestIds = new Set(imageResult.best_images.map((img) => img.id));
    const allInBest = round.images.every((img) => bestIds.has(img.id));
    return !allInBest;
  }, [imageResult, videoResult, isVideo]);

  return (
    <div className="flex h-screen w-full overflow-hidden bg-background text-foreground">
      <TaskSidebar
        tasks={historyTasks}
        selectedTaskId={selectedTaskId}
        loading={historyLoading}
        onSelect={handleSelectHistoryTask}
        onRefresh={refreshHistory}
        onNewTask={handleNewTask}
        onDelete={handleDeleteTask}
      />

      <main className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_0%,rgba(255,165,60,0.18),transparent_34%),radial-gradient(circle_at_88%_5%,rgba(255,125,42,0.14),transparent_30%)] opacity-40 dark:opacity-100" />
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_bottom,transparent_0%,rgba(0,0,0,0.18)_100%)] dark:opacity-100 opacity-30" />

        <div className="relative z-10 flex h-full flex-col overflow-y-auto px-4 pb-56 pt-5 sm:px-6 lg:px-8">
          <div className="mb-3 flex items-center justify-end gap-1.5">
            <button
              onClick={() => setLocale(locale === "en" ? "zh" : "en")}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-all duration-200 hover:bg-secondary hover:text-foreground"
              title={t("lang.toggle")}
            >
              <Globe className="h-[15px] w-[15px]" />
            </button>
            <a
              href="https://github.com/NanmiCoder/PicTacticAgent"
              target="_blank"
              rel="noopener noreferrer"
              className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-all duration-200 hover:bg-secondary hover:text-foreground"
              title="GitHub"
            >
              <Github className="h-[15px] w-[15px]" />
            </a>
            <button
              onClick={toggleTheme}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-all duration-200 hover:bg-secondary hover:text-foreground"
            >
              {theme === "dark" ? (
                <Sun className="h-[15px] w-[15px]" />
              ) : (
                <Moon className="h-[15px] w-[15px]" />
              )}
            </button>
          </div>

          <TaskHeader
            status={status}
            taskContext={taskContext}
            onNewTask={handleNewTask}
            selectedTaskId={selectedTaskId}
          />

          {!hasResults && !isGenerating && status === "idle" ? (
            <section className="flex flex-1 items-center justify-center py-10">
              <div className="brand-card w-full max-w-xl p-8 text-center">
                <div className="mx-auto mb-5 flex h-20 w-20 items-center justify-center rounded-3xl bg-secondary text-muted-foreground">
                  {isVideo ? <Film className="h-9 w-9" /> : <Grid className="h-9 w-9" />}
                </div>
                <h2 className="text-2xl font-semibold tracking-tight text-foreground">{t("empty.readyTitle")}</h2>
                <p className="mx-auto mt-3 max-w-md text-sm leading-relaxed text-muted-foreground">
                  {isVideo
                    ? t("empty.readyText")
                    : (mode === "text_to_image" ? t("empty.readyText") : t("empty.readyImageText"))
                  }
                </p>
              </div>
            </section>
          ) : null}

          {status === "loading" ? (
            <section className="flex flex-1 items-center justify-center py-10">
              <div className="flex flex-col items-center gap-3 text-muted-foreground">
                <Loader2 className="h-8 w-8 animate-spin" />
                <p className="text-sm">{t("progress.loading")}</p>
              </div>
            </section>
          ) : null}

          {(isGenerating || error) && generationConfig ? (
            <div className="flex flex-1 flex-col items-center justify-start pt-2">
              <GeneratingState
                progress={progress}
                streamedImages={streamedImages}
                streamedVideos={streamedVideos}
                config={generationConfig}
                onCancel={status !== "completing"
                  ? (isVideo ? cancelVideoGeneration : cancelImageGeneration)
                  : undefined
                }
                error={error}
                isCompleting={status === "completing"}
                onRetry={error && !isGenerating ? () => (isVideo ? resetVideo() : resetImage()) : undefined}
                taskContext={taskContext}
                mediaType={mediaType}
              />
            </div>
          ) : null}

          {hasResults && !isGenerating ? (
            <div className="animate-fade-in w-full max-w-7xl self-center">
              {taskContext ? (
                <div className="mb-6 flex items-start gap-3 rounded-2xl border border-border/65 bg-card/75 p-4 backdrop-blur-xl">
                  {!isVideo && taskContext.templateImages?.length > 0 ? (
                    <div className="flex flex-shrink-0 gap-1.5">
                      {taskContext.templateImages.slice(0, 3).map((src, index) => (
                        <img
                          key={`${src.slice(0, 12)}-${index}`}
                          src={src}
                          alt=""
                          className="h-12 w-12 rounded-lg border border-border/60 object-cover"
                        />
                      ))}
                      {taskContext.templateImages.length > 3 ? (
                        <div className="flex h-12 w-12 items-center justify-center rounded-lg border border-border/60 bg-secondary text-xs text-muted-foreground">
                          +{taskContext.templateImages.length - 3}
                        </div>
                      ) : null}
                    </div>
                  ) : null}

                  {isVideo && taskContext.startImage ? (
                    <div className="flex flex-shrink-0 gap-1.5">
                      <img
                        src={taskContext.startImage}
                        alt="Start frame"
                        className="h-12 w-12 rounded-lg border border-border/60 object-cover"
                      />
                    </div>
                  ) : null}

                  <div className="min-w-0 flex-1">
                    <p className="line-clamp-2 text-sm leading-relaxed text-foreground/90">{taskContext.prompt}</p>
                    <div className="mt-2 flex flex-wrap items-center gap-1.5">
                      <span className="rounded-full border border-border/70 bg-secondary/70 px-2 py-0.5 text-[10px] text-muted-foreground">
                        {isVideo
                          ? (taskContext.mode === "image_to_video" ? t("mode.imageToVideo") : t("mode.textToVideo"))
                          : (taskContext.mode === "image_to_image" ? t("mode.imageEdit") : t("mode.textToImage"))
                        }
                      </span>
                      {isVideo && taskContext.provider && (
                        <span className="rounded-full border border-border/70 bg-secondary/70 px-2 py-0.5 text-[10px] text-muted-foreground capitalize">
                          {taskContext.provider}
                        </span>
                      )}
                      {!isVideo && (
                        <>
                          <span className="rounded-full border border-border/70 bg-secondary/70 px-2 py-0.5 text-[10px] text-muted-foreground">
                            {taskContext.aspectRatio}
                          </span>
                          <span className="rounded-full border border-border/70 bg-secondary/70 px-2 py-0.5 text-[10px] text-muted-foreground">
                            {taskContext.size}
                          </span>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              ) : null}

              <div className="mb-6 flex items-center justify-between px-1">
                <div className="flex items-center gap-3">
                  <h3 className="text-xl font-semibold text-foreground">
                    {isVideo ? t("video.completed") : t("gallery.bestResults")}
                  </h3>
                  {!isVideo && (
                    <span className="rounded-full border border-border/70 bg-card/75 px-2.5 py-0.5 text-[10px] text-muted-foreground">
                      {config.aspectRatio} &bull; {config.size}
                    </span>
                  )}
                </div>

                <div className="flex gap-2">
                  {!isVideo && taskContext?.prompt && (
                    <SaveAsTemplateButton
                      prompt={taskContext.prompt}
                      config={imageGenerationConfig}
                      previewImageUrl={imageResult?.best_images?.[0]?.thumbnail_url || imageResult?.best_images?.[0]?.url || null}
                    />
                  )}
                  <button className="rounded-xl border border-border/65 bg-card/70 p-2 text-muted-foreground transition-colors hover:text-foreground">
                    <Share2 size={16} />
                  </button>
                  <button className="rounded-xl border border-border/65 bg-card/70 p-2 text-muted-foreground transition-colors hover:text-foreground">
                    <Download size={16} />
                  </button>
                </div>
              </div>

              {isVideo ? (
                <>
                  {videoResult?.videos?.length > 0 ? (
                    <ImageGallery
                      videos={videoResult.videos}
                      title={t("video.completed")}
                      aspectRatio={videoGenerationConfig?.aspectRatio || "16:9"}
                      mediaType="video"
                    />
                  ) : null}

                  {shouldShowRounds && videoResult?.rounds ? (
                    <div className="mt-8 space-y-8 pb-4">
                      {videoResult.rounds.map((round) => (
                        <ImageGallery
                          key={round.round_number}
                          videos={round.videos}
                          title={t("gallery.round", {
                            number: round.round_number,
                            score: "",
                          })}
                          aspectRatio={videoGenerationConfig?.aspectRatio || "16:9"}
                          mediaType="video"
                        />
                      ))}
                    </div>
                  ) : null}
                </>
              ) : (
                <>
                  {imageResult?.best_images?.length > 0 ? (
                    <ImageGallery
                      images={imageResult.best_images}
                      title={t("gallery.bestResults")}
                      aspectRatio={config.aspectRatio}
                      imageEdit={imageEdit}
                      enhancedPrompt={imageResult.enhanced_prompt}
                    />
                  ) : null}

                  {shouldShowRounds ? (
                    <div className="mt-8 space-y-8 pb-4">
                      {imageResult.rounds.map((round) => (
                        <ImageGallery
                          key={round.round_number}
                          images={round.images}
                          title={t("gallery.round", {
                            number: round.round_number,
                            score: (round.best_score ?? 0).toFixed(2),
                          })}
                          aspectRatio={config.aspectRatio}
                          imageEdit={imageEdit}
                          enhancedPrompt={imageResult.enhanced_prompt}
                        />
                      ))}
                    </div>
                  ) : null}
                </>
              )}
            </div>
          ) : null}
        </div>

        <FloatingControlPanel
          config={config}
          onConfigChange={setConfig}
          onGenerate={handleGenerate}
          onVideoGenerate={handleVideoGenerate}
          isGenerating={isGenerating}
          mode={mode}
          onModeChange={handleModeChange}
          disabledModes={disabledModes}
          canGenerate={canGenerate}
          templateImages={templateImages}
          onTemplateImagesChange={setTemplateImages}
          mediaType={mediaType}
          onMediaTypeChange={handleMediaTypeChange}
          activeVideoProvider={videoTaskContext?.provider}
          initialPrompt={initialPrompt}
        />
      </main>
    </div>
  );
}
