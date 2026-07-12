import { Play, Pencil, Trash2, FileText, ExternalLink, ZoomIn } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

export function TemplateCard({ template, onGenerate, onEdit, onDelete, onImageClick }) {
  const { t } = useLocale();
  const isSystem = template.is_system;

  return (
    <div className="group relative overflow-hidden rounded-xl border border-border/50 bg-card transition-all duration-200 hover:border-border hover:shadow-md hover:-translate-y-0.5">
      {/* Preview */}
      <div className="group/img relative bg-secondary/30 cursor-pointer" onClick={() => onImageClick?.(template)}>
        {template.preview_image_url ? (
          <img
            src={template.preview_image_url}
            alt={template.title}
            className="w-full h-auto max-h-[400px] object-cover rounded-t-xl"
            loading="lazy"
          />
        ) : (
          <div className="flex h-40 w-full items-center justify-center text-muted-foreground/40">
            <FileText className="h-10 w-10" />
          </div>
        )}
        {/* Zoom hint */}
        {template.preview_image_url && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/0 transition-colors group-hover/img:bg-black/20 rounded-t-xl">
            <div className="flex items-center gap-1.5 rounded-full bg-black/50 px-2.5 py-1 text-[10px] text-white/80 opacity-0 transition-opacity group-hover/img:opacity-100 backdrop-blur-sm">
              <ZoomIn className="h-3 w-3" />
              {t("modal.clickToZoom")}
            </div>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="p-3.5 space-y-2">
        <h3 className="text-sm font-semibold text-foreground truncate">{template.title}</h3>

        <div className="flex items-center gap-2">
          <span className="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
            {t(`templates.category.${template.category}`, template.category)}
          </span>
          {isSystem && template.source_author && (
            <span className="text-[10px] text-muted-foreground truncate max-w-[80px]">
              @{template.source_author}
            </span>
          )}
          <span className="text-[10px] text-muted-foreground">
            {t("templates.usageCount", { count: template.usage_count })}
          </span>
        </div>

        {template.tags?.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {template.tags.slice(0, 4).map((tag) => (
              <span
                key={tag}
                className="rounded-full border border-border/50 bg-secondary/50 px-1.5 py-0.5 text-[10px] text-muted-foreground"
              >
                {tag}
              </span>
            ))}
            {template.tags.length > 4 && (
              <span className="text-[10px] text-muted-foreground">+{template.tags.length - 4}</span>
            )}
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center gap-1.5 pt-1 border-t border-border/30">
          <button
            onClick={() => onGenerate(template)}
            className="flex flex-1 items-center justify-center gap-1 rounded-lg bg-primary/90 py-1.5 text-[11px] font-medium text-primary-foreground transition-colors hover:bg-primary"
          >
            <Play className="h-3 w-3" />
            {t("templates.generate")}
          </button>
          {!isSystem && (
            <>
              <button
                onClick={() => onEdit(template)}
                className="flex items-center justify-center rounded-lg border border-border/50 bg-card p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
              >
                <Pencil className="h-3 w-3" />
              </button>
              <button
                onClick={() => onDelete(template)}
                className="flex items-center justify-center rounded-lg border border-border/50 bg-card p-1.5 text-muted-foreground transition-colors hover:bg-red-500/10 hover:text-red-500 hover:border-red-500/30"
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </>
          )}
          {isSystem && template.source_url && (
            <a
              href={template.source_url}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center justify-center rounded-lg border border-border/50 bg-card p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
              title="Source"
            >
              <ExternalLink className="h-3 w-3" />
            </a>
          )}
        </div>
      </div>
    </div>
  );
}
