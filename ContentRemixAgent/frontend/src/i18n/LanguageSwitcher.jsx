/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/i18n/LanguageSwitcher.jsx
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

import React from 'react';
import { useTranslation } from 'react-i18next';
import { getCurrentLanguage, toggleLanguage } from './index';

/**
 * LanguageSwitcher - Compact language toggle button
 *
 * Toggles between English and Chinese with a single click.
 * Shows current language indicator (EN / 中).
 */
function LanguageSwitcher({ className = '' }) {
  const { i18n } = useTranslation();
  const currentLang = getCurrentLanguage();
  const isEnglish = currentLang === 'en';

  const handleToggle = () => {
    toggleLanguage();
  };

  return (
    <button
      type="button"
      onClick={handleToggle}
      className={`
        p-2 rounded-lg transition-colors duration-200
        text-text-secondary hover:text-text-primary hover:bg-bg-tertiary
        font-medium text-sm
        ${className}
      `}
      title={isEnglish ? '点击切换中文' : 'Switch to English'}
      aria-label={isEnglish ? 'Switch to Chinese' : '切换到英文'}
    >
      {/*
        这里可以只显示图标，或者显示当前语言的简称。
        为了极简，我们显示"对方"语言的简称，引导用户点击切换？
        或者显示一个通用的翻译图标。

        方案：显示 standard translate icon + 当前语言标识 (EN/中)
        或者更极简： EN / 中 text only.
      */}
      <div className="flex items-center gap-1">
        <span className="material-symbols-outlined text-xl">translate</span>
        {/* <span className="text-xs font-bold uppercase">{isEnglish ? 'EN' : '中'}</span> */}
      </div>
    </button>
  );
}

export default LanguageSwitcher;
