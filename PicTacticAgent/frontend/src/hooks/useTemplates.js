import { useState, useCallback, useEffect, useRef } from "react";
import { apiClient } from "@/lib/api";

export function useTemplates() {
  const [templates, setTemplates] = useState([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [categories, setCategories] = useState([]);
  const [hasMore, setHasMore] = useState(true);
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState(null);

  const PAGE_SIZE = 30;

  // Use refs so callbacks don't need state in deps → stable identity
  const pageRef = useRef(1);
  const loadingRef = useRef(false);
  const hasMoreRef = useRef(true);
  const searchRef = useRef("");
  const categoryRef = useRef(null);

  const fetchCategories = useCallback(async () => {
    try {
      const cats = await apiClient.getTemplateCategories();
      setCategories(cats);
    } catch {
      setCategories([]);
    }
  }, []);

  useEffect(() => {
    fetchCategories();
  }, [fetchCategories]);

  // Core fetch — fully driven by refs, so identity never changes
  const fetchPage = useCallback(async (page, reset) => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setLoading(true);
    setError(null);

    try {
      const result = await apiClient.listTemplates({
        search: searchRef.current,
        category: categoryRef.current,
        page,
        pageSize: PAGE_SIZE,
      });

      if (reset) {
        setTemplates(result.items);
      } else {
        setTemplates((prev) => [...prev, ...result.items]);
      }
      setTotal(result.total);

      const more = page * PAGE_SIZE < result.total;
      setHasMore(more);
      hasMoreRef.current = more;
      pageRef.current = page;
    } catch (err) {
      setError(err.message);
    } finally {
      loadingRef.current = false;
      setLoading(false);
    }
  }, []); // no deps → stable forever

  // Initial load
  const fetchTemplates = useCallback(() => {
    pageRef.current = 1;
    fetchPage(1, true);
  }, [fetchPage]);

  const loadMore = useCallback(() => {
    if (loadingRef.current || !hasMoreRef.current) return;
    fetchPage(pageRef.current + 1, false);
  }, [fetchPage]);

  const createTemplate = useCallback(async (data) => {
    const created = await apiClient.createTemplate(data);
    fetchTemplates();
    return created;
  }, [fetchTemplates]);

  const updateTemplate = useCallback(async (templateId, data) => {
    const updated = await apiClient.updateTemplate(templateId, data);
    fetchTemplates();
    return updated;
  }, [fetchTemplates]);

  const deleteTemplate = useCallback(async (templateId) => {
    await apiClient.deleteTemplate(templateId);
    fetchTemplates();
  }, [fetchTemplates]);

  const generateFromTemplate = useCallback(async (templateId, overrides) => {
    return await apiClient.generateFromTemplate(templateId, overrides);
  }, []);

  const handleSearch = useCallback((value) => {
    setSearch(value);
    searchRef.current = value;
    pageRef.current = 1;
    fetchPage(1, true);
  }, [fetchPage]);

  const handleCategoryChange = useCallback((cat) => {
    setCategory(cat);
    categoryRef.current = cat;
    pageRef.current = 1;
    fetchPage(1, true);
  }, [fetchPage]);

  return {
    templates,
    total,
    loading,
    error,
    search,
    category,
    categories,
    hasMore,
    fetchTemplates,
    loadMore,
    createTemplate,
    updateTemplate,
    deleteTemplate,
    generateFromTemplate,
    handleSearch,
    handleCategoryChange,
  };
}
