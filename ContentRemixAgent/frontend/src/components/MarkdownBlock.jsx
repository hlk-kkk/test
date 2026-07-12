/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/MarkdownBlock.jsx
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

import React, { memo } from 'react';
import StreamingMarkdown from './StreamingMarkdown';

/**
 * MarkdownBlock - AI Report 样式的 Markdown 内容块组件
 *
 * Remix AI Studio 风格的分析报告展示
 * 包含 glass-effect 卡片和标题分割线
 */
function MarkdownBlock({ segment, isStreaming = false, showReportHeader = false }) {
  if (!segment.content || !segment.content.trim()) {
    return null;
  }

  return (
    <section className="markdown-container glass-effect p-10 rounded-3xl border border-border-default shadow-2xl animate-fade-in">
      {/* Report header divider */}
      {showReportHeader && (
        <div className="report-section-divider mb-8">
          <div className="line" />
          <h2 className="title">AI Insight Engine Report</h2>
          <div className="line" />
        </div>
      )}

      {/* Markdown content */}
      <StreamingMarkdown
        content={segment.content}
        isStreaming={isStreaming}
      />
    </section>
  );
}

export default memo(MarkdownBlock);
