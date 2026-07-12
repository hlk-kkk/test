# Web UI 图片编辑交互设计方案

## 1. 概述

在现有图片生成结果的基础上，用户可以选中某张生成的图片进行「编辑重生成」：输入新的提示词，可选 prompt 增强，调用后端对该图片进行重新生成，完成后原地替换。支持撤销（恢复原始图片）和编辑历史记录。

## 2. 交互流程

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户操作流程                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. 鼠标悬停图片 → 显示编辑按钮（铅笔图标）                       │
│                   ↓                                             │
│  2. 点击编辑按钮 → 图片卡片展开编辑面板                           │
│                    ├── prompt 输入框（预填当前 enhanced_prompt）   │
│                    ├── 增强开关（默认开）                         │
│                    ├── 确认按钮 / 取消按钮                        │
│                   ↓                                             │
│  3. 点击确认 → 图片区域显示加载动画（shimmer + spinner）          │
│               后端开始单图重生成                                  │
│                   ↓                                             │
│  4. 生成完成 → 新图片淡入替换原图                                 │
│               显示撤销按钮（3秒后自动隐藏，悬停时常驻）            │
│                   ↓                                             │
│  5. 点击撤销 → 恢复原始图片                                      │
│     （可多级撤销，恢复到上一次编辑的版本）                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 组件结构

### 3.1 组件树

```
ImageGallery (现有)
├── ImageCard (现有，增加编辑按钮)
│   ├── EditButton (新增，hover 时显示)
│   └── ImageEditOverlay (新增，编辑中的加载层)
├── ImageEditPanel (新增，inline 编辑面板)
│   ├── prompt textarea
│   ├── enhance toggle
│   └── confirm / cancel buttons
├── ImageModal (现有，增加编辑入口)
│   └── EditButton (Modal 内的编辑按钮)
└── UndoBanner (新增，编辑完成后显示)
```

### 3.2 新增组件

#### `ImageEditPanel`

```
frontend/src/components/gallery/ImageEditPanel.jsx
```

内联编辑面板，在被编辑的 ImageCard 下方展开显示。

**Props：**

| Prop | 类型 | 说明 |
|------|------|------|
| `image` | GeneratedImage | 被编辑的图片对象 |
| `defaultPrompt` | string | 预填的 prompt（来自 enhanced_prompt） |
| `onConfirm` | (editPrompt: string, enhance: boolean) => void | 确认编辑回调 |
| `onCancel` | () => void | 取消编辑回调 |
| `isLoading` | boolean | 是否正在生成中 |

**UI 结构：**

```
┌──────────────────────────────────────────┐
│  Edit Image                        [X]  │
│                                          │
│  ┌────────────────────────────────────┐  │
│  │ prompt textarea (auto-expand)     │  │
│  │                                    │  │
│  └────────────────────────────────────┘  │
│                                          │
│  [x] Enhance prompt                      │
│                                          │
│  [ Cancel ]            [ Regenerate ]    │
└──────────────────────────────────────────┘
```

#### `ImageEditOverlay`

```
frontend/src/components/gallery/ImageEditOverlay.jsx
```

覆盖在正在编辑的 ImageCard 上的加载动画层。

**Props：**

| Prop | 类型 | 说明 |
|------|------|------|
| `visible` | boolean | 是否显示 |

**视觉效果：**
- 半透明黑色遮罩 `bg-black/40`
- 中央 spinner 动画 + "Regenerating..." 文字
- 复用现有的 shimmer 动画 (`animate-shimmer`)

#### `UndoBanner`

```
frontend/src/components/gallery/UndoBanner.jsx
```

编辑完成后在图片卡片底部显示的撤销提示条。

**Props：**

| Prop | 类型 | 说明 |
|------|------|------|
| `onUndo` | () => void | 撤销回调 |
| `autoHideMs` | number | 自动隐藏延时，默认 5000ms |

## 4. 状态管理方案

### 4.1 新增 Hook: `useImageEdit`

```
frontend/src/hooks/useImageEdit.js
```

管理图片编辑的所有状态，与现有 `useGeneration` 平级使用。

**状态定义：**

