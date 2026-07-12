import { useState, useCallback, useRef, useEffect } from "react";
import { apiClient } from "@/lib/api";

export function useVideoGeneration() {
  const [status, setStatus] = useState("idle");
  const [progress, setProgress] = useState(null);
  const [result, setResult] = useState(null);
  const [error, setError] = useState(null);
  const [videoConfig, setVideoConfig] = useState(null);
  const [streamedVideos, setStreamedVideos] = useState([]);
  const [taskContext, setTaskContext] = useState(null);

  const taskIdRef = useRef(null);
  const wsRef = useRef(null);
  const pollingRef = useRef(null);
  const lastRoundRef = useRef(0);

  const cleanup = useCallback(() => {
    taskIdRef.current = null;
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
    // Skip if this task is no longer the active one
    if (taskIdRef.current !== taskId) return;
    try {
      const taskStatus = await apiClient.getVideoTaskStatus(taskId);
      if (taskIdRef.current !== taskId) return;
      setProgress(taskStatus);

      if (["completed", "failed", "cancelled"].includes(taskStatus.status)) {
        cleanup();
        if (taskStatus.status === "completed") {
          setStatus("completing");
          try {
            const taskResult = await apiClient.getVideoTaskResult(taskId);
            setResult(taskResult);
            setStreamedVideos([]);
            setStatus("completed");
          } catch (fetchErr) {
            console.error("Failed to fetch video result:", fetchErr);
            setError("Failed to fetch video result");
            setStatus("failed");
          }
        } else if (taskStatus.status === "failed") {
          setStreamedVideos([]);
          setStatus("failed");
          setError(taskStatus.error || "Video generation failed");
        } else {
          setStreamedVideos([]);
          setStatus("cancelled");
        }
      }
    } catch (err) {
      console.error("Error polling video status:", err);
    }
  }, [cleanup]);

  const connectWebSocket = useCallback((taskId) => {
    const wsUrl = apiClient.getVideoWebSocketUrl(taskId);
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

        if (data.error) {
          setError(data.error);
          return;
        }

        // Handle streamed individual video
        if (data.type === "video_ready" && data.video) {
          setStreamedVideos((prev) => [...prev, data.video]);
          return;
        }

        // Handle video progress update
        if (data.type === "video_progress") {
          setProgress((prev) => ({
            ...prev,
            current_video_progress: data.progress_percent,
            current_video_eta: data.eta,
            current_video_index: data.video_index,
          }));
          return;
        }

        // Handle status change
        if (data.type === "status_change") {
          setProgress((prev) => ({
            ...prev,
            status: data.status,
            current_round: data.current_round,
            message: data.message,
          }));
          return;
        }

        // General progress update
        if (data.current_round || data.status) {
          setProgress(data);
        }

        // Clear streamed videos when round changes
        if (data.current_round && data.current_round !== lastRoundRef.current) {
          lastRoundRef.current = data.current_round;
          setStreamedVideos([]);
        }

        if (["completed", "failed", "cancelled"].includes(data.status)) {
          cleanup();
          if (data.status === "completed") {
            setStatus("completing");
            try {
              const taskResult = await apiClient.getVideoTaskResult(taskId);
              setResult(taskResult);
              setStreamedVideos([]);
              setStatus("completed");
            } catch (fetchErr) {
              console.error("Failed to fetch video result:", fetchErr);
              setError("Failed to fetch video result");
              setStatus("failed");
            }
          } else if (data.status === "failed") {
            setStreamedVideos([]);
            setStatus("failed");
            setError(data.error || "Video generation failed");
          } else {
            setStreamedVideos([]);
            setStatus("cancelled");
          }
        }
      };

      ws.onerror = () => {
        console.warn("Video WebSocket error, falling back to polling");
        wsRef.current = null;
        ws.close();
        if (taskIdRef.current === taskId && !pollingRef.current) {
          pollingRef.current = setInterval(() => pollStatus(taskId), 3000);
        }
      };

      ws.onclose = () => {
        // Only fallback to polling if this WS wasn't intentionally closed via cleanup()
        // cleanup() sets wsRef.current = null before the async onclose fires
        if (wsRef.current === ws) {
          wsRef.current = null;
          if (!pollingRef.current && taskIdRef.current === taskId) {
            pollingRef.current = setInterval(() => pollStatus(taskId), 3000);
          }
        }
      };
    } catch {
      if (taskIdRef.current === taskId && !pollingRef.current) {
        pollingRef.current = setInterval(() => pollStatus(taskId), 3000);
      }
    }
  }, [cleanup, pollStatus]);

  const startVideoGeneration = useCallback(async (request) => {
    cleanup();
    setStatus("creating");
    setProgress(null);
    setResult(null);
    setError(null);
    setStreamedVideos([]);
    lastRoundRef.current = 0;

    const nextVideoConfig = {
      duration: request.duration || 5,
      resolution: request.resolution || null,
      aspectRatio: request.aspect_ratio || "16:9",
      videosPerRound: request.videos_per_round || 2,
    };
    setVideoConfig(nextVideoConfig);

    // Set taskContext immediately so the UI shows correct provider before API call
    const nextTaskContext = {
      taskId: null,
      prompt: request.prompt,
      startImage: request.start_image || null,
      endImage: request.end_image || null,
      mode: request.start_image ? "image_to_video" : "text_to_video",
      provider: request.provider || "kling",
    };
    setTaskContext(nextTaskContext);

    try {
      const task = await apiClient.createVideoTask(request);
      taskIdRef.current = task.task_id;

      // Update taskContext with actual task ID
      setTaskContext({ ...nextTaskContext, taskId: task.task_id });

      setStatus("running");
      connectWebSocket(task.task_id);
    } catch (err) {
      setStatus("failed");
      setError(err instanceof Error ? err.message : "Failed to start video generation");
    }
  }, [cleanup, connectWebSocket]);

  const resumeVideoTask = useCallback(async (taskId, taskMeta = null) => {
    cleanup();
    taskIdRef.current = taskId;
    setStatus("running");
    setProgress(null);
    setResult(null);
    setError(null);
    setStreamedVideos([]);
    lastRoundRef.current = 0;

    // Set fallback config immediately so GeneratingState renders during fetch
    const metaConfig = taskMeta?.config || {};
    const fallbackConfig = {
      duration: metaConfig.duration || 5,
      resolution: metaConfig.resolution || null,
      aspectRatio: metaConfig.aspect_ratio || "16:9",
      videosPerRound: metaConfig.videos_per_round || 2,
    };
    setVideoConfig(fallbackConfig);

    // Set fallback task context
    if (taskMeta?.original_prompt) {
      setTaskContext({
        taskId,
        prompt: taskMeta.original_prompt,
        startImage: taskMeta.start_image || null,
        endImage: taskMeta.end_image || null,
        mode: taskMeta.start_image ? "image_to_video" : "text_to_video",
        provider: metaConfig.provider || "kling",
      });
    }

    try {
      const taskStatus = await apiClient.getVideoTaskStatus(taskId);
      if (taskIdRef.current !== taskId) return;
      setProgress(taskStatus);

      // Update config from status response if available
      if (taskStatus.config) {
        const statusConfig = {
          duration: taskStatus.config.duration || fallbackConfig.duration,
          resolution: taskStatus.config.resolution || fallbackConfig.resolution,
          aspectRatio: taskStatus.config.aspect_ratio || fallbackConfig.aspectRatio,
          videosPerRound: taskStatus.config.videos_per_round || fallbackConfig.videosPerRound,
        };
        setVideoConfig(statusConfig);
      }

      // Update task context from status
      if (taskStatus.original_prompt) {
        setTaskContext({
          taskId,
          prompt: taskStatus.original_prompt,
          startImage: taskStatus.start_image || taskMeta?.start_image || null,
          endImage: taskStatus.end_image || taskMeta?.end_image || null,
          mode: (taskStatus.start_image || taskMeta?.start_image) ? "image_to_video" : "text_to_video",
          provider: taskStatus.config?.provider || metaConfig.provider || "kling",
        });
      }

      // If already terminal, load the result
      if (["completed", "failed", "cancelled"].includes(taskStatus.status)) {
        if (taskStatus.status === "completed") {
          setStatus("completing");
          try {
            const taskResult = await apiClient.getVideoTaskResult(taskId);
            setResult(taskResult);
            setStreamedVideos([]);
            setStatus("completed");
          } catch (fetchErr) {
            console.error("Failed to fetch video result:", fetchErr);
            setError("Failed to fetch video result");
            setStatus("failed");
          }
        } else if (taskStatus.status === "failed") {
          setStatus("failed");
          setError(taskStatus.error || "Video generation failed");
        } else {
          setStatus("cancelled");
        }
        return;
      }

      // Still running — connect WS to track progress
      connectWebSocket(taskId);
    } catch (err) {
      setStatus("failed");
      setError(err instanceof Error ? err.message : "Failed to resume video task");
    }
  }, [cleanup, connectWebSocket]);

  const loadVideoResult = useCallback(async (taskId) => {
    cleanup();
    taskIdRef.current = taskId;
    setStatus("loading");
    setProgress(null);
    setResult(null);
    setError(null);
    // Preserve existing videoConfig or set a fallback, so GeneratingState can render during fetch
    setVideoConfig((prev) => prev || { duration: 5, resolution: null, aspectRatio: "16:9", videosPerRound: 2 });

    try {
      const taskResult = await apiClient.getVideoTaskResult(taskId);
      if (taskIdRef.current !== taskId) return;
      setResult(taskResult);

      const resolvedConfig = {
        duration: taskResult.config?.duration || 5,
        resolution: taskResult.config?.resolution || null,
        aspectRatio: taskResult.config?.aspect_ratio || "16:9",
        videosPerRound: taskResult.config?.videos_per_round || 2,
      };
      setVideoConfig(resolvedConfig);

      if (taskResult.original_prompt) {
        setTaskContext({
          taskId,
          prompt: taskResult.original_prompt,
          startImage: taskResult.start_image || null,
          endImage: taskResult.end_image || null,
          mode: taskResult.start_image ? "image_to_video" : "text_to_video",
          provider: taskResult.config?.provider || "kling",
        });
      }

      setStatus(
        taskResult.status === "failed" ? "failed" :
        taskResult.status === "cancelled" ? "cancelled" : "completed"
      );
      if (taskResult.error) setError(taskResult.error);

      setProgress({
        original_prompt: taskResult.original_prompt,
        enhanced_prompt: taskResult.enhanced_prompt,
        message: taskResult.error || "",
        current_round: taskResult.total_rounds,
        max_rounds: taskResult.total_rounds,
        videos_generated: taskResult.total_videos,
      });
    } catch (err) {
      if (taskIdRef.current !== taskId) return;
      setStatus("failed");
      setError(err instanceof Error ? err.message : "Failed to load video result");
    }
  }, [cleanup]);

  const cancelVideoGeneration = useCallback(async () => {
    if (!taskIdRef.current) return;
    try {
      await apiClient.cancelVideoTask(taskIdRef.current);
      cleanup();
      setStatus("cancelled");
    } catch (err) {
      console.error("Error cancelling video task:", err);
    }
  }, [cleanup]);

  const reset = useCallback(() => {
    cleanup();
    setStatus("idle");
    setProgress(null);
    setResult(null);
    setError(null);
    setStreamedVideos([]);
    lastRoundRef.current = 0;
    setVideoConfig(null);
    setTaskContext(null);
  }, [cleanup]);

  return {
    status,
    progress,
    result,
    error,
    videoConfig,
    streamedVideos,
    taskContext,
    startVideoGeneration,
    resumeVideoTask,
    cancelVideoGeneration,
    loadVideoResult,
    reset,
  };
}
