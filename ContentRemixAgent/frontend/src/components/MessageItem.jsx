/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/MessageItem.jsx
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

import React, { memo, useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import StepsCollapsible from './StepsCollapsible';
import MarkdownBlock from './MarkdownBlock';
import ContentCard from './ContentCard';
import TranscriptCard from './TranscriptCard';
import MessageActions from './MessageActions';
import {
  userMessageToMarkdown,
  aiMessageToMarkdown,
  copyToClipboard,
} from '../utils/copyToMarkdown';

/**
 * MessageItem - 消息项组件
 *
 * 深色主题风格的消息渲染
 * 用户消息：青色气泡，右对齐
 * AI消息：左对齐，包含头像、步骤列表、内容卡片、底部操作栏
 */
export function MessageItem({ message, onRetry }) {
  const isUser = message.role === 'user';
  const isStreaming = message.isComplete === false;

  // 使用 segments 数组
  const segments = message.segments || [];

  // 提取结构化数据
  const structuredData = message.structuredData;

  if (isUser) {
    return <UserMessage content={message.content} />;
  }

  return (
    <AIMessage
      segments={segments}
      structuredData={structuredData}
      isStreaming={isStreaming}
      onRetry={onRetry}
    />
  );
}

/**
 * 用户消息组件
 *
 * 直接显示用户原始输入内容，不做任何处理
 * hover 时显示复制按钮
 */
const UserMessage = memo(function UserMessage({ content }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async (e) => {
    e.preventDefault();
    e.stopPropagation();
    try {
      const success = await copyToClipboard(content);
      if (success) {
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
      }
    } catch (err) {
      console.error('Copy failed:', err);
    }
  }, [content]);

  return (
    <div className="flex flex-col items-end mb-8 animate-fade-in group">
      {/* 消息气泡 */}
      <div className="max-w-[85%] bg-gradient-to-br from-primary to-primary-active text-white px-6 py-4 rounded-2xl rounded-tr-none shadow-[0_4px_20px_rgba(255,59,59,0.25)]">
        <div className="text-sm leading-relaxed whitespace-pre-wrap break-all">
          {content}
        </div>
      </div>
      {/* 复制按钮 - 右下角，hover 时显示 */}
      <button
        onClick={handleCopy}
        className={`user-message-copy-btn ${copied ? 'copied' : ''}`}
        title={copied ? '已复制' : '复制'}
      >
        <span className="material-symbols-outlined">
          {copied ? 'check' : 'content_copy'}
        </span>
      </button>
    </div>
  );
});

/**
 * AI 消息组件
 *
 * 完成后在底部显示操作栏（复制、重试）
 */
const AIMessage = memo(function AIMessage({ segments, structuredData, isStreaming, onRetry }) {
  const { t } = useTranslation();

  // 简化：直接根据 segment type 分类，不再需要复杂的顺序判断
  // 后端通过 final_report_start 事件已正确标记了 process_text 和 markdown
  const stepSegments = segments.filter(
    s => s.type === 'intent' ||
         s.type === 'tool_call' ||
         s.type === 'thinking' ||
         s.type === 'sub_step' ||
         s.type === 'process_text'  // 后端已正确标记
  );

  const contentSegments = segments.filter(
    s =>
      (s.type === 'text' || s.type === 'markdown') &&
      typeof s.content === 'string' &&
      s.content.trim().length > 0 // 过滤空白内容，避免渲染空卡片
  );

  // 获取 content_info segment
  const contentInfoSegment = segments.find(s => s.type === 'content_info');
  const contentInfo = contentInfoSegment?.data;

  // 获取 transcript（从多个来源尝试获取）
  // 后端返回 TranscriptResult: { text: string, segments: [{start, end, text}, ...] }
  // 优先级: transcript segment > contentInfo > structuredData (done 事件)
  const transcriptSegment = segments.find(s => s.type === 'transcript');
  const transcriptData =
    transcriptSegment?.data ||
    transcriptSegment?.content ||
    contentInfo?.transcript ||
    structuredData?.transcript;  // 从 done 事件的 structuredData 获取

  // 提取 segments 数组（如果存在）
  const transcriptSegments = transcriptData?.segments || null;
  const transcript = transcriptData?.text || (typeof transcriptData === 'string' ? transcriptData : null);

  // 检查是否有内容显示
  const hasSteps = stepSegments.length > 0;
  const hasContent = contentSegments.length > 0 || contentInfo;
  const hasAnyContent = hasSteps || hasContent;

  // 是否显示报告头部（第一个 markdown segment）
  const showReportHeader = contentSegments.length > 0 && contentSegments[0].type === 'markdown';

  // 复制为 Markdown 的回调
  const getMarkdown = useCallback(
    () => aiMessageToMarkdown(segments, structuredData),
    [segments, structuredData]
  );

  return (
    <div className="flex flex-col gap-8 animate-fade-in">
      {/* 步骤列表（可折叠） - Analysis Complete 卡片 */}
      {hasSteps && (
        <StepsCollapsible steps={stepSegments} isStreaming={isStreaming} />
      )}

      {/* 内容卡片 - ContentCard（根据类型展示视频或图文） */}
      {contentInfo && (
        <ContentCard contentInfo={contentInfo} />
      )}

      {/* 转录卡片 - TranscriptCard */}
      {(transcript || transcriptSegments) && (
        <TranscriptCard
          transcript={transcript}
          segments={transcriptSegments}
        />
      )}

      {/* Markdown/Text 内容 - AI Report */}
      {contentSegments.map((segment, index) => {
        if (segment.type === 'markdown') {
          return (
            <MarkdownBlock
              key={`md-${index}`}
              segment={segment}
              isStreaming={index === contentSegments.length - 1 && isStreaming}
              showReportHeader={index === 0 && showReportHeader}
            />
          );
        }
        if (segment.type === 'text' && segment.content) {
          return (
            <div
              key={`text-${index}`}
              className="text-sm text-text-secondary leading-relaxed whitespace-pre-wrap"
            >
              {segment.content}
            </div>
          );
        }
        return null;
      })}

      {/* 如果没有任何内容且正在流式输出，显示加载指示器 */}
      {!hasAnyContent && isStreaming && (
        <div className="flex items-center justify-center gap-3 py-8">
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full bg-primary animate-bounce" />
            <div className="w-2 h-2 rounded-full bg-primary animate-bounce" style={{ animationDelay: '0.1s' }} />
            <div className="w-2 h-2 rounded-full bg-primary animate-bounce" style={{ animationDelay: '0.2s' }} />
          </div>
          <span className="text-sm text-text-muted">{t('emptyState.analyzing')}</span>
        </div>
      )}

      {/* 底部操作栏 - 仅在完成后显示 */}
      {!isStreaming && hasAnyContent && (
        <MessageActions
          getMarkdown={getMarkdown}
          onRetry={onRetry}
          showRetry={!!onRetry}
          modelName={structuredData?.model_name}
        />
      )}
    </div>
  );
});

export default memo(MessageItem);
