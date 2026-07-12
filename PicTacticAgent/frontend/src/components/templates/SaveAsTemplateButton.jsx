import { useState, useMemo } from "react";
import { Bookmark } from "lucide-react";
import { toast } from "sonner";
import { useLocale } from "@/hooks/useLocale";
import { apiClient } from "@/lib/api";
import { TemplateDialog } from "./TemplateDialog";

export function SaveAsTemplateButton({ prompt, config, previewImageUrl }) {
  const { t } = useLocale();
  const [open, setOpen] = useState(false);

  const prefilled = useMemo(() => ({
    prompt_text: prompt || "",
    preview_image_url: previewImageUrl || null,
    config: config
      ? {
          aspect_ratio: config.aspectRatio || config.aspect_ratio || null,
          size: config.size || null,
          images_per_round: config.imagesPerRound || config.images_per_round || null,
          max_rounds: config.maxRounds || config.max_rounds || null,
        }
      : null,
  }), [prompt, config, previewImageUrl]);

  const handleSave = async (data) => {
    try {
      if (previewImageUrl) {
        data.preview_image_url = previewImageUrl;
      }
      await apiClient.createTemplate(data);
      toast.success(t("templates.saveSuccess"));
    } catch (err) {
      toast.error(t("templates.saveFailed"));
      throw err;
    }
  };

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="flex items-center gap-1.5 rounded-xl border border-border/65 bg-card/70 px-3 py-2 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground hover:bg-secondary"
      >
        <Bookmark className="h-3.5 w-3.5" />
        {t("templates.saveAs")}
      </button>
      <TemplateDialog
        open={open}
        onClose={() => setOpen(false)}
        template={null}
        initialData={prefilled}
        onSave={handleSave}
      />
    </>
  );
}
