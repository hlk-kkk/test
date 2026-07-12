import { FileText } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";
import { TemplateCard } from "./TemplateCard";

export function TemplateList({ templates, loading, onGenerate, onEdit, onDelete, onImageClick }) {
  const { t } = useLocale();

  if (loading && !templates.length) {
    return (
      <div className="columns-1 gap-4 sm:columns-2 lg:columns-3 xl:columns-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="mb-4 break-inside-avoid animate-pulse rounded-xl border border-border/30 bg-card">
            <div className="bg-secondary/30 rounded-t-xl" style={{ height: `${180 + (i % 3) * 60}px` }} />
            <div className="p-3.5 space-y-2">
              <div className="h-4 w-2/3 rounded bg-secondary/50" />
              <div className="h-3 w-1/3 rounded bg-secondary/30" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (!templates.length) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-secondary text-muted-foreground/40">
          <FileText className="h-8 w-8" />
        </div>
        <p className="text-sm text-muted-foreground">{t("templates.empty")}</p>
      </div>
    );
  }

  return (
    <div className="columns-1 gap-4 sm:columns-2 lg:columns-3 xl:columns-4">
      {templates.map((template) => (
        <div key={template.id} className="mb-4 break-inside-avoid">
          <TemplateCard
            template={template}
            onGenerate={onGenerate}
            onEdit={onEdit}
            onDelete={onDelete}
            onImageClick={onImageClick}
          />
        </div>
      ))}
    </div>
  );
}
