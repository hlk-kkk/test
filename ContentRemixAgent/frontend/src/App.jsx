/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/App.jsx
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

import React, { useState, useRef, useCallback, useEffect } from 'react';
import { Routes, Route } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import { ChatContainer } from './components/ChatContainer';
import ChatInputFooter from './components/ChatInputFooter';
import LanguageSwitcher from './i18n/LanguageSwitcher';
import LandingPage from './components/LandingPage';
import { useChat } from './hooks/useChat';
import { useConversations } from './hooks/useConversations';
import { useTitlePolling } from './hooks/useTitlePolling';
import { useSessionRouter } from './hooks/useSessionRouter';
import { AuthProvider, PublicOnlyRoute, ProtectedRoute, useAuth } from './hooks/useAuth';
import { ThemeProvider } from './hooks/useTheme';
import ThemeSwitcher from './components/ThemeSwitcher';
import {
  LoginPage,
  RegisterPage,
  OAuthCallback,
  ForgotPasswordPage,
  ResetPasswordPage,
  VerifyEmailPage,
} from './components/Auth';
import { getAccessToken } from './services/auth';
import { fetchLLMConfigList } from './services/api';

function AppContent() {
  // UI State
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // 每会话模型选择
  const [selectedModel, setSelectedModel] = useState(null);
  const [defaultModel, setDefaultModel] = useState(null);

  // 新创建会话的信息（用于标题轮询）
  const [newSessionInfo, setNewSessionInfo] = useState(null);

  // 会话管理
  const {
    conversations,
    currentId,
    createConversation,
    deleteConversation,
    selectConversation,
    addConversation,
    updateConversation,
    loadConversations,
    setCurrentId,
    loading: conversationsLoading,
  } = useConversations();

  // 初始化加载默认模型
  useEffect(() => {
    fetchLLMConfigList()
      .then(data => {
        if (data?.items) {
          const active = data.items.find(c => c.is_active);
          if (active) {
            setDefaultModel(active.config_name);
            if (!selectedModel) {
              setSelectedModel(active.config_name);
            }
          }
        }
      })
      .catch(err => console.warn('Failed to load LLM configs:', err));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // 当 currentId 变化时（包括刷新页面路由恢复），从 conversation 同步模型选择
  useEffect(() => {
    if (!currentId || conversationsLoading) return;
    const conv = conversations.find(c => c.session_id === currentId);
    if (conv?.llm_config_name) {
      setSelectedModel(conv.llm_config_name);
    } else if (defaultModel) {
      setSelectedModel(defaultModel);
    }
  }, [currentId, conversations, conversationsLoading, defaultModel]);

  // 路由同步
  const { navigateToHome, navigateToSession } = useSessionRouter({
    currentId,
    setCurrentId,
    conversations,
    conversationsLoaded: !conversationsLoading,
  });

  // 标题轮询 Hook
  const { startPolling } = useTitlePolling(
    newSessionInfo?.session_id,
    newSessionInfo?.title,
    (newTitle) => {
      // 标题更新回调
      if (newSessionInfo?.session_id) {
        updateConversation(newSessionInfo.session_id, { title: newTitle });
        setNewSessionInfo(null); // 清除状态
      }
    },
    {
      delay: 3000,      // 3 秒后开始轮询
      interval: 2000,   // 每 2 秒轮询一次
      maxRetries: 5,    // 最多重试 5 次
    }
  );

  // 处理新会话创建（当后端返回 session_id 时）
  const handleSessionCreated = useCallback((session) => {
    // 保存模型信息到会话数据
    if (selectedModel) {
      session.llm_config_name = selectedModel;
    }
    addConversation(session);
    // 导航到新会话 URL
    navigateToSession(session.session_id);
    // 保存新会话信息，useEffect 会自动启动标题轮询
    setNewSessionInfo(session);
  }, [addConversation, navigateToSession, selectedModel]);

  // 当 newSessionInfo 变化时启动标题轮询
  // 使用 useEffect 避免闭包陷阱（确保 startPolling 捕获最新的 sessionId）
  useEffect(() => {
    if (newSessionInfo?.session_id) {
      // 延迟启动轮询，确保状态已更新
      const timer = setTimeout(() => startPolling(), 100);
      return () => clearTimeout(timer);
    }
  }, [newSessionInfo?.session_id, startPolling]);

  // 获取真实 token（如果已登录）
  const authToken = getAccessToken() || 'ANONYMOUS';

  // 聊天功能
  const {
    messages,
    streamingMessage,
    loading: chatLoading,
    sendMessage,
    clearMessages,
    retryFromIndex,
    stopGeneration,
  } = useChat(authToken, {
    currentSessionId: currentId,
    onSessionCreated: handleSessionCreated,
    onSessionUpdated: loadConversations,
  });

  const chatContainerRef = useRef(null);
  const scrollContainerRef = useRef(null);

  // --- Handlers ---

  // 发送分析请求（从底部输入域）
  // 接收 url 和 mode 参数，mode 由 ChatInputFooter 的模式选择器传入
  const handleAnalyze = async (url, mode = 'analyze') => {
    await sendMessage(url, [], { mode, llmConfigName: selectedModel });
  };

  // 发送对话消息（从底部输入）
  const handleChat = async (text) => {
    await sendMessage(text, [], { llmConfigName: selectedModel });
  };

  // 开始新对话
  const handleNewConversation = () => {
    createConversation();  // 清空 currentId
    clearMessages();       // 清空消息
    navigateToHome();      // 导航到首页
    setSelectedModel(defaultModel);  // 重置为默认模型
  };

  // 选择历史对话
  const handleSelectConversation = (id) => {
    selectConversation(id);
    navigateToSession(id); // 导航到会话 URL
    // 模型选择由 useEffect 监听 currentId 变化自动同步
  };

  // 判断是否有内容（用于显示空状态）
  const hasContent = messages.length > 0 || streamingMessage;

  // --- Render ---

  return (
    <div className="flex h-screen w-full overflow-hidden bg-bg-primary text-text-primary selection:bg-primary/20">

      {/* Left Sidebar - Fixed 300px */}
      <Sidebar
        isOpen={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
        conversations={conversations}
        currentId={currentId}
        onSelect={handleSelectConversation}
        onNew={handleNewConversation}
        onDelete={deleteConversation}
      />

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col h-full bg-bg-primary relative overflow-hidden">
        {/* Header - Simplified */}
        <header className="z-20 bg-bg-primary/80 backdrop-blur-md border-b border-border-default px-6 py-3">
          <div className="max-w-4xl mx-auto flex items-center justify-between">
            {/* Logo area or empty space */}
            <div className="flex-1">
              {/* 可以在此添加 Logo 或其他导航元素 */}
            </div>

            {/* Right side toggles */}
            <div className="flex items-center gap-1.5 ml-auto">
              <ThemeSwitcher />
              <LanguageSwitcher />
            </div>
          </div>
        </header>

        {/* Mobile menu button */}
        <button
          onClick={() => setSidebarOpen(true)}
          className="lg:hidden fixed top-4 left-4 z-50 p-2 bg-bg-secondary border border-border-default rounded-lg text-text-secondary hover:text-text-primary transition-colors"
        >
          <span className="material-symbols-outlined">menu</span>
        </button>

        {/* Scrollable Content Area */}
        <div ref={scrollContainerRef} className="flex-1 overflow-y-auto custom-scrollbar pb-48 pt-8">
          <div className="max-w-4xl mx-auto w-full px-8">
            <ChatContainer
              ref={chatContainerRef}
              scrollContainerRef={scrollContainerRef}
              messages={messages}
              streamingMessage={streamingMessage}
              loading={chatLoading}
              showEmptyState={!hasContent}
              onRetry={retryFromIndex}
            />
          </div>
        </div>

        {/* Bottom Footer with Chat Input - 始终显示 */}
        <ChatInputFooter
          onSend={handleChat}
          onAnalyze={handleAnalyze}
          onStop={stopGeneration}
          loading={chatLoading}
          sessionId={currentId}
          selectedModel={selectedModel}
          onSelectModel={setSelectedModel}
        />
      </main>
    </div>
  );
}

/**
 * HomeRoute - 首页路由
 *
 * 根据登录状态显示不同内容：
 * - 已登录：显示聊天界面 (AppContent)
 * - 未登录：显示落地页 (LandingPage)
 */
function HomeRoute() {
  const { isAuthenticated, loading } = useAuth();

  // 加载中显示加载指示器
  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-bg-primary">
        <div className="animate-spin rounded-full h-8 w-8 border-2 border-primary border-t-transparent" />
      </div>
    );
  }

  // 根据登录状态显示不同内容
  return isAuthenticated ? <AppContent /> : <LandingPage />;
}

function App() {
  return (
    <ThemeProvider>
    <AuthProvider>
      <Routes>
        {/* 首页：根据登录状态显示落地页或聊天界面 */}
        <Route path="/" element={<HomeRoute />} />

        {/* 会话页面：需要登录 */}
        <Route
          path="/session/:sessionId"
          element={
            <ProtectedRoute>
              <AppContent />
            </ProtectedRoute>
          }
        />

        {/* 认证路由（仅未登录用户可访问） */}
        <Route
          path="/auth/login"
          element={
            <PublicOnlyRoute>
              <LoginPage />
            </PublicOnlyRoute>
          }
        />
        <Route
          path="/auth/register"
          element={
            <PublicOnlyRoute>
              <RegisterPage />
            </PublicOnlyRoute>
          }
        />
        <Route path="/auth/callback/:provider" element={<OAuthCallback />} />
        <Route path="/auth/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/auth/reset-password" element={<ResetPasswordPage />} />
        <Route path="/auth/verify-email" element={<VerifyEmailPage />} />
      </Routes>
    </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