```javascript
function useImageEdit(taskId) {
  // 当前正在编辑的图片 ID
  const [editingImageId, setEditingImageId] = useState(null);

  // 编辑加载状态 Map<imageId, boolean>
  const [loadingImages, setLoadingImages] = useState({});

  // 编辑历史 Map<imageId, EditHistoryEntry[]>
  const [editHistory, setEditHistory] = useState({});

  // 操作方法
  return {
    editingImageId,
    loadingImages,
    editHistory,
    startEdit,       // (imageId) => void
    cancelEdit,      // () => void
    confirmEdit,     // (imageId, prompt, enhance) => Promise<void>
    undoEdit,         // (imageId) => void
    getEditCount,     // (imageId) => number
    isEditing,        // (imageId) => boolean
    isLoading,        // (imageId) => boolean
    canUndo,          // (imageId) => boolean
  };
}
```

### 4.2 编辑历史数据结构

```typescript
interface EditHistoryEntry {
  timestamp: number;
  originalImage: GeneratedImage;    // 编辑前的图片数据
  editPrompt: string;               // 使用的编辑 prompt
  enhanceEnabled: boolean;          // 是否启用了增强
  resultImage: GeneratedImage;      // 编辑后的图片数据
}

// 存储结构: Map<imageId, EditHistoryEntry[]>
// 最新的 entry 在数组末尾
// 撤销 = pop 最后一个 entry，恢复 originalImage
```

### 4.3 与现有状态的集成

编辑完成后需要更新 `useGeneration` 中的 `result` 状态（替换对应图片）。通过回调函数实现：

```javascript
// 在主页面组件中
const { result, ...generation } = useGeneration();
const imageEdit = useImageEdit(taskIdRef.current);

// 编辑完成后更新 result
const handleEditComplete = (imageId, newImage) => {
  setResult(prev => ({
    ...prev,
    rounds: prev.rounds.map(round => ({
      ...round,
      images: round.images.map(img =>
        img.id === imageId ? { ...img, ...newImage } : img
      ),
    })),
    best_images: prev.best_images.map(img =>
      img.id === imageId ? { ...img, ...newImage } : img
    ),
  }));
};
```

## 5. 后端 API 设计

### 5.1 新增 REST 端点

#### `POST /api/v1/generation/{task_id}/images/{image_id}/edit`

对指定任务中的某张图片进行编辑重生成。

**Request Body：**

```json
{
  "prompt": "新的提示词",
  "enhance_prompt": true,
  "aspect_ratio": "16:9",
  "size": "2K"
}
```

**Schema 定义（新增到 `models/schemas.py`）：**

```python
class ImageEditRequest(BaseModel):
    """Request to edit/regenerate a specific image."""
    prompt: str = Field(..., min_length=1, description="New prompt for regeneration")
    enhance_prompt: bool = Field(default=True, description="Whether to enhance prompt")
    aspect_ratio: str | None = Field(default=None, description="Override aspect ratio")
    size: str | None = Field(default=None, description="Override image size")
```

**Response（成功 200）：**

```json
{
  "success": true,
  "original_image": {
    "id": "abc-123",
    "url": "...",
    "score": 0.75
  },
  "new_image": {
    "id": "def-456",
    "url": "...",
    "local_path": "./storage/images/...",
    "score": 0.88,
    "score_details": { ... },
    "prompt": "增强后的提示词",
    "provider": "gemini",
    "round_number": 1,
    "metadata": {
      "edit_source_image_id": "abc-123",
      "edit_prompt": "原始编辑 prompt"
    }
  }
}
```

**Schema 定义（新增到 `models/schemas.py`）：**

```python
class ImageEditResult(BaseModel):
    """Result of an image edit operation."""
    success: bool
    original_image: GeneratedImage
    new_image: GeneratedImage
    enhanced_prompt: str | None = None
```

**错误响应：**

| 状态码 | 场景 |
|--------|------|
| 400 | prompt 为空 |
| 404 | task_id 或 image_id 不存在 |
| 409 | 该图片正在编辑中（防止并发编辑） |
| 500 | 生成失败 |

### 5.2 路由实现

```
backend/src/pictactic/api/routes/generation.py  （在现有文件中添加）
```

```python
@router.post(
    "/{task_id}/images/{image_id}/edit",
    response_model=ImageEditResult,
    responses={404: {"model": ErrorResponse}, 409: {"model": ErrorResponse}},
)
async def edit_image(
    task_id: str,
    image_id: str,
    request: ImageEditRequest,
    req: Request = None,
    user: User = Depends(get_current_user),
):
    """
    Edit/regenerate a specific image within a completed task.

    This endpoint:
    1. Validates the task and image exist
    2. Optionally enhances the new prompt
    3. Generates a single replacement image using the same provider
    4. Returns both original and new image data
    """
    service = get_generation_service()
    lang = _lang(req) if req else "en"
    try:
        result = await service.edit_image(
            task_id=task_id,
            image_id=image_id,
            prompt=request.prompt,
            enhance_prompt=request.enhance_prompt,
            aspect_ratio=request.aspect_ratio,
            size=request.size,
            user_id=user.user_id,
        )
        return result
    except TaskNotFoundError:
        raise HTTPException(status_code=404, detail="Task or image not found")
    except ImageEditConflictError:
        raise HTTPException(status_code=409, detail="Image is already being edited")
```

