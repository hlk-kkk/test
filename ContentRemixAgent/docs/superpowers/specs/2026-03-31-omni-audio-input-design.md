# 全模态音频直传设计

> 日期: 2026-03-31
> 状态: 已确认

## 背景

当前 Agent 处理视频内容的流程：下载视频 → 提取音频 → ASR 转录为文字 → LLM 分析文字。

部分全模态模型（如 Qwen3.5-Omni、Gemini 3.1 Pro）支持直接接收音频输入进行推理，无需 ASR 转录。本设计新增一条分支路径，让全模态模型直接"听"音频内容，同时不影响非全模态模型的现有流程。

## 核心决策

| 决策项 | 结论 |
|--------|------|
| 模态范围 | 第一阶段只做音频输入，不做视频直传 |
| 数据库字段命名 | `support_omni`（便于未来扩展到视频） |
| 音频传递方式 | URL（上传七牛云获取公网 URL） |
| 对象存储 | 七牛云 SDK（`pip install qiniu`） |
| 架构模式 | Tool 分支 + 音频注入中间件（方案 A） |

## 架构概览

```
下载视频 → 提取音频(WAV)
    │
    ├─ [support_omni=true] → 上传七牛云 → 存 omni_audio_url → 跳过 ASR
    │   中间件注入音频 URL 到 HumanMessage → LLM 直接听音频分析
    │
    └─ [support_omni=false] → ASR 转录 → 存 transcript（现有流程不变）
        LLM 读取转录文字分析
```

## 详细设计

### 1. 数据库变更

`llm_configs` 表新增字段：

```sql
ALTER TABLE llm_configs ADD COLUMN support_omni BOOLEAN DEFAULT 0;
```

### 2. 配置变更 (settings.py)

新增七牛云配置项：

```python
# 七牛云对象存储（全模态音频上传）
QINIU_ACCESS_KEY: Optional[str] = None
QINIU_SECRET_KEY: Optional[str] = None
QINIU_BUCKET: Optional[str] = None
QINIU_DOMAIN: Optional[str] = None          # CDN 域名，如 cdn.example.com
QINIU_DELETE_AFTER_DAYS: int = 1             # 音频文件自动过期天数
```

### 3. 新服务：QiniuStorageService

**文件:** `backend/services/qiniu_storage.py`

职责：上传音频文件到七牛云，返回公网 URL，文件自动过期删除。

```python
class QiniuStorageService:
    """七牛云存储服务"""

    def is_available(self) -> bool:
        """检查七牛配置是否完整（AK/SK/Bucket/Domain 均已配置）"""

    async def upload_audio(self, local_path: str, key: str) -> str:
        """
        上传音频到七牛，返回公网 URL

        Args:
            local_path: 本地音频文件路径
            key: 七牛存储 key，格式: audio/{session_id}/{content_id}.wav

        Returns:
            公网 URL: https://{domain}/{key}

        实现:
            - 使用 qiniu.put_file_v2() 上传（自动选择表单/分片）
            - policy 设置 deleteAfterDays 自动过期
            - 通过 asyncio.to_thread 包装同步 SDK
        """
```

key 命名规则：`audio/{session_id}/{content_id}.wav`

### 4. State 变更 (state.py)

**RemixAgentState 新增字段：**

```python
# 全模态音频 URL（七牛云公网 URL，供 omni 模型直接理解）
omni_audio_url: Annotated[Optional[str], last_value]
```

**RemixContext 新增字段：**

```python
config_name: Optional[str] = None  # LLM 配置名称（用于判断 omni 等能力）
```

### 5. LLM Provider 变更 (llm_provider.py)

**SQL 查询扩展：** `_query_llm_config` 的 SELECT 加入 `support_omni` 字段。

**新增函数：**

```python
def is_omni_enabled(config_name: Optional[str] = None) -> bool:
    """
    检查配置是否为全模态模型

    优先级:
    1. 指定的 config_name 配置
    2. 数据库激活配置的 support_omni 字段
    3. 默认 False
    """
```

### 6. process_video Tool 变更

在提取音频之后、ASR 之前加入分支判断：

