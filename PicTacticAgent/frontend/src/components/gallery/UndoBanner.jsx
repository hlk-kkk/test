import { useState, useEffect } from "react";
import { Undo2 } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

export function UndoBanner({ onUndo, autoHideMs = 5000 }) {
  const [visible, setVisible] = useState(true);
  const { t } = useLocale();

  useEffect(() => {
    if (autoHideMs <= 0) return;
    const timer = setTimeout(() => setVisible(false), autoHideMs);
    return () => clearTimeout(timer);
  }, [autoHideMs]);

  if (!visible) return null;

  return (
    <div
      className="mt-1.5 flex items-center justify-between rounded-lg bg-muted/60 px-3 py-1.5 animate-slide-up"
      onMouseEnter={() => setVisible(true)}
    >
      <span className="text-[10px] text-muted-foreground">{t("edit.undoHint")}</span>
      <button
        onClick={() => {
          onUndo?.();
          setVisible(false);
        }}
        className="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] font-medium text-primary hover:bg-primary/10 transition-colors"
      >
        <Undo2 size={11} />
        {t("edit.undo")}
      </button>
    </div>
  );
}
