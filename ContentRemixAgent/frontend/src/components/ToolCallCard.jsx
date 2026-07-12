/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ToolCallCard.jsx
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
import CheckCircle from 'lucide-react/dist/esm/icons/check-circle';
import Loader2 from 'lucide-react/dist/esm/icons/loader-2';
import Circle from 'lucide-react/dist/esm/icons/circle';
import { getToolIcon } from '../constants/tools';
import { useTranslation } from 'react-i18next';

/**
 * ToolCallCard - 深色主题工具调用卡片
 *
 * ViralAI 风格的工具执行状态卡片
 * 青色进行中 + 绿色完成
 *
 * ┌─────────────────────────────┐
 * │ ✓ Completed video analysis  │
 * └─────────────────────────────┘
 *
 * ┌─────────────────────────────┐
 * │ ⟳ Parsing link...          │
 * │   Detecting platform...     │
 * └─────────────────────────────┘
 */
function ToolCallCard({ segment }) {
  const { t } = useTranslation();
  const isRunning = segment.status === 'running';
  const isCompleted = segment.status === 'completed';

  const IconComponent = getToolIcon(segment.tool);
  const label = segment.toolLabel || segment.tool;

  return (
    <div className={`
      my-3 px-4 py-3 rounded-xl border
      transition-all duration-300 animate-fade-in
      ${isRunning
        ? 'bg-accent-cyan/5 border-accent-cyan/30 shadow-[0_0_15px_rgba(6,182,212,0.1)]'
        : 'bg-bg-secondary border-border-default'
      }
    `}>
      <div className="flex items-center gap-3">
        {/* 状态图标 */}
        <div className={`
          w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0
          ${isRunning
            ? 'bg-accent-cyan/20'
            : isCompleted
              ? 'bg-accent-green/20'
              : 'bg-bg-tertiary'
          }
        `}>
          {isRunning ? (
            <Loader2 className="w-4 h-4 animate-spin text-accent-cyan" />
          ) : isCompleted ? (
            <CheckCircle className="w-4 h-4 text-accent-green" />
          ) : (
            <Circle className="w-4 h-4 text-text-muted" />
          )}
        </div>

        {/* 工具信息 */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className={`
              text-sm font-medium
              ${isRunning
                ? 'text-accent-cyan'
                : isCompleted
                  ? 'text-accent-green'
                  : 'text-text-secondary'
              }
            `}>
              {isCompleted ? `✓ ${label}` : label}
            </span>

            {isRunning && (
              <span className="text-xs text-accent-cyan/70 animate-pulse">
                {t('steps.processing')}...
              </span>
            )}
          </div>

          {/* 进度消息 */}
          {isRunning && (segment.progress || segment.message) && (
            <p className="text-xs text-text-muted mt-1 truncate">
              {segment.progress || segment.message}
            </p>
          )}
        </div>

        {/* 工具图标 */}
        <div className="flex-shrink-0">
          <IconComponent className={`
            w-4 h-4
            ${isRunning ? 'text-accent-cyan' : 'text-text-muted'}
          `} />
        </div>
      </div>
    </div>
  );
}

export default memo(ToolCallCard);
