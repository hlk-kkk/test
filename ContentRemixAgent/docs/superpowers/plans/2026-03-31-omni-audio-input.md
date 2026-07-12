# Omni Audio Input Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let omni-modal models (Qwen3.5-Omni, Gemini) receive audio directly via URL, skipping ASR transcription, while preserving the existing ASR flow for non-omni models.

**Architecture:** Add `support_omni` flag to model config. When enabled, `process_video` uploads extracted audio to Qiniu Cloud instead of running ASR, and a new `@before_model` middleware injects the audio URL into the LLM message. This mirrors the existing `multimodal_injection_middleware` pattern for images.

**Tech Stack:** Python 3.10+ / qiniu SDK / LangChain 1.0 middleware / FastAPI / React

---

### Task 1: Database Migration — Add `support_omni` Field

**Files:**
- Create: `backend/schema/migrations/V010__llm_configs_omni.sql`

- [ ] **Step 1: Create migration file**

```sql
-- V010: Add Omni-Modal Support Field to LLM Configs
-- 标识模型是否为全模态（支持音频/视频/图片输入）

ALTER TABLE llm_configs
    ADD COLUMN support_omni TINYINT DEFAULT 0
    COMMENT '是否为全模态模型（支持音频输入）: 0=否, 1=是'
    AFTER support_multimodal;
```

- [ ] **Step 2: Run migration**

Run: `cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python scripts/run_migrations.py`

Expected: Migration V010 applied successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/schema/migrations/V010__llm_configs_omni.sql
git commit -m "feat: add support_omni field to llm_configs (V010 migration)"
```

---

### Task 2: Backend Config & Schemas — Qiniu + Omni Fields

**Files:**
- Modify: `backend/config/settings.py` (after line 73, the multimodal section)
- Modify: `backend/schemas.py` (LLMConfigCreateRequest, LLMConfigUpdateRequest, LLMConfigInfo)

- [ ] **Step 1: Add Qiniu config to settings.py**

Add after the `MULTIMODAL_MAX_DIMENSION` line (line 73) in the multimodal section:

```python
    # ========== 七牛云对象存储（全模态音频上传）==========
    QINIU_ACCESS_KEY: Optional[str] = None
    QINIU_SECRET_KEY: Optional[str] = None
    QINIU_BUCKET: Optional[str] = None
    QINIU_DOMAIN: Optional[str] = None          # CDN 域名，如 cdn.example.com
    QINIU_DELETE_AFTER_DAYS: int = 1             # 音频文件自动过期天数
```

- [ ] **Step 2: Add `support_omni` to Pydantic schemas**

In `backend/schemas.py`, add `support_omni` field to three classes:

**LLMConfigCreateRequest** (after `support_multimodal` at line 250):
```python
    support_omni: bool = Field(False, description="是否为全模态模型（支持音频输入）")
```

**LLMConfigUpdateRequest** (after `support_multimodal` at line 271):
```python
    support_omni: Optional[bool] = Field(None, description="是否为全模态模型")
```

**LLMConfigInfo** (after `support_multimodal` at line 294):
```python
    support_omni: bool = Field(False, description="是否为全模态模型")
```

- [ ] **Step 3: Commit**

```bash
git add backend/config/settings.py backend/schemas.py
git commit -m "feat: add Qiniu config and support_omni schema fields"
```

---

### Task 3: LLM Config Service & API — support_omni CRUD

**Files:**
- Modify: `backend/services/llm_config_service.py`
- Modify: `backend/api/routes/llm_config.py`
- Modify: `backend/llm_provider.py`

- [ ] **Step 1: Update `llm_config_service.py` — list_all SELECT**

In `list_all()` method (line 79), add `support_omni` to the SELECT:

```sql
SELECT
    config_name, provider, is_active,
    api_key IS NOT NULL AND api_key != '' as has_api_key,
    base_url, model_name,
    enable_thinking, thinking_budget_tokens, reasoning_effort,
    support_multimodal, support_omni, description, updated_at
FROM llm_configs
ORDER BY is_active DESC, updated_at DESC
```

In the return mapping (line 91), add:
```python
support_omni=bool(row["support_omni"]),
```

- [ ] **Step 2: Update `llm_config_service.py` — create_config INSERT**

In `create_config()` method (line 166), add `support_omni` to the INSERT:

```sql
INSERT INTO llm_configs (
    config_name, provider, api_key, base_url, model_name,
    enable_thinking, thinking_budget_tokens, reasoning_effort,
    support_multimodal, support_omni, description
) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
```

Add the corresponding value in the execute params:
```python
1 if request.support_omni else 0,
```

- [ ] **Step 3: Update `llm_config_service.py` — update_config**

In `update_config()` method, add after the `support_multimodal` block (around line 248):

```python
if request.support_omni is not None:
    update_fields.append("support_omni = %s")
    values.append(1 if request.support_omni else 0)
