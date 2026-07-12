/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ChatInputFooter.jsx
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

import React, { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import ModeSelector from './ModeSelector';
import ModelSelector from './ModelSelector';
import { DEFAULT_MODE } from '../constants/modes';

/**
 * ChatInputFooter - 底部统一输入组件
 *
 * 支持：
 * 1. 自由对话（无需链接）
 * 2. 链接分析（自动检测 URL + 模式选择）
 * 3. 智能切换（有会话后隐藏模式选择器）
 * 4. 每会话独立模型选择
 */
function ChatInputFooter({ onSend, onAnalyze, onStop, loading, sessionId, selectedModel, onSelectModel }) {
  const { t } = useTranslation();
  const [inputValue, setInputValue] = useState('');
  const [selectedMode, setSelectedMode] = useState(DEFAULT_MODE);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [detectedUrl, setDetectedUrl] = useState(null);
  const textareaRef = useRef(null);

  // 自动调整 textarea 高度
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = 'auto';
      textarea.style.height = Math.min(textarea.scrollHeight, 120) + 'px';
    }
  }, [inputValue]);

  // 从输入内容中提取 URL
  const extractUrl = (text) => {
    const urlMatch = text.match(/https?:\/\/[^\s]+/);
    return urlMatch ? urlMatch[0] : null;
  };

  // 自动检测 URL
  useEffect(() => {
    const url = extractUrl(inputValue);
    setDetectedUrl(url);
  }, [inputValue]);

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!inputValue.trim() || loading) return;

    const url = extractUrl(inputValue);
    if (url && !sessionId) {
      // 有 URL 且无会话 → 调用分析
      onAnalyze(inputValue.trim(), selectedMode.id);
    } else {
      // 无 URL 或有会话 → 调用对话
      onSend(inputValue.trim());
    }
    setInputValue('');
  };

  const handleButtonClick = (e) => {
    e.preventDefault();

    if (loading) {
      // Loading 时点击 = 停止生成
      onStop?.();
    } else {
      // 否则发送消息
      if (!inputValue.trim()) return;

      const url = extractUrl(inputValue);
      if (url && !sessionId) {
        onAnalyze(inputValue.trim(), selectedMode.id);
      } else {
        onSend(inputValue.trim());
      }
      setInputValue('');
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const handleSelectMode = (mode) => {
    setSelectedMode(mode);
    // 预填充提示词（可选）
    const prefill = mode.prefill || t(`${mode.i18nKey}.prefill`, { defaultValue: '' });
    setInputValue(prefill ? `${prefill}：` : '');
    // 聚焦输入框并定位光标到末尾
    setTimeout(() => {
      if (textareaRef.current) {
        textareaRef.current.focus();
        const len = textareaRef.current.value.length;
        textareaRef.current.setSelectionRange(len, len);
      }
    }, 0);
  };

  // 判断是否显示模式选择器（无会话时显示）
  const showModeSelector = !sessionId;

  // 判断发送按钮图标
  const isAnalyzeMode = detectedUrl && !sessionId;

  // 动态 placeholder
  const placeholder = detectedUrl && !sessionId
    ? t('chatFooter.placeholderWithUrl', { defaultValue: '已检测到链接，选择分析模式后点击发送' })
    : t('chatFooter.placeholder', { defaultValue: '随时开始对话，或粘贴链接进行分析...' });

  return (
    <footer className="chat-input-footer">
      <div className="max-w-4xl mx-auto w-full px-8 flex flex-col gap-4">
        {/* Chat Input */}
        <div className="chat-input-wrapper">
          {/* Glow effect on focus */}
          <div className="glow" />

          <div className="chat-input-container">
            {/* 模式选择器（仅在无会话时显示）*/}
            {showModeSelector && (
              <ModeSelector
                selectedMode={selectedMode}
                onSelectMode={handleSelectMode}
                isOpen={isDropdownOpen}
                onToggle={() => setIsDropdownOpen(!isDropdownOpen)}
              />
            )}

            {/* 模型选择器（始终显示）*/}
            <ModelSelector
              selectedConfigName={selectedModel}
              onSelect={onSelectModel}
              disabled={loading}
            />

            {/* Input field */}
            <textarea
              ref={textareaRef}
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={placeholder}
              className="chat-input-field"
              disabled={loading}
              rows={1}
            />

            {/* Send/Stop button */}
            <button
              onClick={handleButtonClick}
              disabled={!loading && !inputValue.trim()}
              className={`chat-send-button ${isAnalyzeMode ? 'url-detected' : ''} ${loading ? 'stop-mode' : ''}`}
              title={loading ? t('chatFooter.stop') : (isAnalyzeMode ? t('chatFooter.startAnalysis') : t('chatFooter.send'))}
            >
              <span className="material-symbols-outlined">
                {loading ? 'stop_circle' : (isAnalyzeMode ? 'auto_awesome' : 'send')}
              </span>
            </button>
          </div>
        </div>

      </div>
    </footer>
  );
}

export default ChatInputFooter;
