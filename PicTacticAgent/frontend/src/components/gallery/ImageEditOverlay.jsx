import { Loader2 } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

export function ImageEditOverlay({ visible = false }) {
  const { t } = useLocale();

  if (!visible) return null;

  return (
    <div className="absolute inset-0 z-10 flex flex-col items-center justify-center rounded-xl overflow-hidden">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-[2px]" />
      <div className="absolute inset-y-0 -left-1/3 w-1/2 animate-shimmer opacity-60" />
      <div className="relative flex flex-col items-center gap-2">
        <Loader2 className="h-6 w-6 animate-spin text-white" />
        <span className="text-xs font-medium text-white/90">{t("edit.loading")}</span>
      </div>
    </div>
  );
}
