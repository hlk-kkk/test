import { useCallback, useMemo } from "react";
import { Film, Image, Music, Plus, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";

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

const RATIO_OPTIONS = [
  { label: "1:1", value: "1:1", w: 12, h: 12 },
  { label: "16:9", value: "16:9", w: 14, h: 8 },
  { label: "9:16", value: "9:16", w: 8, h: 14 },
  { label: "4:3", value: "4:3", w: 13, h: 10 },
  { label: "3:4", value: "3:4", w: 10, h: 13 },
];

export function VideoConfigPanel({
  provider,
  providers = [],
  config,
  onConfigChange,
  onProviderChange,
  mode,
  startImage,
  endImage,
  onStartImageChange,
  onEndImageChange,
}) {
  const { t } = useLocale();

  const currentProvider = useMemo(() => {
    return providers.find((p) => p.name === provider) || null;
  }, [providers, provider]);

  const capabilities = useMemo(() => currentProvider?.capabilities || {}, [currentProvider]);

  // Duration control: range or button group depending on provider
  const durationOptions = capabilities.duration_options || null;
  const durationRange = capabilities.duration_range || null;

  const supportsAudio = capabilities.supports_audio || false;
  const supportsEndFrame = capabilities.supports_end_frame || false;
  const supportsResolution = capabilities.supports_resolution || false;
  const supportsAspectRatio = capabilities.supports_aspect_ratio || false;
  const resolutionOptions = capabilities.resolution_options || [];

  // Filter RATIO_OPTIONS to only show provider-supported ratios
  const filteredRatios = useMemo(() => {
    const options = capabilities.aspect_ratio_options || [];
    if (!options.length) return RATIO_OPTIONS;
    return RATIO_OPTIONS.filter((opt) => options.includes(opt.value));
  }, [capabilities]);

  const updateConfig = useCallback(
    (key, value) => {
      onConfigChange({ ...config, [key]: value });
    },
    [config, onConfigChange]
  );

  const handleImageUpload = useCallback(
    async (files, setter) => {
      if (!files || files.length === 0) return;
      const file = Array.from(files).find((f) => f.type.startsWith("image/"));
      if (!file) return;
      try {
        const dataUrl = await readFileAsDataUrl(file);
        if (dataUrl) setter(dataUrl);
      } catch (err) {
        console.error("Failed to load image:", err);
      }
    },
    []
  );

  return (
    <div className="space-y-3">
      {/* Provider selector */}
      {providers.length > 0 && (
        <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
          {providers.map((p) => (
            <Tooltip key={p.name} text={p.display_name || p.name}>
              <button
                onClick={() => onProviderChange(p.name)}
                className={cn(
                  "flex-1 rounded-lg px-3 py-1.5 text-[11px] font-medium transition-all capitalize",
                  provider === p.name
                    ? "bg-primary/18 text-primary"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {p.display_name || p.name}
              </button>
            </Tooltip>
          ))}
        </div>
      )}

      {/* Start frame upload (image_to_video mode) */}
      {mode === "image_to_video" && (
        <div className="rounded-xl border border-border/65 bg-secondary/45 p-2.5">
          <div className="mb-1.5 text-[11px] font-medium text-muted-foreground">
            {t("video.startFrame")}
          </div>
          <div className="flex items-center gap-2">
            {startImage ? (
              <div className="group relative h-14 w-14 shrink-0 overflow-hidden rounded-xl border border-border/70">
                <img src={startImage} alt="Start frame" className="h-full w-full object-cover" />
                <button
                  onClick={() => onStartImageChange(null)}
                  className="absolute inset-0 flex items-center justify-center bg-black/55 opacity-0 transition-opacity group-hover:opacity-100"
                >
                  <X className="h-3.5 w-3.5 text-white" />
                </button>
              </div>
            ) : (
              <label className="flex h-14 w-14 shrink-0 cursor-pointer items-center justify-center rounded-xl border-2 border-dashed border-border/70 bg-card transition-all hover:border-primary/45 hover:bg-secondary/60">
                <Plus className="h-4 w-4 text-muted-foreground" />
                <input
                  type="file"
                  accept="image/*"
                  onChange={(e) => handleImageUpload(e.target.files, onStartImageChange)}
                  className="hidden"
                />
              </label>
            )}
            <span className="text-[11px] text-muted-foreground">
              {t("video.startFrame.hint")}
            </span>
          </div>
        </div>
      )}

      {/* End frame upload (i2v mode + kling, veo only) */}
      {mode === "image_to_video" && supportsEndFrame && (
        <div className="rounded-xl border border-border/65 bg-secondary/45 p-2.5">
          <div className="mb-1.5 text-[11px] font-medium text-muted-foreground">
            {t("video.endFrame")}
          </div>
          <div className="flex items-center gap-2">
            {endImage ? (
              <div className="group relative h-14 w-14 shrink-0 overflow-hidden rounded-xl border border-border/70">
                <img src={endImage} alt="End frame" className="h-full w-full object-cover" />
                <button
                  onClick={() => onEndImageChange(null)}
                  className="absolute inset-0 flex items-center justify-center bg-black/55 opacity-0 transition-opacity group-hover:opacity-100"
                >
                  <X className="h-3.5 w-3.5 text-white" />
                </button>
              </div>
            ) : (
              <label className="flex h-14 w-14 shrink-0 cursor-pointer items-center justify-center rounded-xl border-2 border-dashed border-border/70 bg-card transition-all hover:border-primary/45 hover:bg-secondary/60">
                <Plus className="h-4 w-4 text-muted-foreground" />
                <input
                  type="file"
                  accept="image/*"
                  onChange={(e) => handleImageUpload(e.target.files, onEndImageChange)}
                  className="hidden"
                />
              </label>
            )}
            <span className="text-[11px] text-muted-foreground">
              {t("video.endFrame.hint")}
            </span>
          </div>
        </div>
      )}

      {/* Duration control */}
      {durationOptions ? (
        /* Button group for discrete duration options (sora, veo) */
        <div>
          <div className="mb-1.5 text-[11px] font-medium text-muted-foreground">
            {t("video.duration")}
          </div>
          <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
            {durationOptions.map((dur) => (
              <button
                key={dur}
                onClick={() => updateConfig("duration", dur)}
                className={cn(
                  "flex-1 rounded-lg px-2.5 py-1.5 text-[11px] font-medium transition-all",
                  config.duration === dur
                    ? "bg-primary/18 text-primary"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {dur}s
              </button>
            ))}
          </div>
        </div>
      ) : durationRange ? (
        /* Range slider for continuous duration (kling) */
        <div className="space-y-2">
          <div className="flex items-center justify-between text-xs">
            <span className="text-muted-foreground">{t("video.duration")}</span>
            <span className="font-mono font-semibold text-primary">{config.duration}s</span>
          </div>
          <input
            type="range"
            min={durationRange.min}
            max={durationRange.max}
            step={durationRange.step || 1}
            value={config.duration}
            onChange={(e) => updateConfig("duration", parseInt(e.target.value, 10))}
            className="custom-range w-full"
          />
        </div>
      ) : null}

      {/* Resolution buttons (sora, veo only) */}
      {supportsResolution && resolutionOptions.length > 0 && (
        <div>
          <div className="mb-1.5 text-[11px] font-medium text-muted-foreground">
            {t("video.resolution")}
          </div>
          <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
            {resolutionOptions.map((res) => (
              <button
                key={res}
                onClick={() => updateConfig("resolution", res)}
                className={cn(
                  "flex-1 rounded-lg px-2.5 py-1.5 text-[11px] font-medium transition-all",
                  config.resolution === res
                    ? "bg-primary/18 text-primary"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {res}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Audio toggle (kling, veo only) */}
      {supportsAudio && (
        <button
          type="button"
          onClick={() => updateConfig("audio", !config.audio)}
          className={cn(
            "flex w-full cursor-pointer items-center gap-2.5 rounded-xl border px-3 py-2.5 text-xs transition-all",
            config.audio
              ? "border-primary/40 bg-primary/10 text-primary"
              : "border-border/60 bg-card/65 text-muted-foreground hover:border-border hover:text-foreground"
          )}
        >
          <Music size={14} />
          <span className="flex-1 text-left font-medium">{t("video.audio")}</span>
          <div
            className={cn(
              "relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors",
              config.audio ? "bg-primary" : "bg-muted"
            )}
          >
            <span
              className={cn(
                "inline-block h-4 w-4 transform rounded-full bg-primary-foreground shadow-sm transition-transform",
                config.audio ? "translate-x-6" : "translate-x-1"
              )}
            />
          </div>
        </button>
      )}

      {/* Aspect ratio buttons */}
      {supportsAspectRatio && filteredRatios.length > 0 && (
        <div>
          <div className="mb-1.5 text-[11px] font-medium text-muted-foreground">
            {t("video.aspectRatio")}
          </div>
          <div className="flex items-center gap-1 rounded-xl border border-border/65 bg-card/60 p-1">
            {filteredRatios.map((opt) => (
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
                  <div
                    className="rounded-[2px] border-2 border-current"
                    style={{ width: opt.w, height: opt.h }}
                  />
                </button>
              </Tooltip>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
