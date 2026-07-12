import { useState, useCallback, useEffect, useRef } from "react";
import { apiClient } from "@/lib/api";

const RUNNING_STATUSES = ["pending", "enhancing", "generating", "reviewing"];

export function useTaskHistory() {
  const [tasks, setTasks] = useState([]);
  const [selectedTaskId, setSelectedTaskId] = useState(null);
  const [loading, setLoading] = useState(false);
  const pollingRef = useRef(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const history = await apiClient.listTaskHistory(50, 0);
      setTasks(history);
    } catch (err) {
      console.error("Failed to load task history:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  const selectTask = useCallback((taskId) => {
    setSelectedTaskId(taskId);
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedTaskId(null);
  }, []);

  const deleteTask = useCallback(async (taskId) => {
    await apiClient.deleteTask(taskId);
    setTasks((prev) => prev.filter((t) => t.task_id !== taskId));
    if (selectedTaskId === taskId) {
      setSelectedTaskId(null);
    }
  }, [selectedTaskId]);

  // 检查是否有运行中任务
  const hasRunningTasks = tasks.some(t => RUNNING_STATUSES.includes(t.status));

  // 动态轮询：有运行中任务时刷新（5秒）
  useEffect(() => {
    if (hasRunningTasks) {
      pollingRef.current = setInterval(refresh, 5000);
    } else {
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    }
    return () => {
      if (pollingRef.current) clearInterval(pollingRef.current);
    };
  }, [hasRunningTasks, refresh]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  useEffect(() => {
    if (!selectedTaskId) return;
    const exists = tasks.some((task) => task.task_id === selectedTaskId);
    if (!exists) {
      setSelectedTaskId(null);
    }
  }, [tasks, selectedTaskId]);

  return { tasks, selectedTaskId, loading, hasRunningTasks, selectTask, clearSelection, deleteTask, refresh };
}
