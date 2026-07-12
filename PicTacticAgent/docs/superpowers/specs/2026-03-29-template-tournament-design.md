# Template Tournament: 从 xprompt-watch 精选模板库

## 概述

从 xprompt-watch 项目（2,056 条 X/Twitter AI 生图提示词 + 3,348 张图片）中，通过 Vision LLM 锦标赛淘汰机制，精选 ~50 条高质量提示词作为 PicTacticAgent 的系统预装模板。

## 数据源

**xprompt-watch** (`/Users/nanmi/workspace/myself_code/xprompt-watch/`)

| 项目 | 数量 |
|------|------|
| 提示词 | 2,056 |
| 图片 | 3,348 |
| 作者 | 486 |
| 工具 | 17 种（Nano Banana 73%、Midjourney 5%、Grok 2% 等） |
| 时间跨度 | 2026-02-06 ~ 2026-03-25 |

**数据库**: SQLite (`db/prompts.db`)
- `prompts` 表: tweet_id, author, tool, prompt_text, tweet_url, created_at
- `images` 表: tweet_id, image_index, url, local_path
- `videos` 表: tweet_id, video_index, url, local_path

**提示词格式混合**: 自然语言、结构化 JSON（Nano Banana 参数）、半结构化（prompt + metadata）。保留原始格式不做改写。

## 目标系统

**PicTacticAgent 模板系统** (`PromptTemplate` 模型)

现有字段：id, user_id, title, description, prompt_text, category, tags, preview_image_url, config, usage_count, is_public, created_at, updated_at

**需改动：**
- 增加 `is_system` 字段：区分系统预装模板和用户模板
- 系统模板所有用户可见，不可编辑/删除
- 预览图存储在 `storage/templates/` 目录，通过 `/static/templates/` 提供访问

## 整体流程

```
Stage 1: Load       读取 xprompt-watch 全部 2,056 条（过滤无图片的）
Stage 2: Classify   LLM 分析全部提示词，自动聚类生成分类方案
Stage 3: Assign     每条提示词自动归入一个分类
Stage 4: Tournament 每个分类内独立跑锦标赛（Vision LLM 1v1 对决）
Stage 5: Export     迁移图片 + 生成种子数据脚本
Stage 6: Integrate  修改后端支持系统模板 + 数据库迁移
```

## Stage 1: 数据加载

从 xprompt-watch SQLite 读取数据，过滤掉：
- 无图片的条目
- 纯视频条目（108 条，不含图片的跳过）
- prompt_text 为空的

预计有效候选：~1,900-2,000 条

```python
# data_loader.py
def load_candidates(db_path: str) -> list[Candidate]:
    """从 xprompt-watch 加载候选数据"""
    # SELECT p.*, GROUP_CONCAT(i.local_path) as image_paths
    # FROM prompts p JOIN images i ON p.tweet_id = i.tweet_id
    # GROUP BY p.tweet_id HAVING COUNT(i.id) > 0
```

每个 Candidate 包含：
- tweet_id, author, tool, prompt_text
- image_paths: list[str]（1-4 张图片的本地路径）
- tweet_url

## Stage 2: 主题分类

将全部候选的 prompt_text 发送给 LLM 进行主题聚类分析。

**方法**：随机采样 200 条提示词发给 LLM，要求输出 5-8 个分类及定义。然后用这套分类方案对全部候选打标。

**Prompt 模板**（分类发现）：
```
Analyze these 200 AI image generation prompts.
Identify 5-8 natural topic categories that best organize them.
For each category provide: name (English, lowercase, snake_case),
display_name (English), description (what belongs here).
Output JSON: [{name, display_name, description}, ...]
```

**Prompt 模板**（批量分类）：
```
Given these categories: [list]
Classify each prompt into exactly one category.
Output JSON: [{tweet_id, category}, ...]
```

分批处理（每批 50 条），并发 5 个请求。

**分类方案将替换现有的** `TemplateCategory` 枚举。现有的 cover/portrait/landscape/abstract/product/other 将被数据驱动的新分类替代。

## Stage 3: 锦标赛评分

### 对决机制

每场对决：Vision LLM 同时看到两个候选的图片，选出更好的。

**输入**：候选 A 的最佳图片（选第 1 张）vs 候选 B 的最佳图片

**Prompt**：
```
You are judging an AI image generation tournament.
Compare Image A and Image B on these criteria:
1. Composition and layout
2. Aesthetic quality and color harmony
3. Detail and technical quality
4. Visual impact and creativity
5. Overall polish

Which image is better? Reply ONLY with JSON:
{"winner": "A" or "B", "confidence": 0.0-1.0, "reason": "brief explanation"}
```

### 防偏差

每场对决执行 2 次，A/B 顺序互换：
- 两次结果一致 → 确认胜者
- 两次结果矛盾 → 标记为平局，随机选胜者（下轮优先淘汰）

### 淘汰赛结构

每个分类内独立进行：

```
假设某分类有 N 个候选:
Round 1: N/2 场对决 → N/2 胜者
Round 2: N/4 场对决 → N/4 胜者
...
继续直到剩余 ≤ 目标数量（该分类的配额）
```

**配额分配**：50 个名额按分类内候选数量比例分配，每个分类至少 3 个名额。

**奇数处理**：轮空直接晋级。

### 并发控制

