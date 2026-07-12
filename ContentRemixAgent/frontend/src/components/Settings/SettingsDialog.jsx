/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/Settings/SettingsDialog.jsx
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

import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { createPortal } from 'react-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Cookie, Bot, Settings as SettingsIcon, Palette } from 'lucide-react';
import CookiesManager from './CookiesManager';
import LLMConfigManager from './LLMConfigManager';
import InsightModeManager from './InsightModeManager';

/**
 * 设置菜单配置
 * 可扩展添加更多设置项
 */
const getMenuItems = (t) => [
  {
    key: 'llm',
    icon: Bot,
    label: t('llmConfig.title'),
  },
  {
    key: 'modes',
    icon: Palette,
    label: t('insightModes.title'),
  },
  {
    key: 'cookies',
    icon: Cookie,
    label: t('settings.cookies'),
  },
  // 未来可添加更多设置项
  // { key: 'general', icon: SettingsIcon, label: t('settings.general') },
];

/**
 * SettingsDialog - 设置弹窗
 *
 * 左侧菜单 + 右侧内容区域的设置界面
 * 可扩展添加更多设置项
 */
function SettingsDialog({ isOpen, onClose }) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('llm');

  const menuItems = getMenuItems(t);

  // ESC 关闭
  React.useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = '';
    };
  }, [isOpen, onClose]);

  // 点击遮罩关闭
  const handleBackdropClick = (e) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  // 渲染右侧内容
  const renderContent = () => {
    switch (activeTab) {
      case 'llm':
        return <LLMConfigManager />;
      case 'modes':
        return <InsightModeManager />;
      case 'cookies':
        return <CookiesManager />;
      default:
        return null;
    }
  };

  const modalContent = (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2 }}
          onClick={handleBackdropClick}
        >
          <motion.div
            className="w-full max-w-6xl h-[85vh] max-h-[900px] bg-bg-secondary border border-border-default rounded-2xl shadow-2xl overflow-hidden flex flex-col mx-4"
            initial={{ opacity: 0, scale: 0.95, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 20 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
          >
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-4 border-b border-border-default">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
                  <SettingsIcon size={18} className="text-primary" />
                </div>
                <h2 className="text-lg font-semibold text-text-primary">{t('settings.title')}</h2>
              </div>
              <button
                onClick={onClose}
                className="p-2 text-text-muted hover:text-text-primary hover:bg-white/5 rounded-lg transition-colors"
                aria-label="Close"
              >
                <X size={20} />
              </button>
            </div>

            {/* Body: Left Menu + Right Content */}
            <div className="flex-1 flex overflow-hidden">
              {/* Left Menu */}
              <nav className="w-56 border-r border-border-default p-4 flex-shrink-0">
                <div className="space-y-1">
                  {menuItems.map((item) => {
                    const Icon = item.icon;
                    const isActive = activeTab === item.key;
                    return (
                      <button
                        key={item.key}
                        onClick={() => setActiveTab(item.key)}
                        className={`
                          w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left text-sm font-medium transition-all
                          ${isActive
                            ? 'bg-primary/10 text-primary border border-primary/20'
                            : 'text-text-secondary hover:text-text-primary hover:bg-white/5 border border-transparent'
                          }
                        `}
                      >
                        <Icon size={18} />
                        {item.label}
                      </button>
                    );
                  })}
                </div>
              </nav>

              {/* Right Content */}
              <div className="flex-1 p-6 overflow-y-auto custom-scrollbar">
                {renderContent()}
              </div>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );

  return createPortal(modalContent, document.body);
}

export default SettingsDialog;
