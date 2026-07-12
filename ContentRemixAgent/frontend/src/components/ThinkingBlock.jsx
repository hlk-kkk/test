/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ThinkingBlock.jsx
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

import React, { useState, memo } from 'react';
import Brain from 'lucide-react/dist/esm/icons/brain';
import Loader2 from 'lucide-react/dist/esm/icons/loader-2';
import ChevronDown from 'lucide-react/dist/esm/icons/chevron-down';

/**
 * ThinkingBlock - 深色主题可折叠思考块
 *
 * ViralAI 风格的 AI 思考过程展示
 * 紫色渐变 + 深色背景
 *
 * ┌─────────────────────────────┐
 * │ 🧠 Thinking 1.13s ▼         │
 * │   (点击展开查看思考内容)     │
 * └─────────────────────────────┘
 */
function ThinkingBlock({ segment }) {
  const [isExpanded, setIsExpanded] = useState(false);

  const isRunning = segment.status === 'running';
  const isCompleted = segment.status === 'completed';
  const duration = segment.duration;
  const hasContent = segment.content && segment.content.trim().length > 0;

  // 如果已完成但没有思考内容，不显示
  if (isCompleted && !hasContent) {
    return null;
  }

  const formatDuration = (seconds) => {
    if (!seconds) return '';
    if (seconds < 1) return `${Math.round(seconds * 1000)}ms`;
    return `${seconds.toFixed(2)}s`;
  };

  return (
    <div className="my-3 animate-fade-in">
      {/* 折叠按钮 */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className={`
          flex items-center gap-2.5 px-4 py-2.5 rounded-xl text-sm
          transition-all duration-200
          border
          ${isRunning
            ? 'bg-accent-purple/10 border-accent-purple/30 text-accent-purple'
            : 'bg-bg-secondary border-border-default text-text-secondary hover:bg-bg-tertiary'
          }
        `}
      >
        {/* 图标 */}
        <div className={`
          w-6 h-6 rounded-lg flex items-center justify-center
          ${isRunning ? 'bg-accent-purple/20' : 'bg-bg-tertiary'}
        `}>
          {isRunning ? (
            <Loader2 className="w-3.5 h-3.5 animate-spin text-accent-purple" />
          ) : (
            <Brain className="w-3.5 h-3.5 text-text-muted" />
          )}
        </div>

        {/* 状态文字 */}
        <span className="font-medium">
          {isRunning ? 'Thinking...' : 'Thought'}
        </span>

        {/* 时长 */}
        {isCompleted && duration && (
          <span className="text-text-muted text-xs font-mono">
            {formatDuration(duration)}
          </span>
        )}

        {/* 展开/折叠箭头 */}
        <ChevronDown
          className={`
            w-4 h-4 text-text-muted transition-transform duration-200
            ${isExpanded ? 'rotate-180' : ''}
          `}
        />
      </button>

      {/* 思考内容（展开时显示） */}
      {isExpanded && segment.content && (
        <div className="mt-3 ml-4 animate-fade-in">
          <div className="
            bg-accent-purple/5 border border-accent-purple/20 rounded-xl p-4
            text-sm text-text-secondary leading-relaxed
            whitespace-pre-wrap font-mono
          ">
            {segment.content}
          </div>
        </div>
      )}
    </div>
  );
}

export default memo(ThinkingBlock);
