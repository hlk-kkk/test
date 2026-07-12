/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ActionButtons.jsx
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
import Edit from 'lucide-react/dist/esm/icons/edit';
import Film from 'lucide-react/dist/esm/icons/film';
import AtSign from 'lucide-react/dist/esm/icons/at-sign';

/**
 * ActionButtons - 底部快捷操作按钮
 *
 * 提供快速操作入口
 *
 * Props:
 * - onAction: (actionId) => void - 点击按钮的回调
 * - disabled: boolean - 是否禁用按钮
 */
function ActionButtons({ onAction, disabled = false }) {
  const actions = [
    {
      id: 'remix',
      icon: Edit,
      label: 'Remix into Script',
      description: '将内容改写成脚本',
    },
    {
      id: 'storyboard',
      icon: Film,
      label: 'Generate Storyboard',
      description: '生成故事板',
    },
    {
      id: 'collaborator',
      icon: AtSign,
      label: 'Mention Collaborator',
      description: '提及协作者',
    },
  ];

  const handleClick = (actionId) => {
    if (!disabled && onAction) {
      onAction(actionId);
    }
  };

  return (
    <div className="flex flex-wrap gap-2 justify-center mb-4">
      {actions.map((action) => {
        const Icon = action.icon;
        return (
          <button
            key={action.id}
            onClick={() => handleClick(action.id)}
            disabled={disabled}
            className={`
              inline-flex items-center gap-2 px-4 py-2
              bg-white border border-bg-300 rounded-full
              text-xs font-medium text-text-200
              transition-all duration-200
              hover:border-primary-100 hover:text-primary-100
              disabled:opacity-50 disabled:cursor-not-allowed
              dark:bg-slate-800 dark:border-slate-700 dark:text-slate-400
              dark:hover:border-primary-100 dark:hover:text-primary-100
            `}
            title={action.description}
          >
            <Icon className="w-4 h-4" />
            <span>{action.label}</span>
          </button>
        );
      })}
    </div>
  );
}

export default memo(ActionButtons);
