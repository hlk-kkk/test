import { useState, useEffect, useCallback, useRef } from "react";
import { X, Plus, ChevronDown } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";
import { cn } from "@/lib/utils";
import { Select, SelectItem } from "@/components/ui/Select";

export function TemplateDialog({ open, onClose, template, initialData, onSave, categories = [] }) {
  const { t } = useLocale();
  const isEdit = !!template?.id;

  const [form, setForm] = useState({
    title: "",
    description: "",
    prompt_text: "",
    category: "other",
    tags: [],
    config: null,
  });
  const [tagInput, setTagInput] = useState("");
  const [showConfig, setShowConfig] = useState(false);
  const [saving, setSaving] = useState(false);

  const prevOpenRef = useRef(false);

  useEffect(() => {
    if (open && !prevOpenRef.current) {
      const source = template || initialData;
      if (source) {
        setForm({
          title: source.title || "",
          description: source.description || "",
          prompt_text: source.prompt_text || "",
          category: source.category || "other",
          tags: source.tags || [],
          config: source.config || null,
        });
        setShowConfig(!!source.config);
      } else {
        setForm({
          title: "",
          description: "",
          prompt_text: "",
          category: "other",
          tags: [],
          config: null,
        });
        setShowConfig(false);
      }
      setTagInput("");
    }
    prevOpenRef.current = open;
  }, [open, template, initialData]);

  const updateField = (key, value) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const updateConfig = (key, value) => {
    setForm((prev) => ({
      ...prev,
      config: { ...(prev.config || {}), [key]: value },
    }));
  };

  const handleAddTag = useCallback((e) => {
    if (e.key === "Enter" && tagInput.trim()) {
      e.preventDefault();
      const tag = tagInput.trim();
      if (!form.tags.includes(tag) && form.tags.length < 20) {
        updateField("tags", [...form.tags, tag]);
      }
      setTagInput("");
    }
  }, [tagInput, form.tags]);

  const handleRemoveTag = (tag) => {
    updateField("tags", form.tags.filter((t) => t !== tag));
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!form.title.trim() || !form.prompt_text.trim()) return;
    setSaving(true);
    try {
      const data = {
        ...form,
        category: form.category || "other",
        config: showConfig ? form.config : null,
      };
      await onSave(data);
      onClose();
    } catch {
      // Error handling delegated to parent
    } finally {
      setSaving(false);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />
      <div className="relative z-10 w-full max-w-lg max-h-[85vh] overflow-y-auto rounded-2xl border border-border bg-card shadow-2xl mx-4">
        {/* Header */}
        <div className="sticky top-0 z-10 flex items-center justify-between border-b border-border/50 bg-card px-5 py-4">
          <h2 className="text-base font-semibold text-foreground">
            {isEdit ? t("templates.edit") : t("templates.create")}
          </h2>
          <button
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {/* Title */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t("templates.form.title")} *</label>
            <input
              type="text"
              value={form.title}
              onChange={(e) => updateField("title", e.target.value)}
              maxLength={200}
              required
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/50 focus:border-primary/50 focus:outline-none focus:ring-1 focus:ring-primary/30"
            />
          </div>

          {/* Description */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t("templates.form.description")}</label>
            <textarea
              value={form.description}
              onChange={(e) => updateField("description", e.target.value)}
              maxLength={2000}
              rows={2}
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/50 focus:border-primary/50 focus:outline-none focus:ring-1 focus:ring-primary/30 resize-none"
            />
          </div>

          {/* Prompt Text */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t("templates.form.promptText")} *</label>
            <textarea
              value={form.prompt_text}
              onChange={(e) => updateField("prompt_text", e.target.value)}
              required
              rows={4}
              className="w-full rounded-lg border border-border/50 bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/50 focus:border-primary/50 focus:outline-none focus:ring-1 focus:ring-primary/30 resize-none"
            />
          </div>

          {/* Category */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t("templates.form.category")}</label>
            <Select
              value={form.category}
              onValueChange={(val) => updateField("category", val)}
            >
              {(categories || []).map((cat) => (
                <SelectItem key={cat} value={cat}>
                  {t(`templates.category.${cat}`, cat)}
                </SelectItem>
              ))}
            </Select>
          </div>

          {/* Tags */}
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">{t("templates.form.tags")}</label>
            <div className="flex flex-wrap gap-1.5 rounded-lg border border-border/50 bg-background p-2 min-h-[38px]">
              {form.tags.map((tag) => (
                <span
                  key={tag}
                  className="flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary"
                >
                  {tag}
                  <button
                    type="button"
                    onClick={() => handleRemoveTag(tag)}
                    className="ml-0.5 text-primary/60 hover:text-primary"
                  >
                    <X className="h-2.5 w-2.5" />
                  </button>
                </span>
              ))}
              <input
                type="text"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={handleAddTag}
                placeholder={form.tags.length === 0 ? "Enter to add..." : ""}
                className="flex-1 min-w-[80px] border-none bg-transparent p-0 text-xs text-foreground placeholder:text-muted-foreground/40 focus:outline-none"
              />
            </div>
          </div>

          {/* Config Toggle */}
          <button
            type="button"
            onClick={() => setShowConfig(!showConfig)}
            className="flex w-full items-center justify-between rounded-lg border border-border/30 bg-secondary/30 px-3 py-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-secondary/50"
          >
            <span>{t("templates.form.config")}</span>
            <ChevronDown className={cn("h-3.5 w-3.5 transition-transform", showConfig && "rotate-180")} />
          </button>

          {showConfig && (
            <div className="space-y-3 rounded-lg border border-border/30 bg-secondary/20 p-3">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-[10px] text-muted-foreground">{t("config.rounds")}</label>
                  <input
                    type="number"
                    min={1}
                    max={10}
                    value={form.config?.max_rounds ?? ""}
                    onChange={(e) => updateConfig("max_rounds", e.target.value ? parseInt(e.target.value) : null)}
                    className="w-full rounded border border-border/40 bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-[10px] text-muted-foreground">{t("config.perRound")}</label>
                  <input
                    type="number"
                    min={1}
                    max={10}
                    value={form.config?.images_per_round ?? ""}
                    onChange={(e) => updateConfig("images_per_round", e.target.value ? parseInt(e.target.value) : null)}
                    className="w-full rounded border border-border/40 bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary/30"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-[10px] text-muted-foreground">{t("config.ratio")}</label>
                  <Select
                    value={form.config?.aspect_ratio ?? "__none__"}
                    onValueChange={(val) => updateConfig("aspect_ratio", val === "__none__" ? null : val)}
                    className="text-xs px-2 py-1.5"
                  >
                    <SelectItem value="__none__" className="text-xs">-</SelectItem>
                    {["1:1", "16:9", "9:16", "4:3", "3:4"].map((r) => (
                      <SelectItem key={r} value={r} className="text-xs">{r}</SelectItem>
                    ))}
                  </Select>
                </div>
                <div className="space-y-1">
                  <label className="text-[10px] text-muted-foreground">{t("config.resolution")}</label>
                  <Select
                    value={form.config?.size ?? "__none__"}
                    onValueChange={(val) => updateConfig("size", val === "__none__" ? null : val)}
                    className="text-xs px-2 py-1.5"
                  >
                    <SelectItem value="__none__" className="text-xs">-</SelectItem>
                    {["1K", "2K", "4K"].map((s) => (
                      <SelectItem key={s} value={s} className="text-xs">{s}</SelectItem>
                    ))}
                  </Select>
                </div>
              </div>
            </div>
          )}

          {/* Submit */}
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-border/50 bg-card px-4 py-2 text-xs font-medium text-foreground transition-colors hover:bg-secondary"
            >
              {t("edit.cancel")}
            </button>
            <button
              type="submit"
              disabled={saving || !form.title.trim() || !form.prompt_text.trim()}
              className="rounded-lg bg-primary px-4 py-2 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? "..." : isEdit ? t("templates.edit") : t("templates.create")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