```

- [ ] **Step 4: Update `llm_provider.py` — _query_llm_config SELECT**

In `_query_llm_config()` (line 83), add `support_omni` to the SELECT:

```sql
SELECT provider, api_key, base_url, model_name,
       enable_thinking, thinking_budget_tokens, reasoning_effort,
       support_multimodal, support_omni
FROM llm_configs
WHERE {where_clause}
LIMIT 1
```

- [ ] **Step 5: Add `is_omni_enabled()` function to `llm_provider.py`**

Add after the `is_multimodal_enabled()` function (after line 407):

```python
def is_omni_enabled(config_name: Optional[str] = None) -> bool:
    """
    检查配置是否为全模态模型（支持音频输入）

    配置优先级:
    1. 指定的 config_name 配置
    2. 数据库激活配置的 support_omni 字段
    3. 默认 False

    Args:
        config_name: 可选的 LLM 配置名称

    Returns:
        bool: 是否为全模态模型
    """
    db_config = None
    if config_name:
        db_config = _get_db_config_by_name(config_name)
    if not db_config:
        db_config = _get_active_db_config()
    if db_config:
        return bool(db_config.get('support_omni'))
    return False
```

- [ ] **Step 6: Update `llm_config.py` route — get_llm_config response**

In `get_llm_config()` (line 172), add to the LLMConfigInfo constructor:
```python
support_omni=bool(row.get("support_omni", 0)),
```

- [ ] **Step 7: Update templates in `llm_config.py`**

In `get_provider_templates()`, add Qwen3.5-Omni template to the `openai` list:
```python
{
    "name": "Qwen3.5-Omni",
    "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "model_name": "qwen3.5-omni-plus",
    "description": "阿里云 Qwen3.5-Omni 全模态模型",
    "support_omni": True,
    "support_multimodal": True,
},
```

- [ ] **Step 8: Commit**

```bash
git add backend/services/llm_config_service.py backend/api/routes/llm_config.py backend/llm_provider.py
git commit -m "feat: support_omni CRUD in config service and is_omni_enabled()"
```

---

### Task 4: QiniuStorageService — Audio Upload

**Files:**
- Create: `backend/services/qiniu_storage.py`

- [ ] **Step 1: Create QiniuStorageService**

```python
# -*- coding: utf-8 -*-
"""
七牛云存储服务 - 上传音频文件并获取公网 URL

用于全模态模型（omni）的音频直传场景。
音频上传后自动设置过期删除（默认 1 天）。
"""
import asyncio
from typing import Optional

from config import settings
from utils.logger import logger


class QiniuStorageError(Exception):
    """七牛云存储错误"""
    pass


class QiniuStorageService:
    """
    七牛云存储服务

    上传本地音频文件到七牛云，返回公网 URL。
    文件通过 deleteAfterDays 策略自动过期删除。
    """

    def is_available(self) -> bool:
        """检查七牛配置是否完整"""
        return all([
            settings.QINIU_ACCESS_KEY,
            settings.QINIU_SECRET_KEY,
            settings.QINIU_BUCKET,
            settings.QINIU_DOMAIN,
        ])

    async def upload_audio(self, local_path: str, key: str) -> str:
        """
        上传音频文件到七牛云

        Args:
            local_path: 本地音频文件路径
            key: 七牛存储 key，如 audio/{session_id}/{content_id}.wav

        Returns:
            公网访问 URL

        Raises:
            QiniuStorageError: 上传失败或配置不完整
        """
        if not self.is_available():
            raise QiniuStorageError(
                "七牛云配置不完整，请在 .env 中配置 QINIU_ACCESS_KEY, QINIU_SECRET_KEY, QINIU_BUCKET, QINIU_DOMAIN"
            )

        try:
            url = await asyncio.to_thread(self._sync_upload, local_path, key)
            logger.info(f"[Qiniu] Audio uploaded: {key} -> {url}")
            return url
        except QiniuStorageError:
            raise
        except Exception as e:
            raise QiniuStorageError(f"上传失败: {e}") from e

    def _sync_upload(self, local_path: str, key: str) -> str:
        """同步上传（在线程池中执行）"""
        import qiniu

        auth = qiniu.Auth(settings.QINIU_ACCESS_KEY, settings.QINIU_SECRET_KEY)

        # 上传策略：文件 N 天后自动删除
        policy = {
            "deleteAfterDays": settings.QINIU_DELETE_AFTER_DAYS,
        }

        token = auth.upload_token(
            settings.QINIU_BUCKET, key, 3600, policy=policy
        )

        ret, info = qiniu.put_file(token, key, local_path, mime_type="audio/wav")

        if ret is None:
            raise QiniuStorageError(f"七牛上传失败: {info}")

        # 拼接公网 URL
        domain = settings.QINIU_DOMAIN.rstrip("/")
        if not domain.startswith(("http://", "https://")):
            domain = f"https://{domain}"

        return f"{domain}/{key}"