### 5.3 Service 层新增方法

在 `GenerationService` 中新增 `edit_image` 方法：

```python
async def edit_image(
    self,
    task_id: str,
    image_id: str,
    prompt: str,
    enhance_prompt: bool = True,
    aspect_ratio: str | None = None,
    size: str | None = None,
    user_id: str | None = None,
) -> ImageEditResult:
    """Edit a single image by regenerating with a new prompt."""

    # 1. 验证 task 存在且属于当前用户
    # 2. 找到原始图片
    # 3. 获取原任务的 provider 和参数
    # 4. 可选: enhance prompt (调用 LLM)
    # 5. 调用 provider.generate() 生成单张图片
    # 6. 可选: 评分 (调用 evaluate 节点)
    # 7. 更新内存中的 task state
    # 8. 返回 ImageEditResult
```

**关键实现细节：**

- 编辑操作**不走完整 workflow**（不需要多轮迭代），而是直接调用 provider 的 `generate()` 方法生成单张图片
- prompt 增强复用现有 `enhance` 节点的逻辑（提取为独立函数）
- 评分复用现有 `evaluate` 节点逻辑
- 使用并发锁防止同一图片被并发编辑

### 5.4 新增异常类

```python
# backend/src/pictactic/core/exceptions.py

class ImageEditConflictError(Exception):
    """Raised when an image is already being edited."""
    pass
```

## 6. 前端 API 客户端扩展

在 `frontend/src/lib/api.js` 的 `ApiClient` 中新增方法：

```javascript
async editImage(taskId, imageId, { prompt, enhancePrompt = true, aspectRatio, size }) {
  const response = await this._fetchWithAuth(
    `${this.baseUrl}/api/v1/generation/${taskId}/images/${imageId}/edit`,
    {
      method: "POST",
      headers: this._headers({ "Content-Type": "application/json" }),
      body: JSON.stringify({
        prompt,
        enhance_prompt: enhancePrompt,
        aspect_ratio: aspectRatio,
        size: size,
      }),
    }
  );
  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.detail || "Failed to edit image");
  }
  return response.json();
}
```

## 7. ImageCard 改造

在现有 `ImageCard` 组件的 hover overlay 中增加编辑按钮：

```jsx
// 在现有的底部 hover overlay 中添加编辑按钮
<div className="flex gap-1">
  {/* 编辑按钮（新增） */}
  {onEdit && (
    <button
      onClick={(e) => {
        e.stopPropagation();
        onEdit(image);
      }}
      className="p-1.5 bg-white/10 backdrop-blur-md rounded-md hover:bg-white/20
                 text-white transition-colors"
    >
      <Pencil size={12} />
    </button>
  )}
  {/* 现有的放大和下载按钮 */}
  ...
</div>
```

**新增 Props：**

| Prop | 类型 | 说明 |
|------|------|------|
| `onEdit` | (image) => void | 点击编辑按钮回调，为 null 时不显示编辑按钮 |
| `isEditLoading` | boolean | 该图片是否正在编辑重生成中 |
| `editCount` | number | 该图片的编辑次数（显示标记） |

## 8. ImageModal 改造

在 `ImageModal` 的操作区域增加编辑按钮：

```jsx
{/* 在 Download 按钮上方添加 Edit 按钮 */}
<button
  onClick={() => onEdit?.(image)}
  className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg
             bg-secondary py-2 text-xs font-semibold text-secondary-foreground
             transition-all hover:brightness-110"
>
  <Pencil className="h-3.5 w-3.5" />
  {t("modal.edit")}
</button>
```

## 9. 编辑历史面板

编辑历史记录保存在 `useImageEdit` hook 的内存中（不持久化到后端），用于支持撤销操作。

在 `ImageModal` 中增加编辑历史标签页：

```
┌──────────────────────────────────────────┐
│ [Details]  [Edit History (3)]            │
│                                          │
│  v3 (current) - 2024-01-15 14:30        │
│  "更科技感的配色"                         │
│  Score: 0.92                             │
│                                          │
│  v2 - 2024-01-15 14:25                  │
│  "添加渐变光效"                           │
│  Score: 0.85    [Restore]                │
│                                          │
│  v1 (original) - 2024-01-15 14:20       │
│  Score: 0.75    [Restore]                │
└──────────────────────────────────────────┘
```