```python
semaphore = asyncio.Semaphore(5)  # 最多 5 个并发 LLM 调用
```

### 结果日志

`results.tsv`（autoresearch 风格）：
```
round	category	match	winner_id	loser_id	confidence	reason	timestamp
1	portrait	1	tweet_123	tweet_456	0.85	"Better composition"	2026-03-29T14:00:00
```

### 成本估算

假设 6 个分类，平均每分类 ~330 个候选：
- 每分类约 330 场对决（含正反两次 = 660 次 LLM 调用）
- 总计：~4,000 次 Vision LLM 调用
- 使用 gemini-3.1-flash-lite（便宜快速）

## Stage 4: 数据导出

### 图片迁移

```
xprompt-watch/images/{tweet_id}/1.jpg
  → PicTacticAgent/storage/templates/{tweet_id}/1.jpg

URL: /static/templates/{tweet_id}/1.jpg
```

对于精选出的 ~50 条，迁移所有关联图片（1-4 张）。主预览图使用第 1 张。

### 种子数据生成

生成 Python 种子脚本 `scripts/seed_templates.py`：

```python
SYSTEM_TEMPLATES = [
    {
        "id": "system-template-{uuid}",
        "title": "...",  # LLM 从 prompt_text 生成简短标题
        "description": "...",  # LLM 生成简短描述
        "prompt_text": "...",  # 原始提示词
        "category": "portrait",
        "tags": ["fashion", "editorial"],  # LLM 自动打标
        "preview_image_url": "/static/templates/{tweet_id}/1.jpg",
        "is_system": True,
        "is_public": True,
        "source_tweet_url": "https://x.com/...",
        "source_author": "@username",
    },
    ...
]
```

标题和描述由 LLM 基于 prompt_text 自动生成（一批处理）。

## Stage 5: 后端集成

### 数据库变更

`PromptTemplate` 模型新增字段：

```python
is_system: Mapped[bool] = mapped_column(default=False)
source_url: Mapped[str | None] = mapped_column(String(512), nullable=True)
source_author: Mapped[str | None] = mapped_column(String(100), nullable=True)
```

### API 变更

`GET /api/v1/templates/` 查询逻辑修改：
- 返回用户自己的模板 + 所有 `is_system=True` 的模板
- 系统模板不可 PUT/DELETE（返回 403）

### 静态文件服务

在 FastAPI app 中挂载：
```python
app.mount("/static/templates", StaticFiles(directory="storage/templates"))
```

### 分类枚举更新

`TemplateCategory` 从硬编码枚举改为数据驱动：
- 从数据库中读取所有已有分类
- 或在 config 中维护分类列表
- 保持向后兼容：旧的 6 个分类仍然有效

### 种子数据执行

启动时自动检查并导入：

```python
# backend/src/pictactic/db/seed.py
async def seed_system_templates(session: AsyncSession):
    """如果 is_system 模板为空，从种子数据导入"""
    existing = await session.execute(
        select(PromptTemplate).where(PromptTemplate.is_system == True).limit(1)
    )
    if existing.scalar_one_or_none() is None:
        # 导入种子数据
        for tmpl in SYSTEM_TEMPLATES:
            session.add(PromptTemplate(**tmpl))
        await session.commit()
```

## 脚本结构

```
scripts/template_tournament/
├── tournament.py          # 主入口，编排整个流程
├── data_loader.py         # 从 xprompt-watch SQLite 读取
├── classifier.py          # LLM 主题聚类 + 分类
├── evaluator.py           # Vision LLM 1v1 对决
├── exporter.py            # 图片迁移 + 种子数据生成
├── llm_client.py          # 复用项目 LLM 配置的客户端封装
└── results.tsv            # 运行时生成的锦标赛日志
```

**依赖**：复用 PicTacticAgent 的 `pyproject.toml` 依赖（openai、aiosqlite），不新增依赖。

**LLM 配置**：读取项目 `.env` 中的 `JIEKOUAI_*` 配置，通过 OpenAI-compatible API 调用。

## 运行方式

```bash
cd /Users/nanmi/workspace/myself_code/PicTacticAgent

# 完整流程（约 30-60 分钟，取决于 LLM 速度）
uv run python scripts/template_tournament/tournament.py

# 可选参数
uv run python scripts/template_tournament/tournament.py \
  --xprompt-db /Users/nanmi/workspace/myself_code/xprompt-watch/db/prompts.db \
  --target-count 50 \
  --concurrency 5 \
  --output-dir ./storage/templates
```

## 前端变更

### TemplateCard 组件

系统模板卡片显示来源标记：
- 显示 `@author` 来源
- 隐藏 Edit/Delete 按钮（系统模板不可修改）
- 显示所有关联图片（而非仅预览图）

### TemplateFilters 组件

分类筛选器改为动态读取（从 API 获取可用分类列表，而非硬编码枚举）。

### TemplatesPage

新增"系统模板"标签页或筛选项，区分系统模板和用户模板。

## 成功标准

1. 锦标赛脚本可一键运行，产出 ~50 条精选模板
2. 精选模板包含：标题、描述、原始提示词、分类、标签、预览图
3. 新用户注册后可立即看到系统模板
4. 系统模板不可编辑/删除，用户可"复制到我的模板"后修改
5. 预览图直接可见，不需要额外生成
6. 锦标赛结果可追溯（results.tsv 记录每轮对决）