```python
# 提取音频完成后
from llm_provider import is_omni_enabled

config_name = getattr(runtime.context, 'config_name', None)
if is_omni_enabled(config_name):
    # === Omni 分支：上传音频到七牛，跳过 ASR ===
    from services.qiniu_storage import QiniuStorageService
    qiniu_service = QiniuStorageService()

    if not qiniu_service.is_available():
        # 七牛未配置，回退到 ASR 流程
        logger.warning("Qiniu not configured, falling back to ASR")
        # ... 继续 ASR 流程
    else:
        key = f"audio/{session_id}/{content_id}.wav"
        audio_url = await qiniu_service.upload_audio(audio_path, key)

        return Command(update={
            "messages": [ToolMessage(
                content=f"音频已提取（时长 {duration}），模型将直接分析音频内容。",
                tool_call_id=runtime.tool_call_id
            )],
            "omni_audio_url": audio_url,
            "current_stage": "音频已上传",
        })

# === 非 Omni：现有 ASR 流程（完全不变）===
```

**兜底策略：** 如果七牛未配置，自动回退到现有 ASR 流程，确保功能可用。

### 7. 新中间件：omni_audio_injection_middleware

**文件:** `backend/agent/middleware/omni_audio_injection.py`

与现有 `multimodal_injection_middleware`（图片注入）完全对称。

```python
@before_model
def omni_audio_injection_middleware(state, runtime):
    """
    全模态音频注入中间件

    触发条件:
    1. is_omni_enabled() 返回 True
    2. state.omni_audio_url 非空
    3. 存在 HumanMessage

    行为:
    将音频 URL 注入到最后一条 HumanMessage 的 content 中：
    {"type": "input_audio", "input_audio": {"url": "https://..."}}

    避免重复注入（检查 content 列表是否已有 input_audio 类型）。
    """
```

### 8. System Prompt 变更 (prompts.py)

新增全模态音频提示段落：

```python
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

在 `remix_dynamic_prompt` 中，当 omni 模式启用时追加此段落：

```python
config_name = getattr(context, 'config_name', None)
if is_omni_enabled(config_name):
    full_prompt += "\n\n" + OMNI_AUDIO_GUIDE
```

### 9. Agent 工厂变更 (remix_agent.py)

中间件列表新增 `omni_audio_injection_middleware`：

```python
middleware=[
    context_compression_middleware,
    multimodal_injection_middleware,
    omni_audio_injection_middleware,   # 新增：全模态音频注入
    remix_dynamic_prompt,
    token_tracking_middleware,
]
```

### 10. API Route 变更 (remix.py)

构建 RemixContext 时传入 config_name：

```python
context = RemixContext(
    session_id=session_id,
    mode=mode,
    instruction=instruction,
    config_name=resolved_config_name,  # 新增
    use_mock=settings.USE_MOCK,
)
```

### 11. 前端变更

**LLMConfigManager.jsx：**
- 表单新增"全模态模型"开关（`support_omni` boolean）
- 提示文字："启用后将跳过语音转录，直接将音频传给模型理解"

**ModelSelector.jsx：**
- 全模态配置显示 omni 标记/badge

**api.js：**
- create/update LLM 配置接口增加 `support_omni` 字段

## 变更文件清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `config/settings.py` | 修改 | 新增 QINIU_* 配置项 |
| `services/qiniu_storage.py` | **新建** | 七牛云上传服务 |
| `agent/state.py` | 修改 | 新增 omni_audio_url 和 RemixContext.config_name |
| `llm_provider.py` | 修改 | SELECT 加 support_omni、新增 is_omni_enabled() |
| `agent/tools/video_processor_tool.py` | 修改 | 添加 omni 分支（跳过 ASR，上传七牛） |
| `agent/middleware/omni_audio_injection.py` | **新建** | 全模态音频注入中间件 |
| `agent/middleware/__init__.py` | 修改 | 导出新中间件 |
| `agent/prompts.py` | 修改 | 新增 OMNI_AUDIO_GUIDE |
| `agent/remix_agent.py` | 修改 | 注册新中间件 |
| `api/routes/remix.py` | 修改 | RemixContext 传入 config_name |
| `frontend/.../LLMConfigManager.jsx` | 修改 | 新增 support_omni 开关 |
| `frontend/.../ModelSelector.jsx` | 修改 | 显示 omni 标记 |
| `frontend/.../api.js` | 修改 | 配置接口增加 support_omni |

## 不变的部分

以下模块在此次变更中完全不受影响：

- ASR 服务 (`asr_service.py`)
- 链接解析工具 (`link_parser_tool.py`)
- 内容获取工具 (`fetch_content_tool.py`)
- 流式处理器 (`stream/processor.py`)
- Context 压缩中间件
- Token 追踪中间件
- 已有的多模态图片注入中间件
- 非 omni 模型的完整调用链路