```

- [ ] **Step 2: Install qiniu SDK**

Run: `cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv add qiniu`

Expected: qiniu package added to pyproject.toml.

- [ ] **Step 3: Commit**

```bash
git add backend/services/qiniu_storage.py backend/pyproject.toml backend/uv.lock
git commit -m "feat: QiniuStorageService for omni audio upload"
```

---

### Task 5: State & Context — omni_audio_url + config_name

**Files:**
- Modify: `backend/agent/state.py`

- [ ] **Step 1: Add `omni_audio_url` to RemixAgentState**

Add after `transcript` field (line 127):

```python
    # 全模态音频 URL（七牛云公网 URL，供 omni 模型直接理解音频）
    omni_audio_url: Annotated[Optional[str], last_value]
```

- [ ] **Step 2: Add `config_name` to RemixContext**

Add after `instruction` field (line 160):

```python
    config_name: Optional[str] = None  # LLM 配置名称（用于判断 omni 等能力）
```

- [ ] **Step 3: Commit**

```bash
git add backend/agent/state.py
git commit -m "feat: add omni_audio_url state and config_name context"
```

---

### Task 6: process_video Tool — Omni Branch

**Files:**
- Modify: `backend/agent/tools/video_processor_tool.py`

- [ ] **Step 1: Add omni branch after audio extraction**

In the `process_video` tool function, after the audio extraction step (after line 128, the `await step_callback("end", "extract_audio", ...)` call), before the ASR step (before line 143, `# Step 3: ASR 转录`), insert the omni branch:

```python
        # === Omni 分支：全模态模型直接理解音频，跳过 ASR ===
        from llm_provider import is_omni_enabled
        config_name = getattr(runtime.context, 'config_name', None)

        if is_omni_enabled(config_name):
            from services.qiniu_storage import QiniuStorageService, QiniuStorageError

            qiniu_service = QiniuStorageService()
            if qiniu_service.is_available():
                try:
                    if step_callback:
                        await step_callback("start", "upload_audio", "上传音频到云端...")
                    
                    # 构建七牛存储 key
                    session_id = getattr(runtime.context, 'session_id', 'unknown')
                    content_id = content_info.get("content_id", "unknown")
                    qiniu_key = f"audio/{session_id}/{content_id}.wav"

                    audio_public_url = await qiniu_service.upload_audio(audio_path, qiniu_key)

                    if step_callback:
                        await step_callback("end", "upload_audio", "音频已上传到云端")

                    # 获取音频时长
                    audio_duration_str = "未知"
                    try:
                        probe_cmd = ["ffprobe", "-v", "error", "-show_entries", "format=duration",
                                     "-of", "default=noprint_wrappers=1:nokey=1", audio_path]
                        duration_result = subprocess.run(probe_cmd, capture_output=True, text=True, timeout=30)
                        audio_duration = float(duration_result.stdout.strip())
                        mins, secs = divmod(int(audio_duration), 60)
                        audio_duration_str = f"{mins}:{secs:02d}"
                    except Exception:
                        pass

                    logger.info(f"[process_video] Omni mode: audio uploaded to {audio_public_url}, skipping ASR")

                    # 发送 content_info 事件
                    await adispatch_custom_event(
                        "content_info",
                        {"content_info": dict(content_info)},
                    )

                    # 清理临时文件
                    try:
                        downloader.cleanup(task_id)
                    except Exception as e:
                        logger.warning(f"[{task_id}] Cleanup failed: {e}")

                    return Command(update={
                        "messages": [ToolMessage(
                            content=f"音频已提取并上传（时长 {audio_duration_str}），全模态模型将直接分析音频内容。",
                            tool_call_id=runtime.tool_call_id
                        )],
                        "omni_audio_url": audio_public_url,
                        "current_stage": "音频已上传，等待模型分析",
                    })

                except QiniuStorageError as e:
                    logger.warning(f"[process_video] Qiniu upload failed: {e}, falling back to ASR")
                    # 七牛上传失败，继续走 ASR 流程
            else:
                logger.warning("[process_video] Qiniu not configured, falling back to ASR for omni model")
```

