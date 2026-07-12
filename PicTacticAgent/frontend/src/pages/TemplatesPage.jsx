import { useEffect, useState, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { useTemplates } from "@/hooks/useTemplates";
import { useLocale } from "@/hooks/useLocale";
import { useTheme } from "@/hooks/useTheme";
import { Header } from "@/components/layout/Header";
import { TemplateFilters } from "@/components/templates/TemplateFilters";
import { TemplateList } from "@/components/templates/TemplateList";
import { TemplateDialog } from "@/components/templates/TemplateDialog";
import { TemplatePreviewModal } from "@/components/templates/TemplatePreviewModal";

export default function TemplatesPage() {
  const { t } = useLocale();
  const navigate = useNavigate();
  const { theme, toggleTheme } = useTheme();

  const {
    templates,
    loading,
    search,
    category,
    categories,
    hasMore,
    fetchTemplates,
    loadMore,
    createTemplate,
    updateTemplate,
    deleteTemplate,
    handleSearch,
    handleCategoryChange,
  } = useTemplates();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState(null);
  const [previewTemplate, setPreviewTemplate] = useState(null);

  // Infinite scroll sentinel
  const sentinelRef = useRef(null);

  useEffect(() => {
    fetchTemplates();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          loadMore();
        }
      },
      { rootMargin: "200px" }
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMore, loading, loadMore]);

  const handleCreate = useCallback(() => {
    setEditingTemplate(null);
    setDialogOpen(true);
  }, []);

  const handleEdit = useCallback((template) => {
    setEditingTemplate(template);
    setDialogOpen(true);
  }, []);

  const handleDelete = useCallback(async (template) => {
    if (!window.confirm(t("templates.deleteConfirm"))) return;
    try {
      await deleteTemplate(template.id);
      toast.success(t("templates.deleteSuccess"));
    } catch {
      toast.error(t("templates.deleteFailed"));
    }
  }, [deleteTemplate, t]);

  const handleGenerate = useCallback((template) => {
    const params = new URLSearchParams();
    params.set("templatePrompt", template.prompt_text);
    navigate(`/app?${params.toString()}`);
  }, [navigate]);

  const handleSave = useCallback(async (data) => {
    try {
      if (editingTemplate?.id) {
        await updateTemplate(editingTemplate.id, data);
        toast.success(t("templates.updateSuccess"));
      } else {
        await createTemplate(data);
        toast.success(t("templates.createSuccess"));
      }
    } catch {
      toast.error(t("templates.saveFailed"));
      throw new Error("save failed");
    }
  }, [editingTemplate, createTemplate, updateTemplate, t]);

  const handleImageClick = useCallback((template) => {
    setPreviewTemplate(template);
  }, []);

  return (
    <div className="min-h-screen bg-background text-foreground">
      <Header theme={theme} onToggleTheme={toggleTheme} />

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        {/* Page Title + Actions */}
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">{t("templates.title")}</h1>
          <button
            onClick={handleCreate}
            className="flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <Plus className="h-3.5 w-3.5" />
            {t("templates.create")}
          </button>
        </div>

        {/* Filters */}
        <div className="mb-6">
          <TemplateFilters
            search={search}
            category={category}
            categories={categories}
            onSearchChange={handleSearch}
            onCategoryChange={handleCategoryChange}
          />
        </div>

        {/* Template List */}
        <TemplateList
          templates={templates}
          loading={loading}
          onGenerate={handleGenerate}
          onEdit={handleEdit}
          onDelete={handleDelete}
          onImageClick={handleImageClick}
        />

        {/* Infinite scroll sentinel */}
        {hasMore && <div ref={sentinelRef} className="h-10" />}

        {/* Loading more indicator */}
        {loading && templates.length > 0 && (
          <div className="flex justify-center py-6">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary border-t-transparent" />
          </div>
        )}
      </main>

      <TemplateDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        template={editingTemplate}
        onSave={handleSave}
        categories={categories}
      />

      {previewTemplate && (
        <TemplatePreviewModal
          template={previewTemplate}
          onClose={() => setPreviewTemplate(null)}
        />
      )}
    </div>
  );
}
