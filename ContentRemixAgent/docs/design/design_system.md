# 设计系统 - 配色指南

本项目采用 **Tailwind CSS 4 + CSS 变量** 的配色系统，所有颜色集中定义在 `frontend/src/index.css` 的 `@theme` 块中。

## 如何修改配色

只需修改 `frontend/src/index.css` 中的颜色变量即可全局生效，**无需修改任何 `.jsx` 文件**。

```css
/* frontend/src/index.css */
@theme {
  --color-primary: #3b82f6;  /* 修改这里即可改变所有主色调 */
  /* ... */
}
```

## 颜色变量一览

### 背景色 (Background)

| 变量名 | 色值 | 用途 | Tailwind 类名 |
|--------|------|------|---------------|
| `--color-bg-primary` | `#09090b` | 主背景（页面背景） | `bg-bg-primary` |
| `--color-bg-secondary` | `#18181b` | 次级背景（卡片、表单） | `bg-bg-secondary` |
| `--color-bg-tertiary` | `#27272a` | 三级背景（hover 状态） | `bg-bg-tertiary` |
| `--color-bg-elevated` | `#3f3f46` | 提升背景（active 状态） | `bg-bg-elevated` |

### 边框色 (Border)

| 变量名 | 色值 | 用途 | Tailwind 类名 |
|--------|------|------|---------------|
| `--color-border-default` | `#27272a` | 默认边框 | `border-border-default` |
| `--color-border-subtle` | `#3f3f46` | 细微边框（hover 状态） | `border-border-subtle` |

### 文字色 (Text)

| 变量名 | 色值 | 用途 | Tailwind 类名 |
|--------|------|------|---------------|
| `--color-text-primary` | `#fafafa` | 主文本（标题、重要内容） | `text-text-primary` |
| `--color-text-secondary` | `#a1a1aa` | 次文本（正文、描述） | `text-text-secondary` |
| `--color-text-muted` | `#71717a` | 弱文本（提示、占位符） | `text-text-muted` |

### 强调色 (Accent)

| 变量名 | 色值 | 用途 | Tailwind 类名 |
|--------|------|------|---------------|
| `--color-primary` | `#3b82f6` | 主强调色（按钮、链接） | `bg-primary`, `text-primary` |
| `--color-primary-hover` | `#2563eb` | 主强调色 hover | `bg-primary-hover` |
| `--color-accent` | `#3b82f6` | 强调色（别名） | `bg-accent`, `text-accent` |
| `--color-accent-muted` | `rgba(59, 130, 246, 0.1)` | 淡强调色（背景） | `bg-accent-muted` |

### 状态色 (Status)

| 变量名 | 色值 | 用途 | Tailwind 类名 |
|--------|------|------|---------------|
| `--color-success` | `#22c55e` | 成功状态 | `bg-success`, `text-success` |
| `--color-error` | `#ef4444` | 错误状态 | `bg-error`, `text-error` |
| `--color-warning` | `#f59e0b` | 警告状态 | `bg-warning`, `text-warning` |

### 辅助强调色 (Secondary Accents)

用于图表、标签等需要多色区分的场景：

| 变量名 | 色值 | Tailwind 类名 |
|--------|------|---------------|
| `--color-accent-cyan` | `#3b82f6` | `text-accent-cyan` |
| `--color-accent-orange` | `#f59e0b` | `text-accent-orange` |
| `--color-accent-green` | `#22c55e` | `text-accent-green` |
| `--color-accent-pink` | `#ec4899` | `text-accent-pink` |
| `--color-accent-purple` | `#8b5cf6` | `text-accent-purple` |

## 使用示例

### 在 JSX 中使用

```jsx
// 背景色
<div className="bg-bg-primary">主背景</div>
<div className="bg-bg-secondary">卡片背景</div>

// 文字色
<h1 className="text-text-primary">标题</h1>
<p className="text-text-secondary">正文描述</p>
<span className="text-text-muted">提示文字</span>

// 按钮
<button className="bg-primary hover:bg-primary-hover text-white">
  主按钮
</button>

// 边框
<div className="border border-border-default hover:border-border-subtle">
  带边框的卡片
</div>

// 状态色
<span className="text-success">成功</span>
<span className="text-error">错误</span>
<span className="text-warning">警告</span>

// 透明度变体（Tailwind 内置支持）
<div className="bg-primary/10">10% 透明度的主色背景</div>
<div className="bg-error/20 border border-error/30">错误提示框</div>
```

### 完整的卡片示例

```jsx
<div className="bg-bg-secondary border border-border-default rounded-xl p-6 hover:border-border-subtle transition-all">
  <h3 className="text-text-primary font-semibold mb-2">卡片标题</h3>
  <p className="text-text-secondary text-sm mb-4">卡片描述文字</p>
  <button className="bg-primary hover:bg-primary-hover text-white px-4 py-2 rounded-lg">
    操作按钮
  </button>
</div>
```

### 表单输入框示例

```jsx
<input
  type="text"
  className="w-full px-4 py-2.5 bg-bg-primary border border-border-default rounded-lg
             text-text-primary placeholder-text-muted
             focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20
             transition-all"
  placeholder="请输入..."
/>
```

## 配色主题切换

如果需要切换到其他配色主题，只需修改 `@theme` 块中的颜色值。以下是一些预设主题参考：

### 默认主题（当前）- Vercel/Linear 风格

```css
@theme {
  --color-bg-primary: #09090b;
  --color-bg-secondary: #18181b;
  --color-primary: #3b82f6;
  --color-text-primary: #fafafa;
}
```

### 暖色主题 - Orange

```css
@theme {
  --color-bg-primary: #0c0a09;
  --color-bg-secondary: #1c1917;
  --color-primary: #f97316;
  --color-text-primary: #fafaf9;
}
```

### 绿色主题 - Emerald

```css
@theme {
  --color-bg-primary: #022c22;
  --color-bg-secondary: #064e3b;
  --color-primary: #10b981;
  --color-text-primary: #ecfdf5;
}
```

### 紫色主题 - Violet

```css
@theme {
  --color-bg-primary: #0f0720;
  --color-bg-secondary: #1e1033;
  --color-primary: #8b5cf6;
  --color-text-primary: #f5f3ff;
}
```

## 设计原则

1. **高对比度**：主文本与背景的对比度至少达到 WCAG AA 标准（4.5:1）
2. **层级分明**：通过 `primary → secondary → tertiary` 的背景色层级区分内容层次
3. **一致性**：全局使用统一的颜色变量，避免硬编码颜色值
4. **克制的强调**：强调色仅用于重要的交互元素，避免过度使用

## 文件结构

```
frontend/src/
├── index.css              # 配色变量定义（@theme 块）
├── components/
│   ├── ui/
│   │   ├── Button.jsx     # 使用 bg-primary 等类名
│   │   └── Card.jsx       # 使用 bg-bg-secondary 等类名
│   ├── Auth/
│   │   └── *.jsx          # 使用统一配色变量
│   └── LandingPage.jsx    # 使用统一配色变量
```

## 兼容性说明

为了兼容旧代码，以下变量名也被保留：

| 旧变量名 | 映射到 |
|----------|--------|
| `--color-background-dark` | `--color-bg-primary` |
| `--color-card-dark` | `--color-bg-secondary` |
| `--color-slate-border` | `--color-border-default` |

建议新代码使用新变量名（`bg-primary`, `bg-secondary` 等），以保持一致性。
