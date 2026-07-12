import { useState, useCallback } from "react";
import { Upload, X, Image as ImageIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";

export function TemplateUpload({ images, onChange, maxImages = 5, disabled = false, required = false }) {
  const [isDragging, setIsDragging] = useState(false);
  const { t } = useLocale();

  const handleFiles = useCallback((files) => {
    if (!files || disabled) return;
    const newImages = [];
    const remaining = maxImages - images.length;

    Array.from(files).slice(0, remaining).forEach((file) => {
      if (file.type.startsWith("image/")) {
        const reader = new FileReader();
        reader.onload = (e) => {
          const result = e.target?.result;
          if (result) {
            newImages.push(result);
            if (newImages.length === Math.min(files.length, remaining)) {
              onChange([...images, ...newImages]);
            }
          }
        };
        reader.readAsDataURL(file);
      }
    });
  }, [images, maxImages, onChange, disabled]);

  const handleDrop = useCallback((e) => {
    e.preventDefault();
    setIsDragging(false);
    handleFiles(e.dataTransfer.files);
  }, [handleFiles]);

  const removeImage = (index) => {
    if (disabled) return;
    onChange(images.filter((_, i) => i !== index));
  };

  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.08em] text-foreground">
        <ImageIcon className="h-3.5 w-3.5 text-primary/70" />
        {t("upload.references")}{required ? ` ${t("upload.required")}` : ` ${t("upload.optional")}`}
      </label>

      {images.length < maxImages && (
        <label
          onDrop={handleDrop}
          onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
          onDragLeave={() => setIsDragging(false)}
          className={cn(
            "flex cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed p-5 transition-all duration-200",
            isDragging
              ? "border-primary/50 bg-primary/5"
              : "border-border/40 hover:border-primary/30 hover:bg-secondary/30",
            disabled && "cursor-not-allowed opacity-50"
          )}
        >
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-secondary/80 mb-2">
            <Upload className="h-4 w-4 text-muted-foreground" />
          </div>
          <p className="text-xs font-medium text-muted-foreground">{t("upload.dropOrClick")}</p>
          <p className="mt-0.5 text-[10px] text-muted-foreground/50">{t("upload.maxFiles", { count: maxImages })}</p>
          <input
            type="file"
            accept="image/*"
            multiple
            onChange={(e) => handleFiles(e.target.files)}
            disabled={disabled}
            className="hidden"
          />
        </label>
      )}

      {images.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {images.map((src, index) => (
            <div key={index} className="group relative h-16 w-16 overflow-hidden rounded-lg ring-1 ring-border/30">
              <img src={src} alt={`Ref ${index + 1}`} className="h-full w-full object-cover" />
              {!disabled && (
                <button
                  onClick={() => removeImage(index)}
                  className="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100"
                >
                  <X className="h-3.5 w-3.5 text-white" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {required && images.length === 0 && (
        <p className="text-[10px] text-amber-500 font-medium">{t("upload.requiredHint")}</p>
      )}
    </div>
  );
}
