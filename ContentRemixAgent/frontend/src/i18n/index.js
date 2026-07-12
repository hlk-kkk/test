/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/i18n/index.js
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

/**
 * i18n Configuration
 *
 * Multi-language support with English as default.
 * Supports: English (en), Chinese (zh)
 */
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import en from './locales/en.json';
import zh from './locales/zh.json';

const STORAGE_KEY = 'remix-studio-language';

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      zh: { translation: zh },
    },
    fallbackLng: 'en',
    debug: false,
    interpolation: {
      escapeValue: false, // React already escapes values
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: STORAGE_KEY,
      caches: ['localStorage'],
    },
  });

/**
 * Get current language
 * @returns {string} Current language code ('en' or 'zh')
 */
export function getCurrentLanguage() {
  return i18n.language?.startsWith('zh') ? 'zh' : 'en';
}

/**
 * Set language and persist to localStorage
 * @param {string} lang - Language code ('en' or 'zh')
 */
export function setLanguage(lang) {
  const normalizedLang = lang.startsWith('zh') ? 'zh' : 'en';
  i18n.changeLanguage(normalizedLang);
  localStorage.setItem(STORAGE_KEY, normalizedLang);

  // Dispatch custom event for components that need to react to language changes
  window.dispatchEvent(new CustomEvent('language-change', { detail: { language: normalizedLang } }));
}

/**
 * Toggle between English and Chinese
 */
export function toggleLanguage() {
  const current = getCurrentLanguage();
  setLanguage(current === 'en' ? 'zh' : 'en');
}

export default i18n;
