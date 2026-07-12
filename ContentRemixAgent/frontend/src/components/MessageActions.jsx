/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/MessageActions.jsx
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

import React, { memo, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { copyToClipboard } from '../utils/copyToMarkdown';

/**
 * MessageActions - 消息底部操作栏
 *
 * 显示在 AI 消息底部的操作按钮（复制、重试）
 * 参考 ChatGPT/Claude 的 UI 风格
 *
 * @param {function} getMarkdown - 获取 Markdown 内容的函数
 * @param {function} onRetry - 重试回调（重新发送前一条用户消息）
 * @param {boolean} showRetry - 是否显示重试按钮
 */
function MessageActions({ getMarkdown, onRetry, showRetry = true, modelName }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async (e) => {
    e.preventDefault();
    e.stopPropagation();

    try {
      const content = typeof getMarkdown === 'function' ? getMarkdown() : getMarkdown;
      const success = await copyToClipboard(content);

      if (success) {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }
    } catch (err) {
      console.error('Copy failed:', err);
    }
  }, [getMarkdown]);

  const handleRetry = useCallback((e) => {
    e.preventDefault();
    e.stopPropagation();
    if (onRetry) {
      onRetry();
    }
  }, [onRetry]);

  return (
    <div className="message-actions">
      {/* 复制按钮 */}
      <button
        onClick={handleCopy}
        className={`message-action-btn ${copied ? 'copied' : ''}`}
        title={copied ? t('message.copied') : t('message.copyMarkdown')}
        aria-label={copied ? t('message.copied') : t('message.copyMarkdown')}
      >
        <span className="material-symbols-outlined">
          {copied ? 'check' : 'content_copy'}
        </span>
      </button>

      {/* 重试按钮 */}
      {showRetry && onRetry && (
        <button
          onClick={handleRetry}
          className="message-action-btn"
          title={t('message.retry')}
          aria-label={t('message.retry')}
        >
          <span className="material-symbols-outlined">refresh</span>
        </button>
      )}

      {/* 模型名称标签 */}
      {modelName && (
        <span className="ml-auto text-[11px] text-text-muted/60 select-none flex items-center gap-1">
          <span className="material-symbols-outlined text-[13px]">smart_toy</span>
          {modelName}
        </span>
      )}
    </div>
  );
}

export default memo(MessageActions);
