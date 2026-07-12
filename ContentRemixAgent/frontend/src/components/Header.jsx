/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/Header.jsx
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
import Zap from 'lucide-react/dist/esm/icons/zap';

/**
 * Header - 顶部导航栏
 *
 * ViralAI 风格的深色主题导航栏
 * 包含 Logo、平台标识、标题和操作按钮
 */
function Header({ platform }) {
  return (
    <header className="flex items-center justify-between px-6 py-4 border-b border-border-default bg-bg-primary">
      {/* 左侧: Logo + 平台标签 */}
      <div className="flex items-center gap-4">
        {/* Logo */}
        <div className="flex items-center gap-2">
          <div className="w-9 h-9 rounded-xl bg-gradient-to-br from-accent-cyan to-accent-purple flex items-center justify-center">
            <Zap className="w-5 h-5 text-white" />
          </div>
          <span className="text-lg font-bold text-text-primary tracking-tight">
            Remix<span className="text-accent-cyan">Agent</span>
          </span>
        </div>

        {/* 平台标签 */}
        {platform && (
          <div className="badge badge-cyan">
            <PlatformIcon platform={platform} />
            <span className="ml-1.5">{getPlatformName(platform)}</span>
          </div>
        )}
      </div>

      {/* 中间: 页面标题 */}
      <div className="hidden md:flex items-center gap-2">
        <span className="text-sm text-text-muted">Post Case Study</span>
      </div>

    </header>
  );
}

/**
 * 平台图标组件
 */
function PlatformIcon({ platform }) {
  const iconMap = {
    xhs: '📕',
    dy: '🎵',
    douyin: '🎵',
    bili: '📺',
    bilibili: '📺',
    ks: '⚡',
    kuaishou: '⚡',
  };

  return <span className="text-sm">{iconMap[platform] || '🔗'}</span>;
}

/**
 * 获取平台名称
 */
function getPlatformName(platform) {
  const nameMap = {
    xhs: '小红书',
    dy: '抖音',
    douyin: '抖音',
    bili: 'B站',
    bilibili: 'B站',
    ks: '快手',
    kuaishou: '快手',
  };

  return nameMap[platform] || platform;
}

export default memo(Header);
