/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/hooks/useSessionRouter.js
 * GitHub: https://github.com/NanmiCoder
 * Licensed under NON-COMMERCIAL LEARNING LICENSE 1.1
 *
 * 声明：本代码仅供学习和研究目的使用。使用者应遵守以下原则：
 * 1. 不得用于任何商业用途。
 * 2. 使用时应遵守目标平台的使用条款和robots.txt规则。
 * 3. 不得进行大规模爬取或对平台造成运营干扰。
 * 4. 应合理控制请求频率，避免给目标平台带来不必要的负担。
 * 5. 不得用于任何非法或不当的用途。
 *
 * 详细许可条款请参阅项目根目录下的LICENSE文件。
 * 使用本代码即表示您同意遵守上述原则和LICENSE中的所有条款。
 */

import { useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { getSessionState } from '../services/api';

/**
 * Session 路由同步 Hook
 *
 * 负责:
 * - 从 URL 解析 session ID
 * - URL 变化时验证并同步到应用状态
 * - 应用状态变化时同步到 URL
 * - 处理无效 session ID 重定向
 *
 * @param {object} options
 * @param {string|null} options.currentId - 当前选中的会话 ID
 * @param {function} options.setCurrentId - 设置当前会话 ID
 * @param {Array} options.conversations - 会话列表
 * @param {boolean} options.conversationsLoaded - 会话列表是否已加载
 */
export function useSessionRouter({
  currentId,
  setCurrentId,
  conversations,
  conversationsLoaded,
}) {
  const { sessionId: urlSessionId } = useParams();
  const navigate = useNavigate();
  const location = useLocation();

  // 用于防止循环更新的标记
  const isNavigatingRef = useRef(false);
  const lastUrlSessionIdRef = useRef(null);

  // 导航到首页（新会话）
  const navigateToHome = useCallback(() => {
    isNavigatingRef.current = true;
    navigate('/', { replace: true });
    setTimeout(() => {
      isNavigatingRef.current = false;
    }, 0);
  }, [navigate]);

  // 导航到特定会话
  const navigateToSession = useCallback((sessionId) => {
    if (!sessionId) {
      navigateToHome();
      return;
    }
    isNavigatingRef.current = true;
    navigate(`/session/${sessionId}`, { replace: false });
    setTimeout(() => {
      isNavigatingRef.current = false;
    }, 0);
  }, [navigate, navigateToHome]);

  // URL 变化时同步到应用状态
  useEffect(() => {
    // 如果是由我们触发的导航，跳过
    if (isNavigatingRef.current) {
      return;
    }

    // 如果 URL 没有变化，跳过
    if (urlSessionId === lastUrlSessionIdRef.current) {
      return;
    }
    lastUrlSessionIdRef.current = urlSessionId;

    // 等待会话列表加载完成
    if (!conversationsLoaded) {
      return;
    }

    // URL 中没有 sessionId，清空当前会话
    if (!urlSessionId) {
      if (currentId !== null) {
        setCurrentId(null);
      }
      return;
    }

    // URL 中有 sessionId，验证并同步
    const validateAndSync = async () => {
      // 先检查本地会话列表
      const existsLocally = conversations.some(
        c => c.session_id === urlSessionId
      );

      if (existsLocally) {
        // 本地存在，直接同步
        if (currentId !== urlSessionId) {
          setCurrentId(urlSessionId);
        }
        return;
      }

      // 本地不存在，向后端验证
      try {
        const sessionState = await getSessionState(urlSessionId);
        if (sessionState) {
          // 后端存在，同步状态
          if (currentId !== urlSessionId) {
            setCurrentId(urlSessionId);
          }
        } else {
          // 后端不存在，重定向到首页
          console.warn(`Session ${urlSessionId} not found, redirecting to home`);
          navigateToHome();
        }
      } catch (error) {
        console.error('Failed to validate session:', error);
        // 验证失败，重定向到首页
        navigateToHome();
      }
    };

    validateAndSync();
  }, [
    urlSessionId,
    currentId,
    setCurrentId,
    conversations,
    conversationsLoaded,
    navigateToHome,
  ]);

  // 应用状态变化时同步到 URL（当用户通过侧边栏切换会话时）
  // 注意：这个 effect 只在 currentId 由应用内部改变时触发
  // 如果是由 URL 变化触发的，上面的 effect 会处理
  useEffect(() => {
    // 如果正在导航中，跳过（防止循环）
    if (isNavigatingRef.current) {
      return;
    }

    // 检查当前 URL 是否已经匹配
    const currentUrlSessionId = location.pathname.startsWith('/session/')
      ? location.pathname.split('/session/')[1]
      : null;

    if (currentId === null && currentUrlSessionId !== null) {
      // 应用状态为空，但 URL 有值，需要导航到首页
      // 但这种情况由 URL 变化 effect 处理
      return;
    }

    if (currentId !== null && currentId !== currentUrlSessionId) {
      // 应用状态有值，但 URL 不匹配，需要更新 URL
      navigateToSession(currentId);
    }
  }, [currentId, location.pathname, navigateToSession]);

  return {
    urlSessionId,
    navigateToHome,
    navigateToSession,
  };
}