Note: The `import subprocess` is already present via the existing ffprobe usage. The `downloader` and `task_id` variables are already in scope from the earlier download step.

- [ ] **Step 2: Verify the existing flow is untouched**

The omni branch is a pure insertion before the ASR step. If `is_omni_enabled()` returns False or Qiniu fails, execution falls through to the existing ASR code without any changes.

- [ ] **Step 3: Commit**

```bash
git add backend/agent/tools/video_processor_tool.py
git commit -m "feat: omni branch in process_video — skip ASR, upload to Qiniu"
```

---

### Task 7: Omni Audio Injection Middleware

**Files:**
- Create: `backend/agent/middleware/omni_audio_injection.py`
- Modify: `backend/agent/middleware/__init__.py`

- [ ] **Step 1: Create the middleware**

```python
# -*- coding: utf-8 -*-
"""
全模态音频注入中间件

当全模态模式启用且存在音频 URL 时，将音频注入到最后一条 HumanMessage 中。
与 multimodal_injection_middleware（图片注入）对称设计。

基于 LangChain 1.0 的 @before_model 中间件模式实现。
"""
from typing import Optional, Dict, Any, List

from langchain.agents.middleware import before_model
from langchain_core.messages import HumanMessage, BaseMessage

from utils.logger import logger


@before_model
def omni_audio_injection_middleware(state: Dict[str, Any], runtime: Any) -> Optional[Dict[str, Any]]:
    """
    全模态音频注入中间件

    在模型调用前检查是否需要将音频嵌入到消息中。

    触发条件:
    1. 全模态已启用 (is_omni_enabled() 返回 True)
    2. state.omni_audio_url 非空
    3. 存在 HumanMessage

    注入格式: {"type": "input_audio", "input_audio": {"url": "..."}}

    Args:
        state: Agent 状态字典
        runtime: Runtime 对象

    Returns:
        状态更新字典（包含修改后的 messages），或 None 表示无需更新
    """
    from llm_provider import is_omni_enabled

    # 1. 检查全模态是否启用
    config_name = getattr(runtime.context, 'config_name', None) if runtime.context else None
    if not is_omni_enabled(config_name):
        return None

    # 2. 获取音频 URL
    audio_url = state.get("omni_audio_url")
    if not audio_url:
        return None

    # 3. 获取消息列表
    messages = state.get("messages", [])
    if not messages:
        return None

    # 4. 找到最后一条 HumanMessage
    last_human_idx = -1
    for i in range(len(messages) - 1, -1, -1):
        if isinstance(messages[i], HumanMessage):
            last_human_idx = i
            break

    if last_human_idx < 0:
        return None

    last_human_msg = messages[last_human_idx]

    # 5. 检查是否已经注入过音频（避免重复注入）
    if isinstance(last_human_msg.content, list):
        for item in last_human_msg.content:
            if isinstance(item, dict) and item.get("type") == "input_audio":
                logger.debug("[OmniAudioInjection] Audio already injected, skipping")
                return None

    # 6. 构建带音频的消息内容
    original_content = last_human_msg.content
    if isinstance(original_content, str):
        text = original_content
    elif isinstance(original_content, list):
        text_parts = [
            item.get("text", "") if isinstance(item, dict) and item.get("type") == "text"
            else str(item) if not isinstance(item, dict)
            else ""
            for item in original_content
        ]
        text = "\n".join(filter(None, text_parts))
    else:
        text = str(original_content)

    new_content = [
        {"type": "text", "text": text},
        {"type": "input_audio", "input_audio": {"url": audio_url}},
    ]

    new_human_msg = HumanMessage(content=new_content)

    # 7. 构建新的消息列表
    new_messages = list(messages)
    new_messages[last_human_idx] = new_human_msg

    logger.info(f"[OmniAudioInjection] Injected audio URL into message at index {last_human_idx}")

    return {"messages": new_messages}
```

- [ ] **Step 2: Update middleware `__init__.py`**

In `backend/agent/middleware/__init__.py`, add the import and export:

```python
from agent.middleware.omni_audio_injection import omni_audio_injection_middleware
```

