import { Type, Layers, Film, Image as ImageIcon, Video } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";

const IMAGE_MODES = [
  { id: "text_to_image", labelKey: "mode.textToImage", icon: Type },
  { id: "image_to_image", labelKey: "mode.imageEdit", icon: Layers },
];

const VIDEO_MODES = [
  { id: "text_to_video", labelKey: "mode.textToVideo", icon: Film },
  { id: "image_to_video", labelKey: "mode.imageToVideo", icon: Video },
];

export function ModeSelector({ mode, onChange, disabledModes = [], mediaType = "image", onMediaTypeChange }) {
  const { t } = useLocale();
  const modes = mediaType === "video" ? VIDEO_MODES : IMAGE_MODES;

  return (
    <div className="space-y-2">
      {/* Media type toggle */}
      {onMediaTypeChange && (
        <div className="flex rounded-xl bg-secondary/50 p-1 ring-1 ring-border/30">
          {[
            { id: "image", label: t("mediaType.image"), icon: ImageIcon },
            { id: "video", label: t("mediaType.video"), icon: Film },
          ].map((item) => (
            <button
              key={item.id}
              onClick={() => onMediaTypeChange(item.id)}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 rounded-[10px] px-4 py-2 text-[13px] font-medium transition-all duration-200",
                mediaType === item.id
                  ? "bg-card text-foreground shadow-sm ring-1 ring-border/50"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <item.icon className="h-3.5 w-3.5" />
              {item.label}
            </button>
          ))}
        </div>
      )}

      {/* Mode selector */}
      <div className="flex rounded-xl bg-secondary/50 p-1 ring-1 ring-border/30">
        {modes.map((item) => {
          const isDisabled = disabledModes.includes(item.id);
          return (
            <button
              key={item.id}
              onClick={() => !isDisabled && onChange(item.id)}
              disabled={isDisabled}
              title={isDisabled ? t("mode.notSupported") : undefined}
              className={cn(
                "flex flex-1 items-center justify-center gap-2 rounded-[10px] px-4 py-2 text-[13px] font-medium transition-all duration-200",
                isDisabled
                  ? "cursor-not-allowed text-muted-foreground/40"
                  : mode === item.id
                    ? "bg-card text-foreground shadow-sm ring-1 ring-border/50"
                    : "text-muted-foreground hover:text-foreground"
              )}
            >
              <item.icon className="h-3.5 w-3.5" />
              {t(item.labelKey)}
            </button>
          );
        })}
      </div>
    </div>
  );
}