## 10. 状态流转图

```
ImageCard
  │
  ├── [hover] → 显示编辑按钮
  │
  ├── [click edit] → editingImageId = image.id
  │                  ImageEditPanel 展开
  │
  ├── [cancel edit] → editingImageId = null
  │                   ImageEditPanel 收起
  │
  ├── [confirm edit] → loadingImages[id] = true
  │                    ImageEditOverlay 显示
  │                    API 调用: editImage(taskId, imageId, ...)
  │                    │
  │                    ├── [成功] → 更新 result 中的图片
  │                    │           记录 editHistory
  │                    │           loadingImages[id] = false
  │                    │           editingImageId = null
  │                    │           显示 UndoBanner
  │                    │
  │                    └── [失败] → loadingImages[id] = false
  │                                显示错误提示（toast）
  │
  └── [undo] → 从 editHistory pop 最近一条
               恢复 result 中的图片为 originalImage
```

## 11. 适配现有架构

### 11.1 兼容性

- 编辑功能仅在 `status === "completed"` 时可用
- 编辑按钮通过 `onEdit` prop 控制显示，不影响流式生成阶段的 ImageCard
- `useImageEdit` 独立于 `useGeneration`，通过回调函数连接

### 11.2 样式适配

- 编辑按钮使用 `lucide-react` 的 `Pencil` 图标，与现有 `Download`、`Maximize2` 风格统一
- 编辑面板使用 TailwindCSS 4 的 `bg-card`、`border-border` 等语义 token
- 加载动画复用现有的 `animate-shimmer` 和 `animate-fade-in-scale`

### 11.3 Provider 适配

编辑重生成使用原任务的 provider 设置。由于所有 provider 都实现了 `ImageProvider.generate()` 单图接口，编辑功能天然兼容所有 provider。

如果原任务使用了模板图片（I2I 模式），编辑操作默认切换为 T2I 模式（纯文本生成），因为编辑 prompt 是全新的描述。用户如需保持 I2I 模式，可在未来迭代中添加「参考原图」选项。

## 12. i18n 键值

新增的国际化键值：

```json
{
  "edit.button": "Edit",
  "edit.title": "Edit Image",
  "edit.promptPlaceholder": "Enter new prompt...",
  "edit.enhanceToggle": "Enhance prompt",
  "edit.confirm": "Regenerate",
  "edit.cancel": "Cancel",
  "edit.loading": "Regenerating...",
  "edit.success": "Image updated",
  "edit.failed": "Edit failed",
  "edit.undo": "Undo",
  "edit.undoHint": "Click to restore previous version",
  "edit.history": "Edit History",
  "edit.historyCount": "Edit History ({count})",
  "edit.version": "v{version}",
  "edit.original": "original",
  "edit.current": "current",
  "edit.restore": "Restore",
  "modal.edit": "Edit Image"
}
```

## 13. 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `backend/src/pictactic/models/schemas.py` | 修改 | 新增 ImageEditRequest, ImageEditResult |
| `backend/src/pictactic/core/exceptions.py` | 修改 | 新增 ImageEditConflictError |
| `backend/src/pictactic/services/generation_service.py` | 修改 | 新增 edit_image() 方法 |
| `backend/src/pictactic/api/routes/generation.py` | 修改 | 新增 edit_image 路由 |
| `frontend/src/lib/api.js` | 修改 | 新增 editImage() 方法 |
| `frontend/src/hooks/useImageEdit.js` | 新增 | 图片编辑状态管理 hook |
| `frontend/src/components/gallery/ImageEditPanel.jsx` | 新增 | 内联编辑面板 |
| `frontend/src/components/gallery/ImageEditOverlay.jsx` | 新增 | 编辑加载动画层 |
| `frontend/src/components/gallery/UndoBanner.jsx` | 新增 | 撤销提示条 |
| `frontend/src/components/gallery/ImageCard.jsx` | 修改 | 增加编辑按钮和相关 props |
| `frontend/src/components/gallery/ImageGallery.jsx` | 修改 | 传递编辑相关 props |
| `frontend/src/components/gallery/ImageModal.jsx` | 修改 | 增加编辑按钮和历史面板 |
| `frontend/src/i18n/` | 修改 | 新增编辑相关翻译键值 |