Add to `__all__`:
```python
"omni_audio_injection_middleware",
```

- [ ] **Step 3: Commit**

```bash
git add backend/agent/middleware/omni_audio_injection.py backend/agent/middleware/__init__.py
git commit -m "feat: omni_audio_injection_middleware for audio URL injection"
```

---

### Task 8: System Prompt — Omni Audio Guide

**Files:**
- Modify: `backend/agent/prompts.py`

- [ ] **Step 1: Add OMNI_AUDIO_GUIDE constant**

Add before the `MODE_PROMPTS` dict (before line 308):

```python
# ============================================================================
# 全模态音频直接理解指南
# ============================================================================

OMNI_AUDIO_GUIDE = """## 音频直接理解模式

你当前使用的是全模态模型，可以直接"听到"视频的音频内容。

**工作方式变化：**
- process_video 工具会提取音频并直接提供给你，不会进行文字转录
- 你需要直接听取音频内容，理解其中的语音、语气、节奏等信息
- 相比文字转录，你能获取更多维度的信息：语气变化、停顿节奏、情感表达等

**分析提示：**
- 注意说话者的语气和情感变化
- 关注节奏把控：哪里加速、哪里停顿
- 识别背景音乐/音效对内容氛围的影响
"""
```

- [ ] **Step 2: Update `remix_dynamic_prompt` to conditionally include omni guide**

In the `remix_dynamic_prompt` function (line 435), modify the prompt assembly:

```python
@dynamic_prompt
def remix_dynamic_prompt(request: ModelRequest) -> str:
    """根据 context.mode 动态选择 System Prompt"""
    context = request.runtime.context
    mode = getattr(context, 'mode', None) or 'analyze'

    mode_prompt = _get_cached_mode_prompt(mode)

    full_prompt = f"{BASE_PROMPT}\n\n{ANALYSIS_GUIDE}\n\n{mode_prompt}\n\n{OUTPUT_FORMAT}"

    # 全模态模型：追加音频直接理解指南
    config_name = getattr(context, 'config_name', None)
    from llm_provider import is_omni_enabled
    if is_omni_enabled(config_name):
        full_prompt += f"\n\n{OMNI_AUDIO_GUIDE}"

    return full_prompt
```

- [ ] **Step 3: Add to `__all__` exports**

Add `"OMNI_AUDIO_GUIDE"` to the `__all__` list.

- [ ] **Step 4: Commit**

```bash
git add backend/agent/prompts.py
git commit -m "feat: OMNI_AUDIO_GUIDE in dynamic system prompt"
```

---

### Task 9: Agent Factory — Register Middleware

**Files:**
- Modify: `backend/agent/remix_agent.py`

- [ ] **Step 1: Import the new middleware**

Add to the imports (line 43):
```python
from agent.middleware import (
    context_compression_middleware,
    token_tracking_middleware,
    multimodal_injection_middleware,
    omni_audio_injection_middleware,
)
```

- [ ] **Step 2: Add to middleware list**

In `create_remix_agent()` (line 143), add the new middleware after `multimodal_injection_middleware`:

```python
middleware=[
    context_compression_middleware,
    multimodal_injection_middleware,
    omni_audio_injection_middleware,   # @before_model: 全模态音频注入
    remix_dynamic_prompt,
    token_tracking_middleware,
],
```

- [ ] **Step 3: Commit**

```bash
git add backend/agent/remix_agent.py
git commit -m "feat: register omni_audio_injection_middleware in agent"
```

---

### Task 10: API Route — Pass config_name to RemixContext

**Files:**
- Modify: `backend/api/routes/remix.py`

- [ ] **Step 1: Update analyze() — RemixContext construction**

In `analyze()` function (line 305), add `config_name` to the context:

```python
context = RemixContext(
    session_id=session_id,
    mode=mode,
    instruction=request.instruction,
    config_name=effective_config_name,
    use_mock=settings.USE_MOCK,
)
```

- [ ] **Step 2: Update chat() — RemixContext construction**

In `chat()` function (line 475), add `config_name` to the context:

```python
context = RemixContext(
    session_id=session_id,
    config_name=effective_config_name,
    use_mock=settings.USE_MOCK,
)
```

Note: `effective_config_name` is already resolved at this point (line 491 for chat).
The `context` creation for chat is at line 475 but `effective_config_name` is resolved inside `event_generator()` at line 491. We need to move the context creation inside `event_generator()` after the config resolution.

