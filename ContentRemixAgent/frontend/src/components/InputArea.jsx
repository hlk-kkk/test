/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/InputArea.jsx
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

import React, { useState, useRef, useEffect, memo } from 'react';
import { useTranslation } from 'react-i18next';
import Send from 'lucide-react/dist/esm/icons/send';
import X from 'lucide-react/dist/esm/icons/x';
import Sparkles from 'lucide-react/dist/esm/icons/sparkles';

/**
 * InputArea - 深色主题输入区域
 *
 * ViralAI 风格的底部输入组件
 * 青色发光边框 + 深色背景
 */
function InputArea({
  onSend,
  loading,
  disabled,
  prefillAction,    // { id, prefill, placeholder }
  onClearPrefill    // 清除预填回调
}) {
  const { t } = useTranslation();
  const [input, setInput] = useState('');
  const [isFocused, setIsFocused] = useState(false);
  const textareaRef = useRef(null);

  // 当 prefillAction 变化时，设置预填内容
  useEffect(() => {
    if (prefillAction) {
      const fullText = prefillAction.prefill + prefillAction.placeholder;
      setInput(fullText);

      // 聚焦并选中占位符部分
      if (textareaRef.current) {
        textareaRef.current.focus();
        const start = prefillAction.prefill.length;
        const end = fullText.length;

        const timeoutId = setTimeout(() => {
          if (textareaRef.current) {
            textareaRef.current.setSelectionRange(start, end);
          }
        }, 0);

        return () => clearTimeout(timeoutId);
      }
    }
  }, [prefillAction]);

  const handleSend = () => {
    if (!input.trim() || disabled) return;

    // 检查是否还包含未替换的占位符
    if (prefillAction && input.includes(prefillAction.placeholder)) {
      textareaRef.current?.focus();
      const start = input.indexOf(prefillAction.placeholder);
      const end = start + prefillAction.placeholder.length;
      textareaRef.current?.setSelectionRange(start, end);
      return;
    }

    onSend(input);
    setInput('');
    onClearPrefill?.();
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleInput = (e) => {
    setInput(e.target.value);
    e.target.style.height = 'auto';
    e.target.style.height = Math.min(e.target.scrollHeight, 160) + 'px';
  };

  // 处理粘贴事件 - 自动替换占位符
  const handlePaste = (e) => {
    if (prefillAction && input.includes(prefillAction.placeholder)) {
      e.preventDefault();
      const pastedText = e.clipboardData.getData('text');
      const newInput = input.replace(prefillAction.placeholder, pastedText);
      setInput(newInput);

      setTimeout(() => {
        if (textareaRef.current) {
          const cursorPos = prefillAction.prefill.length + pastedText.length;
          textareaRef.current.setSelectionRange(cursorPos, cursorPos);
        }
      }, 0);
    }
  };

  const handleClear = () => {
    setInput('');
    onClearPrefill?.();
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.focus();
    }
  };

  const showClearButton = prefillAction && input.length > 0;
  const hasInput = input.trim().length > 0;

  return (
    <div className="flex-none border-t border-border-default bg-bg-primary/95 backdrop-blur-sm p-4">
      <div className="w-full max-w-4xl mx-auto">
        {/* 输入框容器 */}
        <div
          className={`
            relative flex items-end gap-3
            bg-bg-secondary rounded-2xl
            border transition-all duration-300
            ${isFocused
              ? 'border-accent-cyan/50 shadow-[0_0_20px_rgba(6,182,212,0.15)]'
              : 'border-border-default hover:border-border-default/80'
            }
            px-4 py-3
          `}
        >
          {/* AI 图标 */}
          <div className={`
            p-2 rounded-lg transition-colors flex-shrink-0
            ${isFocused ? 'text-accent-cyan' : 'text-text-muted'}
          `}>
            <Sparkles size={20} aria-hidden="true" />
          </div>

          {/* 输入框 */}
          <textarea
            ref={textareaRef}
            rows={1}
            value={input}
            onChange={handleInput}
            onKeyDown={handleKeyDown}
            onPaste={handlePaste}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            placeholder={t('inputArea.placeholder')}
            className="
              flex-1 bg-transparent border-none
              focus:ring-0 focus:outline-none
              py-2 max-h-40 resize-none
              text-text-primary placeholder:text-text-muted
              text-sm leading-relaxed
            "
            disabled={disabled}
          />

          {/* 清除按钮 */}
          {showClearButton && (
            <button
              type="button"
              onClick={handleClear}
              aria-label={t('inputArea.clear')}
              className="
                p-2 rounded-lg
                text-text-muted hover:text-text-secondary
                hover:bg-bg-tertiary
                transition-colors flex-shrink-0
              "
            >
              <X size={16} aria-hidden="true" />
            </button>
          )}

          {/* 发送按钮 */}
          <button
            type="button"
            onClick={handleSend}
            disabled={disabled || !hasInput}
            aria-label={t('inputArea.send')}
            className={`
              p-2.5 rounded-xl transition-all duration-200 flex-shrink-0
              ${hasInput && !disabled
                ? 'bg-gradient-to-r from-accent-cyan to-accent-cyan/80 text-white shadow-lg shadow-accent-cyan/25 hover:shadow-accent-cyan/40 hover:scale-105'
                : 'bg-bg-tertiary text-text-muted cursor-not-allowed'
              }
            `}
          >
            <Send size={18} aria-hidden="true" />
          </button>
        </div>

        {/* 快捷键提示 */}
        <div className="flex items-center justify-center gap-4 mt-3">
          <span className="text-[11px] text-text-muted">
            <kbd className="px-1.5 py-0.5 rounded bg-bg-tertiary text-text-secondary font-mono text-[10px]">Enter</kbd> {t('inputArea.enterHint', { defaultValue: 'to send' })}
          </span>
          <span className="text-[11px] text-text-muted">
            <kbd className="px-1.5 py-0.5 rounded bg-bg-tertiary text-text-secondary font-mono text-[10px]">Shift + Enter</kbd> {t('inputArea.shiftEnterHint', { defaultValue: 'for new line' })}
          </span>
        </div>
      </div>
    </div>
  );
}

export default memo(InputArea);
