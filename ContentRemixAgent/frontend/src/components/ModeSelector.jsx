/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ModeSelector.jsx
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

import React, { useRef, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { INSIGHT_MODES, MODE_COLOR_CLASSES, fetchDynamicModes } from '../constants/modes';

/**
 * ModeSelector - 独立的模式选择器组件
 *
 * 可复用的下拉模式选择器，支持：
 * - 动态从后端加载启用的模式
 * - 浮层向上展开（适用于底部输入框）
 * - 点击外部关闭
 * - 自定义样式
 */
function ModeSelector({
  selectedMode,
  onSelectMode,
  isOpen,
  onToggle,
  className = ''
}) {
  const { t, i18n } = useTranslation();
  const dropdownRef = useRef(null);
  const [modes, setModes] = useState(INSIGHT_MODES); // 初始使用静态数据作为回退

  // 加载动态模式数据
  useEffect(() => {
    let mounted = true;

    const loadModes = async () => {
      try {
        const dynamicModes = await fetchDynamicModes();
        if (mounted && dynamicModes.length > 0) {
          setModes(dynamicModes);
        }
      } catch (err) {
        console.warn('Failed to load dynamic modes:', err);
        // 保持使用静态数据
      }
    };

    loadModes();

    return () => {
      mounted = false;
    };
  }, [i18n.language]); // 语言切换时重新加载

  // 打开下拉框时刷新数据（确保显示最新状态）
  useEffect(() => {
    if (isOpen) {
      const refreshModes = async () => {
        try {
          const dynamicModes = await fetchDynamicModes();
          if (dynamicModes.length > 0) {
            setModes(dynamicModes);
          }
        } catch (err) {
          // 忽略错误，保持当前数据
        }
      };
      refreshModes();
    }
  }, [isOpen]);

  // 点击外部关闭下拉框
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        onToggle();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen, onToggle]);

  const colorClasses = MODE_COLOR_CLASSES[selectedMode.color] || MODE_COLOR_CLASSES.cyan;

  // 获取模式的显示文本
  const getModeLabel = (mode) => {
    // 优先使用动态加载的 label（已包含翻译）
    if (mode.label) return mode.label;
    // 否则使用 i18n key
    return t(`${mode.i18nKey}.label`);
  };

  const getModeSubtitle = (mode) => {
    // 优先使用动态加载的 description
    if (mode.description) return mode.description;
    // 否则使用 i18n key
    return t(`${mode.i18nKey}.subtitle`);
  };

  return (
    <div ref={dropdownRef} className={`relative flex items-center ${className}`}>
      {/* 模式选择器按钮 */}
      <button
        type="button"
        onClick={onToggle}
        className={`
          flex items-center gap-1.5 px-3 py-2 rounded-lg
          border border-transparent
          hover:bg-white/5 hover:border-border-subtle
          transition-all duration-200
          ${colorClasses.text}
        `}
        title={getModeLabel(selectedMode)}
      >
        <span className="material-symbols-outlined text-xl">
          {selectedMode.icon}
        </span>
        <span className={`
          material-symbols-outlined text-sm transition-transform duration-200
          ${isOpen ? 'rotate-180' : ''}
        `}>
          expand_more
        </span>
      </button>

      {/* 下拉菜单 - 向上展开 */}
      {isOpen && (
        <div className="mode-selector-dropdown
          absolute bottom-full left-0 mb-2 w-56
          bg-bg-secondary border border-border-default rounded-xl
          shadow-xl shadow-black/30
          overflow-hidden z-50
          animate-fade-in
        ">
          {modes.map((mode) => {
            const modeColors = MODE_COLOR_CLASSES[mode.color] || MODE_COLOR_CLASSES.cyan;
            const isSelected = selectedMode.id === mode.id;
            return (
              <button
                key={mode.id}
                type="button"
                onClick={() => {
                  onSelectMode(mode);
                  onToggle();
                }}
                className={`
                  w-full flex items-center gap-3 px-4 py-3 text-left
                  transition-all duration-150
                  ${isSelected ? 'bg-white/10' : 'hover:bg-white/5'}
                `}
              >
                <span className={`material-symbols-outlined text-xl ${modeColors.text}`}>
                  {mode.icon}
                </span>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-text-primary">
                    {getModeLabel(mode)}
                  </div>
                  <div className="text-xs text-text-muted truncate">
                    {getModeSubtitle(mode)}
                  </div>
                </div>
                {isSelected && (
                  <span className="material-symbols-outlined text-sm text-accent-cyan">
                    check
                  </span>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default ModeSelector;
