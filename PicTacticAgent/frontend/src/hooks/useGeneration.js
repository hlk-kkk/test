import { useState, useCallback, useRef, useEffect } from "react";
import { apiClient } from "@/lib/api";

export function useGeneration() {
  const [status, setStatus] = useState("idle");
  const [progress, setProgress] = useState(null);
  const [result, setResult] = useState(null);
  const [error, setError] = useState(null);
  const [generationConfig, setGenerationConfig] = useState(null);
  const [streamedImages, setStreamedImages] = useState([]);
  const [taskContext, setTaskContext] = useState(null);

  const taskIdRef = useRef(null);
  const taskCacheRef = useRef(new Map());
  const wsRef = useRef(null);
  const pollingRef = useRef(null);
  const lastRoundRef = useRef(0);
  const STORAGE_KEY = "pictactic:task-cache";

  const upsertTaskCache = useCallback((taskId, updates) => {
    if (!taskId) return;
    const prev = taskCacheRef.current.get(taskId) || {};
    taskCacheRef.current.set(taskId, { ...prev, ...updates });
  }, []);

  const upsertStoredTaskCache = useCallback((taskId, updates) => {
    if (!taskId || typeof window === "undefined") return;
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      const cache = raw ? JSON.parse(raw) : {};
      const prev = cache[taskId] || {};
      cache[taskId] = { ...prev, ...updates };
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(cache));
    } catch {
      // Ignore storage errors (quota/private mode)
    }
  }, []);

  const getCachedTask = useCallback((taskId) => {
    if (!taskId) return null;
    const memory = taskCacheRef.current.get(taskId);
    if (memory) return memory;
    if (typeof window === "undefined") return null;
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      const cache = raw ? JSON.parse(raw) : {};
      return cache[taskId] || null;
    } catch {
      return null;
    }
  }, []);

  const cleanup = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  }, []);

  useEffect(() => {
    return cleanup;
  }, [cleanup]);

  const pollStatus = useCallback(async (taskId) => {
    try {
      const taskProgress = await apiClient.getTaskStatus(taskId);
      setProgress(taskProgress);

      if (["completed", "failed", "cancelled"].includes(taskProgress.status)) {
        cleanup();
        if (taskProgress.status === "completed") {
          setStatus("completing"); // 过渡状态：正在加载结果
          const taskResult = await apiClient.getTaskResult(taskId);
          setResult(taskResult);
          setStreamedImages([]); // result 加载完成后再清空
          setStatus("completed");
        } else if (taskProgress.status === "failed") {
          setStreamedImages([]);
          setStatus("failed");
          setError(taskProgress.error || "Generation failed");
        } else {
          setStreamedImages([]);
          setStatus("cancelled");
        }
      }
    } catch (err) {
      console.error("Error polling status:", err);
    }
  }, [cleanup]);

  const connectWebSocket = useCallback((taskId) => {
    const wsUrl = apiClient.getWebSocketUrl(taskId);
    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onmessage = async (event) => {
        let data;
        try {
          data = JSON.parse(event.data);
        } catch {
          console.error("Failed to parse WS message:", event.data);
          return;
        }

        // Handle terminal states first (includes both status and potential error)
        if (["completed", "failed", "cancelled"].includes(data.status)) {
          cleanup();
          // 不立即清空 streamedImages，等 result 加载完成后再清空
          if (data.status === "completed") {
            setStatus("completing"); // 过渡状态：正在加载结果
            const taskResult = await apiClient.getTaskResult(taskId);
            setResult(taskResult);
            setStreamedImages([]); // result 加载完成后再清空
            setStatus("completed");
          } else if (data.status === "failed") {
            setStreamedImages([]);
            setStatus("failed");
            setError(data.error || "Generation failed");
          } else {
            setStreamedImages([]);
            setStatus("cancelled");
          }
          return;
        }

        // Handle pure error messages without status (backward compatibility for old format)
        if (data.error) {
          setError(data.error);
          return;
        }

        // Handle streamed individual image
        if (data.type === "image_ready" && data.image) {
          setStreamedImages((prev) => [...prev, data.image]);
          return;
        }

        setProgress(data);

        // Clear streamed images only when round changes (not on every "generating" status)
        if (data.current_round && data.current_round !== lastRoundRef.current) {
          lastRoundRef.current = data.current_round;
          setStreamedImages([]);
        }
      };

      ws.onerror = () => {
        console.warn("WebSocket error, falling back to polling");
        ws.close();
        pollingRef.current = setInterval(() => pollStatus(taskId), 2000);
      };

      ws.onclose = () => {
        wsRef.current = null;
      };
    } catch {
      pollingRef.current = setInterval(() => pollStatus(taskId), 2000);
    }
  }, [cleanup, pollStatus]);

  const startGeneration = useCallback(async (request) => {
    cleanup();
    setStatus("creating");
    setProgress(null);
    setResult(null);
    setError(null);
    setStreamedImages([]);
    lastRoundRef.current = 0;
    const nextGenerationConfig = {
      aspectRatio: request.aspect_ratio || "16:9",
      imagesPerRound: request.images_per_round || 5,
      size: request.size || "2K",
    };
    setGenerationConfig(nextGenerationConfig);

    try {
      const task = await apiClient.createTask(request);
      taskIdRef.current = task.task_id;

      // 保存任务输入上下文（在创建 task 之后，这样才能访问 task_id）
      const nextTaskContext = {
        taskId: task.task_id,
        prompt: request.prompt,
        templateImages: request.template_images || [],
        mode: request.template_images?.length > 0 ? "image_to_image" : "text_to_image",
        aspectRatio: request.aspect_ratio || "16:9",
        size: request.size || "2K",
      };
      setTaskContext(nextTaskContext);

      upsertTaskCache(task.task_id, {
        generationConfig: nextGenerationConfig,
        taskContext: nextTaskContext,
      });
      upsertStoredTaskCache(task.task_id, {
        generationConfig: nextGenerationConfig,
        taskContext: {
          prompt: nextTaskContext.prompt,
          mode: nextTaskContext.mode,
          aspectRatio: nextTaskContext.aspectRatio,
          size: nextTaskContext.size,
        },
      });
      setStatus("running");
      connectWebSocket(task.task_id);
      pollingRef.current = setInterval(() => pollStatus(task.task_id), 3000);
    } catch (err) {
      // Network error: the request may have reached the backend and created the task,
      // but the response was lost (e.g. proxy/CDN dropped the connection).
      // Try to recover by finding the just-created task in the task history.
      if (err instanceof TypeError) {
        try {
          await new Promise((r) => setTimeout(r, 2000));
          const history = await apiClient.listTaskHistory(5, 0);
          const match = history?.find((t) => {
            if (t.original_prompt !== request.prompt) return false;
            if (!t.created_at) return false;
            return Date.now() - new Date(t.created_at).getTime() < 30000;
          });
          if (match) {
            const recoveredId = match.task_id;
            taskIdRef.current = recoveredId;
            const nextTaskContext = {
              taskId: recoveredId,
              prompt: request.prompt,
              templateImages: request.template_images || [],
              mode: request.template_images?.length > 0 ? "image_to_image" : "text_to_image",
              aspectRatio: request.aspect_ratio || "16:9",
              size: request.size || "2K",
            };
            setTaskContext(nextTaskContext);
            upsertTaskCache(recoveredId, {
              generationConfig: nextGenerationConfig,
              taskContext: nextTaskContext,
            });
            upsertStoredTaskCache(recoveredId, {
              generationConfig: nextGenerationConfig,
              taskContext: {
                prompt: nextTaskContext.prompt,
                mode: nextTaskContext.mode,
                aspectRatio: nextTaskContext.aspectRatio,
                size: nextTaskContext.size,
              },
            });
            setStatus("running");
            connectWebSocket(recoveredId);
            pollingRef.current = setInterval(() => pollStatus(recoveredId), 3000);
            return;
          }
        } catch {
          // Recovery attempt failed, fall through to show original error
        }
      }
      setStatus("failed");
      setError(err instanceof Error ? err.message : "Failed to start generation");
    }
  }, [cleanup, connectWebSocket, pollStatus, upsertTaskCache, upsertStoredTaskCache]);

  const resumeTask = useCallback(async (taskId, taskMeta = null) => {
    cleanup();
    taskIdRef.current = taskId;
    setStatus("running");
    setProgress(null);
    setResult(null);
    setError(null);
    setStreamedImages([]);
    lastRoundRef.current = 0;

    const cached = getCachedTask(taskId);
    const metaConfig = taskMeta?.config || {};
    const metaImagesPerRound = taskMeta?.images_per_round ?? metaConfig.images_per_round;
    const metaAspectRatio = taskMeta?.aspect_ratio ?? metaConfig.aspect_ratio;
    const metaSize = taskMeta?.size ?? metaConfig.size;
    const fallbackConfig = {
      aspectRatio: cached?.generationConfig?.aspectRatio
        || metaAspectRatio
        || generationConfig?.aspectRatio
        || "16:9",
      imagesPerRound: cached?.generationConfig?.imagesPerRound
        || metaImagesPerRound
        || generationConfig?.imagesPerRound
        || 5,
      size: cached?.generationConfig?.size || metaSize || generationConfig?.size || "2K",
    };
    setGenerationConfig(fallbackConfig);

    const fallbackPrompt = cached?.taskContext?.prompt || taskMeta?.original_prompt || "";

    if (cached?.taskContext) {
      setTaskContext({
        ...cached.taskContext,
        taskId: taskId, // 显式添加 taskId（因为缓存中可能没有存储）
      });
    } else if (taskMeta?.original_prompt) {
      const fallbackContext = {
        taskId: taskId,
        prompt: taskMeta.original_prompt,
        templateImages: taskMeta.template_images || [],
        mode: (taskMeta.template_images?.length || 0) > 0 ? "image_to_image" : "text_to_image",
        aspectRatio: fallbackConfig.aspectRatio || "16:9",
        size: fallbackConfig.size || "2K",
      };
      setTaskContext(fallbackContext);
      upsertTaskCache(taskId, { taskContext: fallbackContext, generationConfig: fallbackConfig });
      upsertStoredTaskCache(taskId, {
        generationConfig: fallbackConfig,
        taskContext: {
          prompt: fallbackContext.prompt,
          mode: fallbackContext.mode,
          aspectRatio: fallbackContext.aspectRatio,
          size: fallbackContext.size,
        },
      });
    } else {
      setTaskContext(null);
    }

    try {
      const taskProgress = await apiClient.getTaskStatus(taskId);
      setProgress(taskProgress);
      lastRoundRef.current = taskProgress?.current_round || 0;
      if (taskProgress?.recent_images?.length) {
        setStreamedImages(taskProgress.recent_images);
      }

      const progressConfig = {
        aspectRatio: taskProgress?.aspect_ratio || fallbackConfig.aspectRatio || "16:9",
        imagesPerRound: taskProgress?.images_per_round || fallbackConfig.imagesPerRound || 5,
        size: taskProgress?.size || fallbackConfig.size || "2K",
      };
      setGenerationConfig(progressConfig);
      upsertTaskCache(taskId, { generationConfig: progressConfig });
      upsertStoredTaskCache(taskId, {
        generationConfig: progressConfig,
      });

      const progressTemplateImages = taskProgress?.template_images || [];
      if (taskProgress?.original_prompt || progressTemplateImages.length > 0) {
        // Preserve existing templateImages if API returns empty (API strips them for perf)
        const existingTemplateImages = cached?.taskContext?.templateImages || [];
        const effectiveTemplateImages = progressTemplateImages.length > 0
          ? progressTemplateImages
          : existingTemplateImages;
        const progressContext = {
          taskId: taskId,
          prompt: taskProgress?.original_prompt || fallbackPrompt,
          templateImages: effectiveTemplateImages,
          mode: effectiveTemplateImages.length > 0 ? "image_to_image" : "text_to_image",
          aspectRatio: progressConfig.aspectRatio,
          size: progressConfig.size,
        };
        setTaskContext(progressContext);
        upsertTaskCache(taskId, { taskContext: progressContext });
        upsertStoredTaskCache(taskId, {
          taskContext: {
            prompt: progressContext.prompt,
            mode: progressContext.mode,
            aspectRatio: progressContext.aspectRatio,
            size: progressContext.size,
          },
        });
      }

      if (["completed", "failed", "cancelled"].includes(taskProgress.status)) {
        cleanup();
        if (taskProgress.status === "completed") {
          setStatus("completing");
          const taskResult = await apiClient.getTaskResult(taskId);
          setResult(taskResult);
          setStreamedImages([]);
          setStatus("completed");
        } else if (taskProgress.status === "failed") {
          setStreamedImages([]);
          setStatus("failed");
          setError(taskProgress.error || "Generation failed");
        } else {
          setStreamedImages([]);
          setStatus("cancelled");
        }
        return;
      }

      connectWebSocket(taskId);
      pollingRef.current = setInterval(() => pollStatus(taskId), 3000);
    } catch (err) {
      setStatus("failed");
      setError(err instanceof Error ? err.message : "Failed to load task status");
    }
  }, [cleanup, connectWebSocket, pollStatus, generationConfig, getCachedTask, upsertTaskCache, upsertStoredTaskCache]);

  const cancelGeneration = useCallback(async () => {
    if (!taskIdRef.current) return;
    try {
      await apiClient.cancelTask(taskIdRef.current);
      cleanup();
      setStatus("cancelled");
    } catch (err) {
      console.error("Error cancelling task:", err);
    }
  }, [cleanup]);

  const loadResult = useCallback(async (taskId) => {
    cleanup();
    setStatus("loading");
    setProgress(null);
    setResult(null);
    setError(null);
    const cached = getCachedTask(taskId);
    const fallbackConfig = {
      aspectRatio: cached?.generationConfig?.aspectRatio
        || generationConfig?.aspectRatio
        || "16:9",
      imagesPerRound: cached?.generationConfig?.imagesPerRound
        || generationConfig?.imagesPerRound
        || 5,
      size: cached?.generationConfig?.size || generationConfig?.size || "2K",
    };
    setGenerationConfig(fallbackConfig);
    const cachedTaskContext = cached?.taskContext
      ? { ...cached.taskContext, taskId }
      : null;

    if (cachedTaskContext) {
      setTaskContext(cachedTaskContext);
      upsertTaskCache(taskId, { taskContext: cachedTaskContext });
    } else {
      setTaskContext(null);
    }
    try {
      const taskResult = await apiClient.getTaskResult(taskId);
      const resultConfig = taskResult?.config || {};
      const resolvedConfig = {
        aspectRatio: resultConfig.aspect_ratio || fallbackConfig.aspectRatio || "16:9",
        imagesPerRound: resultConfig.images_per_round || fallbackConfig.imagesPerRound || 5,
        size: resultConfig.size || fallbackConfig.size || "2K",
      };
      setGenerationConfig(resolvedConfig);
      upsertTaskCache(taskId, { generationConfig: resolvedConfig });
      upsertStoredTaskCache(taskId, { generationConfig: resolvedConfig });

      setResult(taskResult);
      setStatus(
        taskResult.status === "failed" ? "failed" :
        taskResult.status === "cancelled" ? "cancelled" : "completed"
      );
      if (taskResult.error) setError(taskResult.error);
      const resultTemplateImages = taskResult?.template_images || [];
      // Preserve cached templateImages if API returns empty (API strips base64 for perf)
      const cachedTemplateImages = cachedTaskContext?.templateImages || [];
      const effectiveResultTemplateImages = resultTemplateImages.length > 0
        ? resultTemplateImages
        : cachedTemplateImages;
      if (!cachedTaskContext && taskResult.original_prompt) {
        const fallbackContext = {
          taskId: taskId,
          prompt: taskResult.original_prompt,
          templateImages: effectiveResultTemplateImages,
          mode: effectiveResultTemplateImages.length > 0 ? "image_to_image" : "text_to_image",
          aspectRatio: resolvedConfig.aspectRatio || "16:9",
          size: resolvedConfig.size || "2K",
        };
        setTaskContext(fallbackContext);
        upsertTaskCache(taskId, { taskContext: fallbackContext, generationConfig: resolvedConfig });
        upsertStoredTaskCache(taskId, {
          generationConfig: resolvedConfig,
          taskContext: {
            prompt: fallbackContext.prompt,
            mode: fallbackContext.mode,
            aspectRatio: fallbackContext.aspectRatio,
            size: fallbackContext.size,
          },
        });
      }
      setProgress({
        original_prompt: taskResult.original_prompt,
        enhanced_prompt: taskResult.enhanced_prompt,
        message: taskResult.error || "",
        current_round: taskResult.total_rounds,
        max_rounds: taskResult.total_rounds,
        images_generated: taskResult.total_images,
        images_in_current_round: 0,
        current_best_score: taskResult.best_images?.[0]?.score || 0,
      });
    } catch (err) {
      setStatus("failed");
      setError(err instanceof Error ? err.message : "Failed to load result");
    }
  }, [cleanup, generationConfig, getCachedTask, upsertTaskCache, upsertStoredTaskCache]);

  const reset = useCallback(() => {
    cleanup();
    taskIdRef.current = null;
    setStatus("idle");
    setProgress(null);
    setResult(null);
    setError(null);
    setStreamedImages([]);
    lastRoundRef.current = 0;
    setGenerationConfig(null);
    setTaskContext(null);
  }, [cleanup]);

  return {
    status, progress, result, error, generationConfig, streamedImages, taskContext,
    startGeneration, cancelGeneration, loadResult, resumeTask, reset,
  };
}
