import { useState, useCallback, useEffect, useRef } from "react";
import {
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Film,
  Image as ImageIcon,
  Layers,
  Loader2,
  Plus,
  Sliders,
  Type,
  Video,
  Wand2,
  X,
  Zap,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";
import { VideoConfigPanel } from "@/components/generation/VideoConfigPanel";
import { apiClient } from "@/lib/api";

const RATIO_OPTIONS = [
  { label: "1:1", value: "1:1", w: 12, h: 12 },
  { label: "16:9", value: "16:9", w: 14, h: 8 },
  { label: "9:16", value: "9:16", w: 8, h: 14 },
  { label: "4:3", value: "4:3", w: 13, h: 10 },
  { label: "3:4", value: "3:4", w: 10, h: 13 },
];

const RESOLUTION_OPTIONS = ["1K", "2K", "4K"];

// Transform flat API provider info into the nested capabilities shape
// that VideoConfigPanel expects
function transformProviders(list) {
  return list.map((p) => ({
    ...p,
    capabilities: p.capabilities || {
      supports_audio: p.supports_audio || false,
      supports_end_frame: p.supports_end_frame || false,
      supports_resolution: (p.resolutions || []).length > 0,
      resolution_options: p.resolutions || [],
      supports_aspect_ratio: (p.aspect_ratios || []).length > 0,
      aspect_ratio_options: p.aspect_ratios || [],
      ...(p.duration_type === "range"
        ? { duration_range: { min: p.duration_min, max: p.duration_max, step: 1 } }
        : { duration_options: p.duration_values || [] }),
    },
  }));
}

function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = (event) => resolve(event.target?.result);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

const Tooltip = ({ children, text }) => (
  <div className="group relative flex items-center justify-center">
    {children}
    <div className="pointer-events-none absolute -top-9 z-50 hidden whitespace-nowrap rounded-lg border border-border/70 bg-card px-2 py-1 text-xs text-foreground shadow-xl group-hover:block">
      {text}
    </div>
  </div>
);

const RangeSlider = ({ label, value, min, max, step, onChange }) => (
  <div className="space-y-2">
    <div className="flex items-center justify-between text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono font-semibold text-primary">
        {typeof value === "number" && step < 1 ? value.toFixed(2) : value}
      </span>
    </div>
    <input
      type="range"
      min={min}
      max={max}
      step={step}
      value={value}
      onChange={(e) => onChange(step < 1 ? parseFloat(e.target.value) : parseInt(e.target.value, 10))}
      className="custom-range w-full"
    />
  </div>
);

export function FloatingControlPanel({
  config,
  onConfigChange,
  onGenerate,
  onVideoGenerate,
  isGenerating = false,
  mode = "text_to_image",
  onModeChange,
  disabledModes = [],
  canGenerate = true,
  templateImages = [],
  onTemplateImagesChange,
  maxTemplateImages = 5,
  mediaType = "image",
  onMediaTypeChange,
  activeVideoProvider,
  initialPrompt = "",
}) {
  const [prompt, setPrompt] = useState(initialPrompt);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const { t } = useLocale();

  useEffect(() => {
    if (initialPrompt) setPrompt(initialPrompt);
  }, [initialPrompt]);

  // Video-specific state
  const [videoProvider, setVideoProvider] = useState("kling");
  const [videoProviders, setVideoProviders] = useState([]);
  const providerUserSelectedRef = useRef(false);
  const [videoConfig, setVideoConfig] = useState({
    duration: 5,
    resolution: null,
    aspectRatio: "16:9",
    audio: true,
    maxRounds: 1,
    videosPerRound: 2,
    enhancePrompt: true,
  });
  const [startImage, setStartImage] = useState(null);
  const [endImage, setEndImage] = useState(null);

  const handleProviderChange = useCallback((name) => {
    providerUserSelectedRef.current = true;
    setVideoProvider(name);
  }, []);

  // Fetch video providers on mount
  useEffect(() => {
    apiClient.listVideoProviders().then((providers) => {
      if (Array.isArray(providers) && providers.length > 0) {
        const transformed = transformProviders(providers);
        setVideoProviders(transformed);
        // Only set default if user hasn't already picked a provider
        if (!providerUserSelectedRef.current) {
          setVideoProvider(transformed[0].name);
        }
      }
    }).catch(() => {
      // Video providers not available, fallback to defaults
      setVideoProviders([
        { name: "kling", display_name: "Kling", capabilities: { duration_range: { min: 3, max: 15, step: 1 }, supports_audio: true, supports_end_frame: true, supports_aspect_ratio: true, aspect_ratio_options: ["16:9", "9:16", "1:1"] } },
        { name: "sora", display_name: "Sora", capabilities: { duration_options: [4, 8, 12], supports_resolution: true, resolution_options: ["720p", "1080p"], supports_aspect_ratio: true, aspect_ratio_options: ["16:9", "9:16"] } },
        { name: "veo", display_name: "Veo", capabilities: { duration_options: [4, 6, 8], supports_audio: true, supports_end_frame: true, supports_resolution: true, resolution_options: ["720p", "1080p"], supports_aspect_ratio: true, aspect_ratio_options: ["16:9", "9:16"] } },
      ]);
    });
  }, []);

  // Sync video provider when loading a previous task
  // (React-recommended pattern: adjust state during render, not in an effect)
  const [prevActiveProvider, setPrevActiveProvider] = useState(activeVideoProvider);
  if (activeVideoProvider && activeVideoProvider !== prevActiveProvider) {
    setPrevActiveProvider(activeVideoProvider);
    setVideoProvider(activeVideoProvider);
  }

  const currentVideoProvider = videoProviders.find((p) => p.name === videoProvider);
  const supportsEndFrame = currentVideoProvider?.capabilities?.supports_end_frame || false;

  const handleFiles = useCallback(
    async (files) => {
      if (!files) return;

      const remaining = maxTemplateImages - templateImages.length;
      if (remaining <= 0) return;

      const selected = Array.from(files)
        .filter((file) => file.type.startsWith("image/"))
        .slice(0, remaining);

      if (!selected.length) return;

      try {
        const encoded = await Promise.all(selected.map((file) => readFileAsDataUrl(file)));
        const valid = encoded.filter(Boolean);
        if (valid.length) {
          onTemplateImagesChange?.([...templateImages, ...valid]);
        }
      } catch (error) {
        console.error("Failed to load template images", error);
      }
    },
    [maxTemplateImages, onTemplateImagesChange, templateImages]
  );

  const handlePaste = useCallback(
    async (e) => {
      const items = e.clipboardData?.items;
      if (!items) return;

      const imageFiles = [];
      for (const item of items) {
        if (item.type.startsWith("image/")) {
          const file = item.getAsFile();
          if (file) imageFiles.push(file);
        }
      }
      if (imageFiles.length === 0) return;

      e.preventDefault();

      const isVideoMode = mediaType === "video";
      if (isVideoMode) {
        if (mode !== "image_to_video") {
          onModeChange?.("image_to_video");
        }
        const dataUrl = await readFileAsDataUrl(imageFiles[0]);
        if (!dataUrl) return;

        if (!startImage) {
          setStartImage(dataUrl);
        } else if (!endImage && supportsEndFrame) {
          setEndImage(dataUrl);
        }
      } else {
        if (mode !== "image_to_image") {
          onModeChange?.("image_to_image");
        }
        handleFiles(imageFiles);
      }
    },
    [mediaType, mode, onModeChange, startImage, endImage, supportsEndFrame, handleFiles]
  );

  const handleDrop = useCallback(
    (e) => {
      e.preventDefault();
      setIsDragging(false);
      if (mode === "image_to_image") {
        handleFiles(e.dataTransfer.files);
      }
    },
    [handleFiles, mode]
  );

  const handleDragOver = useCallback(
    (e) => {
      e.preventDefault();
      if (mode === "image_to_image") {
        setIsDragging(true);
      }
    },
    [mode]
  );

  const handleDragLeave = useCallback(() => {
    setIsDragging(false);
  }, []);

  const removeImage = (index) => {
    onTemplateImagesChange?.(templateImages.filter((_, i) => i !== index));
  };

  const updateConfig = (key, value) => {
    onConfigChange({ ...config, [key]: value });
  };

  const handleSubmit = (e) => {
    e?.preventDefault();
    const trimmed = prompt.trim();
    if (!trimmed) return;

    if (mediaType === "video") {
      if (onVideoGenerate) {
        onVideoGenerate({
          prompt: trimmed,
          provider: videoProvider,
          duration: videoConfig.duration,
          resolution: videoConfig.resolution,
          aspect_ratio: videoConfig.aspectRatio,
          generate_audio: videoConfig.audio,
          max_rounds: videoConfig.maxRounds,
          videos_per_round: videoConfig.videosPerRound,
          enhance_prompt: videoConfig.enhancePrompt,
          start_image: mode === "image_to_video" ? startImage : null,
          end_image: endImage,
        });
        setPrompt("");
        setShowAdvanced(false);
        setIsCollapsed(true);
      }
    } else {
      if (canGenerate) {
        onGenerate(trimmed);
        setPrompt("");
        setShowAdvanced(false);
        setIsCollapsed(true);
      }
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const isVideo = mediaType === "video";
  const canSubmit = isVideo
    ? prompt.trim() && (mode !== "image_to_video" || startImage)
    : prompt.trim() && canGenerate;

  return (
    <div className="pointer-events-none absolute bottom-5 left-0 right-0 z-30 flex justify-center px-4">
      <button
        onClick={() => setIsCollapsed(false)}
        className={cn(
          "pointer-events-auto absolute bottom-0 inline-flex items-center gap-1.5 rounded-full border border-border/70 bg-card/90 px-3 py-2 text-xs font-medium text-foreground shadow-xl backdrop-blur-xl transition-all duration-300 ease-out hover:-translate-y-0.5 hover:bg-card",
          isCollapsed
            ? "translate-y-0 scale-100 opacity-100"
            : "pointer-events-none translate-y-2 scale-95 opacity-0"
        )}
        aria-hidden={!isCollapsed}
        aria-label={t("panel.expand")}
      >
        <ChevronUp size={14} />
        {t("panel.expand")}
      </button>

      <div
        className={cn(
          "pointer-events-auto w-full max-w-4xl rounded-3xl border border-border/70 bg-card/88 p-2 shadow-[0_45px_120px_-60px_rgba(0,0,0,0.15)] dark:shadow-[0_45px_120px_-60px_rgba(0,0,0,0.65)] backdrop-blur-2xl transition-all duration-300 ease-out",
          isCollapsed
            ? "pointer-events-none translate-y-4 scale-[0.97] opacity-0"
            : "translate-y-0 scale-100 opacity-100",
          isDragging && "border-primary/55 shadow-[0_45px_120px_-55px_var(--brand-glow)]"
        )}
        aria-hidden={isCollapsed}
        onPaste={handlePaste}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
      >
          <div className="mb-2 flex justify-end">
            <button
              onClick={() => setIsCollapsed(true)}
              className="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs text-muted-foreground transition-colors hover:bg-secondary/70 hover:text-foreground"
              aria-label={t("panel.collapse")}
            >
              {t("panel.collapse")}
              <ChevronDown size={12} />
            </button>
          </div>

          {/* Image template upload area (image_to_image mode only) */}
          {!isVideo && mode === "image_to_image" ? (
          <div className="mb-2 rounded-2xl border border-border/65 bg-secondary/45 p-2.5">
            <div className="flex items-center gap-2 overflow-x-auto pb-1 scrollbar-hide">
              {templateImages.map((src, index) => (
                <div
                  key={`${src.slice(0, 16)}-${index}`}
                  className="group relative h-12 w-12 shrink-0 overflow-hidden rounded-xl border border-border/70"
                >
                  <img src={src} alt={`Ref ${index + 1}`} className="h-full w-full object-cover" />
                  <button
                    onClick={() => removeImage(index)}
                    className="absolute inset-0 flex items-center justify-center bg-black/55 opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <X className="h-3.5 w-3.5 text-white" />
                  </button>
                </div>
              ))}

              {templateImages.length < maxTemplateImages ? (
                <label
                  className={cn(
                    "flex h-12 w-12 shrink-0 cursor-pointer items-center justify-center rounded-xl border-2 border-dashed transition-all",
                    isDragging
                      ? "border-primary/60 bg-primary/10"
                      : "border-border/70 bg-card hover:border-primary/45 hover:bg-secondary/60"
                  )}
                >
                  <Plus className="h-4 w-4 text-muted-foreground" />
                  <input
                    type="file"
                    accept="image/*"
                    multiple
                    onChange={(e) => handleFiles(e.target.files)}
                    className="hidden"
                  />
                </label>
              ) : null}

              {templateImages.length === 0 ? (
                <span className="ml-1 text-xs text-muted-foreground">
                  {isDragging ? t("upload.dragHint") : t("upload.addReference")}
                </span>
              ) : null}
            </div>

            {templateImages.length === 0 && !isDragging ? (
              <p className="mt-2 text-[11px] font-medium text-primary/85">{t("upload.requiredHint")}</p>
            ) : null}
          </div>
          ) : null}

          {/* Video config panel (video mode only) */}
          {isVideo && (
            <div className="mb-2 rounded-2xl border border-border/65 bg-secondary/45 p-2.5">
              <VideoConfigPanel
                provider={videoProvider}
                providers={videoProviders}
                config={videoConfig}
                onConfigChange={setVideoConfig}
                onProviderChange={handleProviderChange}
                mode={mode}
                startImage={startImage}
                endImage={endImage}
                onStartImageChange={setStartImage}
                onEndImageChange={setEndImage}
              />
            </div>
          )}

          <div className="rounded-2xl border border-border/65 bg-background/55 px-3 py-3">
            <div className="flex items-start gap-3">
              <div className="min-w-0 flex-1">
                <textarea
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder={
                    isVideo
                      ? (mode === "image_to_video"
                        ? t("prompt.placeholder.image")
                        : t("prompt.placeholder.text"))
                      : (mode === "text_to_image"
                        ? t("prompt.placeholder.text")
                        : t("prompt.placeholder.image"))
                  }
                  rows={2}
                  className="min-h-12 w-full resize-none bg-transparent text-sm leading-relaxed text-foreground placeholder:text-muted-foreground outline-none"
                />

                <div className="mt-2 flex items-center justify-between">
                  <button
                    type="button"
                    onClick={() => {
                      if (isVideo) {
                        setVideoConfig((prev) => ({ ...prev, enhancePrompt: !prev.enhancePrompt }));
                      } else {
                        updateConfig("enhancePrompt", !config.enhancePrompt);
                      }
                    }}
                    className={cn(
                      "inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs font-medium transition-colors",
                      (isVideo ? videoConfig.enhancePrompt : config.enhancePrompt)
                        ? "bg-primary/15 text-primary"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    <Wand2 size={13} />
                    {t("config.enhancePrompt")}
                  </button>

                  <span className="text-[11px] text-muted-foreground">{prompt.length > 0 ? `${prompt.length} chars` : ""}</span>
                </div>
              </div>

              <button
                onClick={handleSubmit}
                disabled={!canSubmit}
                className={cn(
                  "flex h-12 min-w-32 items-center justify-center gap-2 rounded-2xl px-4 text-sm font-semibold transition-all",
                  !canSubmit
                    ? "cursor-not-allowed bg-secondary text-muted-foreground"
                    : "bg-gradient-to-r from-primary to-accent text-primary-foreground shadow-[0_20px_45px_-28px_var(--brand-glow)] hover:-translate-y-0.5"
                )}
              >
                {isGenerating ? <Loader2 size={16} className="animate-spin" /> : <Zap size={16} />}
                {isGenerating ? t("progress.working") : t("prompt.generate")}
              </button>
            </div>

            <div className="mt-3 flex flex-wrap items-center justify-between gap-3 border-t border-border/55 pt-3">
              <div className="flex flex-wrap items-center gap-3">
                {/* Media type toggle (Image / Video) */}
                {onMediaTypeChange && (
                  <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
                    {[
                      { id: "image", label: t("mediaType.image"), icon: ImageIcon },
                      { id: "video", label: t("mediaType.video"), icon: Film },
                    ].map((item) => (
                      <Tooltip key={item.id} text={item.label}>
                        <button
                          onClick={() => onMediaTypeChange(item.id)}
                          className={cn(
                            "flex h-8 w-8 items-center justify-center rounded-lg text-xs transition-all",
                            mediaType === item.id
                              ? "bg-primary/18 text-primary"
                              : "text-muted-foreground hover:text-foreground"
                          )}
                        >
                          <item.icon size={14} />
                        </button>
                      </Tooltip>
                    ))}
                  </div>
                )}

                {/* Mode selector */}
                <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
                  {(isVideo
                    ? [
                        { id: "text_to_video", label: t("mode.textToVideo"), icon: Film },
                        { id: "image_to_video", label: t("mode.imageToVideo"), icon: Video },
                      ]
                    : [
                        { id: "text_to_image", label: t("mode.textToImage"), icon: Type },
                        { id: "image_to_image", label: t("mode.imageEdit"), icon: Layers },
                      ]
                  ).map((item) => {
                    const isDisabled = disabledModes.includes(item.id);
                    return (
                      <Tooltip key={item.id} text={item.label}>
                        <button
                          onClick={() => !isDisabled && onModeChange?.(item.id)}
                          disabled={isDisabled}
                          className={cn(
                            "flex h-8 w-8 items-center justify-center rounded-lg text-xs transition-all",
                            isDisabled
                              ? "cursor-not-allowed text-muted-foreground/40"
                              : mode === item.id
                                ? "bg-primary/18 text-primary"
                                : "text-muted-foreground hover:text-foreground"
                          )}
                        >
                          <item.icon size={14} />
                        </button>
                      </Tooltip>
                    );
                  })}
                </div>

                {/* Aspect ratio and resolution for image mode only */}
                {!isVideo && (
                  <>
                    <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
                      {RATIO_OPTIONS.map((opt) => (
                        <Tooltip key={opt.value} text={opt.label}>
                          <button
                            onClick={() => updateConfig("aspectRatio", opt.value)}
                            className={cn(
                              "flex h-8 w-8 items-center justify-center rounded-lg transition-all",
                              config.aspectRatio === opt.value
                                ? "bg-primary/18 text-primary"
                                : "text-muted-foreground hover:text-foreground"
                            )}
                          >
                            <div className="rounded-[2px] border-2 border-current" style={{ width: opt.w, height: opt.h }} />
                          </button>
                        </Tooltip>
                      ))}
                    </div>

                    <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
                      {RESOLUTION_OPTIONS.map((res) => (
                        <button
                          key={res}
                          onClick={() => updateConfig("size", res)}
                          className={cn(
                            "rounded-lg px-2.5 py-1.5 text-[11px] font-medium transition-all",
                            config.size === res
                              ? "bg-primary/18 text-primary"
                              : "text-muted-foreground hover:text-foreground"
                          )}
                        >
                          {res}
                        </button>
                      ))}
                    </div>
                  </>
                )}
              </div>

              <button
                onClick={() => setShowAdvanced((prev) => !prev)}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs font-medium transition-colors",
                  showAdvanced ? "bg-primary/15 text-primary" : "text-muted-foreground hover:text-foreground"
                )}
              >
                <Sliders size={13} />
                {t("config.settings")}
                <ChevronRight size={12} className={cn("transition-transform", showAdvanced && "-rotate-90")} />
              </button>
            </div>

            <div
              className={cn(
                "grid transition-[grid-template-rows,opacity] duration-300 ease-out",
                showAdvanced ? "grid-rows-[1fr] opacity-100" : "grid-rows-[0fr] opacity-0"
              )}
            >
              <div className="overflow-hidden">
                {isVideo ? (
                  <div className="mt-3 grid gap-4 border-t border-border/55 pt-3 sm:grid-cols-2 lg:grid-cols-3">
                    <RangeSlider
                      label={t("video.rounds")}
                      value={videoConfig.maxRounds}
                      min={1}
                      max={5}
                      step={1}
                      onChange={(v) => setVideoConfig((prev) => ({ ...prev, maxRounds: v }))}
                    />
                    <RangeSlider
                      label={t("video.perRound")}
                      value={videoConfig.videosPerRound}
                      min={1}
                      max={4}
                      step={1}
                      onChange={(v) => setVideoConfig((prev) => ({ ...prev, videosPerRound: v }))}
                    />
                    <div className="flex items-center justify-between rounded-xl border border-border/60 bg-card/65 px-3 py-2 text-xs">
                      <span className="text-muted-foreground">{t("video.enhancePrompt")}</span>
                      <button
                        onClick={() => setVideoConfig((prev) => ({ ...prev, enhancePrompt: !prev.enhancePrompt }))}
                        className={cn(
                          "relative inline-flex h-5 w-9 items-center rounded-full transition-colors",
                          videoConfig.enhancePrompt ? "bg-primary" : "bg-muted"
                        )}
                      >
                        <span
                          className={cn(
                            "inline-block h-3.5 w-3.5 transform rounded-full bg-primary-foreground transition-transform",
                            videoConfig.enhancePrompt ? "translate-x-5" : "translate-x-1"
                          )}
                        />
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="mt-3 grid gap-4 border-t border-border/55 pt-3 sm:grid-cols-2 lg:grid-cols-5">
                    <RangeSlider
                      label={t("config.rounds")}
                      value={config.maxRounds}
                      min={1}
                      max={10}
                      step={1}
                      onChange={(v) => updateConfig("maxRounds", v)}
                    />
                    <RangeSlider
                      label={t("config.perRound")}
                      value={config.imagesPerRound}
                      min={1}
                      max={8}
                      step={1}
                      onChange={(v) => updateConfig("imagesPerRound", v)}
                    />
                    <RangeSlider
                      label={t("config.quality")}
                      value={config.qualityThreshold}
                      min={0.1}
                      max={1}
                      step={0.05}
                      onChange={(v) => updateConfig("qualityThreshold", v)}
                    />

                    <div className="flex items-center justify-between rounded-xl border border-border/60 bg-card/65 px-3 py-2 text-xs">
                      <span className="text-muted-foreground">{t("config.enhancePrompt")}</span>
                      <button
                        onClick={() => updateConfig("enhancePrompt", !config.enhancePrompt)}
                        className={cn(
                          "relative inline-flex h-5 w-9 items-center rounded-full transition-colors",
                          config.enhancePrompt ? "bg-primary" : "bg-muted"
                        )}
                      >
                        <span
                          className={cn(
                            "inline-block h-3.5 w-3.5 transform rounded-full bg-primary-foreground transition-transform",
                            config.enhancePrompt ? "translate-x-5" : "translate-x-1"
                          )}
                        />
                      </button>
                    </div>

                    <div className="flex items-center justify-between rounded-xl border border-border/60 bg-card/65 px-3 py-2 text-xs">
                      <span className="text-muted-foreground">{t("config.enableAiScoring")}</span>
                      <button
                        onClick={() => updateConfig("enableAiScoring", !(config.enableAiScoring ?? true))}
                        className={cn(
                          "relative inline-flex h-5 w-9 items-center rounded-full transition-colors",
                          (config.enableAiScoring ?? true) ? "bg-primary" : "bg-muted"
                        )}
                      >
                        <span
                          className={cn(
                            "inline-block h-3.5 w-3.5 transform rounded-full bg-primary-foreground transition-transform",
                            (config.enableAiScoring ?? true) ? "translate-x-5" : "translate-x-1"
                          )}
                        />
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

    </div>
  );
}