Specifically, move the context creation from line 475 to after line 494 (after `_resolve_llm_config` in the chat's event_generator), and pass `config_name`:

```python
# Inside event_generator(), after effective_config_name is resolved:
context = RemixContext(
    session_id=session_id,
    config_name=effective_config_name,
    use_mock=settings.USE_MOCK,
)
```

And remove the original context creation at line 475.

- [ ] **Step 3: Commit**

```bash
git add backend/api/routes/remix.py
git commit -m "feat: pass config_name to RemixContext in analyze/chat routes"
```

---

### Task 11: Frontend — support_omni Toggle

**Files:**
- Modify: `frontend/src/components/Settings/LLMConfigManager.jsx`

- [ ] **Step 1: Add `support_omni` to INITIAL_FORM**

In INITIAL_FORM (line 50), add:
```javascript
support_omni: false,
```

- [ ] **Step 2: Add `support_omni` to handleEdit**

In `handleEdit` (line 133), add to the formData object:
```javascript
support_omni: item.support_omni || false,
```

Also update `setShowAdvanced` condition (line 147):
```javascript
setShowAdvanced(item.enable_thinking || item.support_multimodal || item.support_omni);
```

- [ ] **Step 3: Add `support_omni` to handleApplyTemplate**

In `handleApplyTemplate` (line 161), add:
```javascript
support_omni: template.support_omni || false,
```

Update the `showAdvanced` condition:
```javascript
if (template.enable_thinking || template.support_multimodal || template.support_omni) {
```

- [ ] **Step 4: Add support_omni checkbox in the Advanced Settings section**

In the Advanced Settings section (after the `support_multimodal` checkbox block ending at line 449), add:

```jsx
{/* Support Omni */}
<div className="flex items-center gap-3">
  <input
    type="checkbox"
    id="support_omni"
    checked={formData.support_omni}
    onChange={(e) => setFormData({ ...formData, support_omni: e.target.checked })}
    className="w-4 h-4 rounded border-slate-border text-primary focus:ring-primary"
  />
  <label htmlFor="support_omni" className="text-sm text-text-secondary">
    {t('llmConfig.supportOmni')}
  </label>
  <span className="text-xs text-text-muted">
    ({t('llmConfig.omniHint')})
  </span>
</div>
```

- [ ] **Step 5: Add omni badge in config list**

In the config list (after the `support_multimodal` badge at line 583), add:

```jsx
{item.support_omni && (
  <span className="text-xs text-amber-400 flex items-center gap-1">
    <span className="material-symbols-outlined text-sm">hearing</span>
    {t('llmConfig.omni')}
  </span>
)}
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/Settings/LLMConfigManager.jsx
git commit -m "feat: support_omni toggle in LLM config UI"
```

---

### Task 12: Frontend i18n — Omni Labels

**Files:**
- Modify: `frontend/src/locales/zh.json` (or equivalent i18n file)
- Modify: `frontend/src/locales/en.json` (or equivalent i18n file)

- [ ] **Step 1: Find and update i18n files**

Search for `supportMultimodal` or `multimodalHint` in the i18n files to find the exact location. Add the following keys in the `llmConfig` section:

**Chinese (zh):**
```json
"supportOmni": "全模态模型",
"omniHint": "跳过语音转录，直接将音频传给模型理解",
"omni": "全模态"
```

**English (en):**
```json
"supportOmni": "Omni-Modal",
"omniHint": "Skip ASR, send audio directly to model",
"omni": "Omni"
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/locales/
git commit -m "feat: i18n labels for omni-modal support"
```

---

### Task 13: Integration Verification

- [ ] **Step 1: Verify backend starts without errors**

Run: `cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python -c "from agent.remix_agent import create_remix_agent; print('OK')"`

Expected: `OK` (no import errors).

- [ ] **Step 2: Verify the middleware import chain**

Run: `cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python -c "from agent.middleware import omni_audio_injection_middleware; print('OK')"`

Expected: `OK`

- [ ] **Step 3: Verify QiniuStorageService import**

Run: `cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python -c "from services.qiniu_storage import QiniuStorageService; s = QiniuStorageService(); print('available:', s.is_available())"`

Expected: `available: False` (since no Qiniu config in .env yet).

- [ ] **Step 4: Verify is_omni_enabled import**

Run: `cd /Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent/backend && uv run python -c "from llm_provider import is_omni_enabled; print('omni:', is_omni_enabled())"`

Expected: `omni: False` (no config marked as omni yet).

- [ ] **Step 5: Commit final integration**

```bash
git add -A
git commit -m "feat: omni audio input — complete integration"
```
