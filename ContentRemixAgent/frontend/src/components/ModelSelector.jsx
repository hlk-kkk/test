/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ModelSelector.jsx
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
import { fetchLLMConfigList } from '../services/api';

/**
 * ModelSelector - LLM 模型选择器组件
 *
 * 可复用的下拉模型选择器，支持：
 * - 从后端加载 LLM 配置列表
 * - 浮层向上展开（适用于底部输入框）
 * - 标记全局激活配置
 * - 点击外部关闭
 */
function ModelSelector({
  selectedConfigName,
  onSelect,
  disabled = false,
  className = '',
}) {
  const { t } = useTranslation();
  const dropdownRef = useRef(null);
  const [isOpen, setIsOpen] = useState(false);
  const [configs, setConfigs] = useState([]);
  const [activeConfig, setActiveConfig] = useState(null);

  // 加载 LLM 配置列表
  useEffect(() => {
    let mounted = true;

    const loadConfigs = async () => {
      try {
        const data = await fetchLLMConfigList();
        if (mounted && data?.items) {
          setConfigs(data.items);
          const active = data.items.find(c => c.is_active);
          setActiveConfig(active?.config_name || null);
        }
      } catch (err) {
        console.warn('Failed to load LLM configs:', err);
      }
    };

    loadConfigs();
    return () => { mounted = false; };
  }, []);

  // 打开下拉框时刷新数据
  useEffect(() => {
    if (!isOpen) return;

    const refreshConfigs = async () => {
      try {
        const data = await fetchLLMConfigList();
        if (data?.items) {
          setConfigs(data.items);
          const active = data.items.find(c => c.is_active);
          setActiveConfig(active?.config_name || null);
        }
      } catch (err) {
        // 忽略错误，保持当前数据
      }
    };
    refreshConfigs();
  }, [isOpen]);

  // 点击外部关闭
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  // 获取当前选中的配置信息
  const selectedConfig = configs.find(c => c.config_name === selectedConfigName);
  const displayName = selectedConfig?.model_name || selectedConfigName || t('modelSelector.noModel', { defaultValue: '选择模型' });

  // 没有配置时的处理
  if (configs.length === 0) {
    return (
      <div className={`flex items-center ${className}`}>
        <button
          type="button"
          disabled
          className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-text-muted cursor-not-allowed"
          title={t('modelSelector.noConfigs', { defaultValue: '未配置模型' })}
        >
          <span className="material-symbols-outlined text-xl">smart_toy</span>
        </button>
      </div>
    );
  }

  return (
    <div ref={dropdownRef} className={`relative flex items-center ${className}`}>
      {/* 模型选择器按钮 */}
      <button
        type="button"
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        className={`
          flex items-center gap-1.5 px-3 py-2 rounded-lg
          border border-transparent
          hover:bg-white/5 hover:border-border-subtle
          transition-all duration-200
          text-accent-cyan
          ${disabled ? 'opacity-50 cursor-not-allowed' : ''}
        `}
        title={displayName}
      >
        <span className="material-symbols-outlined text-xl">smart_toy</span>
        <span className="text-xs text-text-secondary max-w-[100px] truncate hidden sm:inline">
          {displayName}
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
        <div className="
          absolute bottom-full left-0 mb-2 w-64
          bg-bg-secondary border border-border-default rounded-xl
          shadow-xl shadow-black/30
          overflow-hidden z-50
          animate-fade-in
        ">
          {configs.map((config) => {
            const isSelected = selectedConfigName === config.config_name;
            const isActive = config.config_name === activeConfig;
            return (
              <button
                key={config.config_name}
                type="button"
                onClick={() => {
                  onSelect(config.config_name);
                  setIsOpen(false);
                }}
                className={`
                  w-full flex items-center gap-3 px-4 py-3 text-left
                  transition-all duration-150
                  ${isSelected ? 'bg-white/10' : 'hover:bg-white/5'}
                `}
              >
                <span className="material-symbols-outlined text-xl text-accent-cyan">
                  smart_toy
                </span>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-1.5">
                    <span className="text-sm font-medium text-text-primary truncate">
                      {config.model_name}
                    </span>
                    {isActive && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-accent-cyan/20 text-accent-cyan whitespace-nowrap">
                        {t('modelSelector.default', { defaultValue: '默认' })}
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-text-muted truncate">
                    {config.config_name} · {config.provider}
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

export default ModelSelector;
