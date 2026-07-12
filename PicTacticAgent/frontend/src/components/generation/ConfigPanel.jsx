import { useState } from "react";
import { SlidersHorizontal, ChevronDown, Sparkles } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";

const aspectRatios = ["1:1", "16:9", "9:16", "4:3", "3:4"];
const sizes = ["1K", "2K", "4K"];

function Toggle({ checked, onChange, disabled }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        checked ? "bg-primary" : "bg-border",
        disabled && "opacity-50 cursor-not-allowed"
      )}
    >
      <span
        className={cn(
          "pointer-events-none block h-4 w-4 rounded-full bg-white shadow-sm ring-0 transition-transform duration-200 mt-0.5",
          checked ? "translate-x-[18px]" : "translate-x-0.5"
        )}
      />
    </button>
  );
}

export function ConfigPanel({ config, onChange, disabled = false }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const { t } = useLocale();

  const updateConfig = (key, value) => {
    onChange({ ...config, [key]: value });
  };

  return (
    <div className="w-full rounded-xl ring-1 ring-border/30 bg-card overflow-hidden transition-all duration-300">
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex w-full items-center justify-between px-4 py-3 text-left transition-colors hover:bg-secondary/50"
        disabled={disabled}
      >
        <div className="flex items-center gap-2">
          <SlidersHorizontal className="h-3.5 w-3.5 text-primary/70" />
          <span className="text-xs font-semibold uppercase tracking-[0.08em] text-foreground">{t("config.settings")}</span>
        </div>
        <ChevronDown className={cn(
          "h-3.5 w-3.5 text-muted-foreground transition-transform duration-200",
          isExpanded && "rotate-180"
        )} />
      </button>

      {isExpanded && (
        <div className="border-t border-border/30 px-4 py-5 animate-fade-in space-y-6">
          {/* Generation Parameters Group */}
          <div className="space-y-4">
            <h4 className="text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground/70">{t("config.groupGeneration")}</h4>
            <div className="space-y-5">
              {[
                { key: "maxRounds", label: t("config.rounds"), min: 1, max: 10, step: 1, format: (v) => v },
                { key: "imagesPerRound", label: t("config.perRound"), min: 1, max: 10, step: 1, format: (v) => v },
                { key: "qualityThreshold", label: t("config.quality"), min: 0.5, max: 1, step: 0.05, format: (v) => (v ?? 0).toFixed(2) },
              ].map(({ key, label, min, max, step, format }) => (
                <div key={key} className="space-y-2">
                  <div className="flex justify-between text-xs">
                    <label className="text-muted-foreground font-medium">{label}</label>
                    <span className="tabular-nums font-semibold text-foreground">{format(config[key])}</span>
                  </div>
                  <input
                    type="range" min={min} max={max} step={step} value={config[key]}
                    onChange={(e) => updateConfig(key, step === 1 ? parseInt(e.target.value) : parseFloat(e.target.value))}
                    disabled={disabled}
                    className="custom-range w-full"
                  />
                </div>
              ))}
            </div>

            {/* Enhance Prompt Toggle */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1.5">
                <Sparkles className="h-3 w-3 text-primary/60" />
                <label className="text-xs text-muted-foreground font-medium">{t("config.enhancePrompt")}</label>
              </div>
              <Toggle
                checked={config.enhancePrompt}
                onChange={(v) => updateConfig("enhancePrompt", v)}
                disabled={disabled}
              />
            </div>
          </div>

          {/* Divider */}
          <div className="border-t border-border/20" />

          {/* Image Settings Group */}
          <div className="space-y-4">
            <h4 className="text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground/70">{t("config.groupImage")}</h4>

            {/* Aspect Ratio */}
            <div className="space-y-2.5">
              <label className="text-xs text-muted-foreground font-medium">{t("config.ratio")}</label>
              <div className="flex flex-wrap gap-1.5">
                {aspectRatios.map((ratio) => (
                  <button
                    key={ratio}
                    onClick={() => updateConfig("aspectRatio", ratio)}
                    disabled={disabled}
                    className={cn(
                      "rounded-full px-3 py-1 text-xs font-medium transition-all duration-150",
                      config.aspectRatio === ratio
                        ? "bg-primary text-primary-foreground shadow-sm"
                        : "bg-secondary/50 text-muted-foreground ring-1 ring-border/30 hover:bg-secondary hover:text-foreground"
                    )}
                  >
                    {ratio}
                  </button>
                ))}
              </div>
            </div>

            {/* Size */}
            <div className="space-y-2.5">
              <label className="text-xs text-muted-foreground font-medium">{t("config.resolution")}</label>
              <div className="inline-flex rounded-full bg-secondary/40 p-0.5 ring-1 ring-border/20">
                {sizes.map((size) => (
                  <button
                    key={size}
                    onClick={() => updateConfig("size", size)}
                    disabled={disabled}
                    className={cn(
                      "rounded-full px-4 py-1 text-xs font-medium transition-all duration-150",
                      config.size === size
                        ? "bg-primary text-primary-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground"
                    )}
                  >
                    {size}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
