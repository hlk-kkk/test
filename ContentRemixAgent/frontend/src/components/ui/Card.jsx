/**
 * Copyright (c) 2026 relakkes@gmail.com
 *
 * This file is part of MediaCrawlerPro-ContentRemixAgent project.
 * Repository: https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/blob/main/frontend/src/components/ui/Card.jsx
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
 * Card Component - Modern Minimal Design
 *
 * A unified card component for consistent styling across the app.
 *
 * Variants:
 * - default: Standard card with border
 * - elevated: Card with shadow that lifts on hover
 * - glass: Semi-transparent card with backdrop blur
 */
import React from 'react';

const variants = {
  default: `
    bg-bg-secondary
    border border-border-default
    hover:border-border-subtle
  `,
  elevated: `
    bg-bg-secondary
    border border-border-default
    shadow-lg shadow-black/5
    hover:shadow-xl hover:shadow-black/10
    hover:border-border-subtle
  `,
  glass: `
    bg-bg-secondary/80
    backdrop-blur-xl
    border border-border-default
    hover:border-border-subtle
  `,
  flat: `
    bg-bg-tertiary/50
    border border-transparent
    hover:bg-bg-tertiary
  `,
};

const paddings = {
  none: '',
  sm: 'p-4',
  md: 'p-6',
  lg: 'p-8',
};

export default function Card({
  children,
  variant = 'default',
  padding = 'md',
  rounded = 'xl',
  className = '',
  onClick,
  as: Component = 'div',
  ...props
}) {
  const baseStyles = `
    transition-all duration-200
    ${rounded === 'xl' ? 'rounded-xl' : rounded === 'lg' ? 'rounded-lg' : 'rounded-2xl'}
    ${onClick ? 'cursor-pointer' : ''}
  `;

  return (
    <Component
      onClick={onClick}
      className={`${baseStyles} ${variants[variant]} ${paddings[padding]} ${className}`}
      {...props}
    >
      {children}
    </Component>
  );
}

/**
 * CardHeader Component
 *
 * A header section for cards with title and optional actions.
 */
export function CardHeader({ children, className = '', actions }) {
  return (
    <div className={`flex items-center justify-between mb-4 ${className}`}>
      <div className="flex items-center gap-3">{children}</div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}

/**
 * CardTitle Component
 *
 * A title for cards with consistent styling.
 */
export function CardTitle({ children, icon, className = '' }) {
  return (
    <h3 className={`font-display font-semibold text-lg text-text-primary flex items-center gap-2 ${className}`}>
      {icon && (
        <span className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center">
          <span className="material-symbols-outlined text-primary text-lg">{icon}</span>
        </span>
      )}
      {children}
    </h3>
  );
}

/**
 * CardDescription Component
 *
 * A description text for cards.
 */
export function CardDescription({ children, className = '' }) {
  return <p className={`text-text-secondary text-sm leading-relaxed ${className}`}>{children}</p>;
}

/**
 * CardContent Component
 *
 * A content wrapper for cards.
 */
export function CardContent({ children, className = '' }) {
  return <div className={className}>{children}</div>;
}

/**
 * CardFooter Component
 *
 * A footer section for cards, typically for actions.
 */
export function CardFooter({ children, className = '' }) {
  return (
    <div className={`flex items-center justify-end gap-3 mt-6 pt-4 border-t border-border-default ${className}`}>
      {children}
    </div>
  );
}
