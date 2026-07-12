import { useState, useCallback } from "react";
import { apiClient } from "@/lib/api";

export function useImageEdit(taskId) {
  const [editingImageId, setEditingImageId] = useState(null);
  const [loadingImages, setLoadingImages] = useState({});
  const [editHistory, setEditHistory] = useState({});
  const [error, setError] = useState(null);

  const startEdit = useCallback((imageId) => {
    setEditingImageId(imageId);
    setError(null);
  }, []);

  const cancelEdit = useCallback(() => {
    setEditingImageId(null);
    setError(null);
  }, []);

  const confirmEdit = useCallback(
    async (imageId, editPrompt, enhancePrompt, onSuccess) => {
      if (!taskId) {
        const missingTaskError = new Error("Missing task id for image edit");
        setError(missingTaskError.message);
        throw missingTaskError;
      }

      if (!imageId) {
        const missingImageError = new Error("Missing image id for image edit");
        setError(missingImageError.message);
        throw missingImageError;
      }

      setLoadingImages((prev) => ({ ...prev, [imageId]: true }));
      setError(null);

      try {
        const result = await apiClient.editImage(taskId, imageId, {
          edit_prompt: editPrompt,
          enhance_prompt: enhancePrompt,
        });

        // Record edit history
        setEditHistory((prev) => {
          const history = prev[imageId] || [];
          return {
            ...prev,
            [imageId]: [
              ...history,
              {
                timestamp: Date.now(),
                editPrompt,
                enhanceEnabled: enhancePrompt,
                newImage: result.new_image,
              },
            ],
          };
        });

        setEditingImageId(null);
        if (onSuccess) onSuccess(imageId, result.new_image);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Edit failed");
        throw err;
      } finally {
        setLoadingImages((prev) => ({ ...prev, [imageId]: false }));
      }
    },
    [taskId]
  );

  const undoEdit = useCallback(
    (imageId, onUndo) => {
      setEditHistory((prev) => {
        const history = prev[imageId];
        if (!history || history.length === 0) return prev;
        const updated = history.slice(0, -1);
        return { ...prev, [imageId]: updated };
      });
      if (onUndo) onUndo(imageId);
    },
    []
  );

  const isEditing = useCallback(
    (imageId) => editingImageId === imageId,
    [editingImageId]
  );

  const isLoading = useCallback(
    (imageId) => !!loadingImages[imageId],
    [loadingImages]
  );

  const canUndo = useCallback(
    (imageId) => {
      const history = editHistory[imageId];
      return history && history.length > 0;
    },
    [editHistory]
  );

  const getEditCount = useCallback(
    (imageId) => (editHistory[imageId] || []).length,
    [editHistory]
  );

  return {
    editingImageId,
    loadingImages,
    editHistory,
    error,
    startEdit,
    cancelEdit,
    confirmEdit,
    undoEdit,
    isEditing,
    isLoading,
    canUndo,
    getEditCount,
  };
}
