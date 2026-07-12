# 全模态 (Omni-Modal) 架构设计

> 让 Agent 直接"看到"和"听到"原始内容，而非依赖文字转录的间接理解。

## 设计思想

### 为什么需要全模态？

传统流程中，视频内容必须经过 ASR（语音转文字）才能被 LLM 理解。这个过程有两个本质问题：

1. **信息损失**：ASR 只保留了语义文字，丢失了语气、节奏、停顿、背景音乐、画面构图等丰富信息
2. **错误累积**：ASR 本身有识别错误，尤其是专业术语、方言、多人对话场景

全模态模型（如 Qwen3.5-Omni、Gemini）可以直接理解音频和视频输入，让 Agent 获得"第一手"的感知能力，分析维度从"文字内容"扩展到"视听体验"。

### 核心原则

**渐进增强，不破坏存量**

全模态是一种增强能力，不是替代。整个设计遵循：
- 非全模态模型的 ASR 流程完全不变
- 全模态能力通过 `support_omni` 标记按模型配置开启
- 七牛云未配置时，自动降级到 ASR 流程
- 所有分支最终汇聚到同一个 Agent 分析输出

**"谁擅长什么，就让谁做什么"**

- ASR 擅长：快速、低成本、确定性强的文字转录
- 全模态模型擅长：多维度感知、情感理解、视觉分析
- 本设计让它们各司其职，不互相替代

## 架构总览

```
用户提交链接 → parse_link → fetch_content → process_video
                                                  │
                              ┌───────────────────┤
                              │                   │
                        [support_omni=true]  [support_omni=false]
                              │                   │
                        获取视频时长          提取音频 → ASR
                              │                   │
                    ┌─────────┴──────────┐        │
                    │                    │        │
              [短视频 ≤400s]      [长视频 >400s]   │
                    │                    │        │
              转码 720p           ASR 转录全文     │
              上传七牛            LLM 语义分段     │
                    │            切割+转码720p    │
                    │            上传各段七牛      │
                    │                    │        │
              中间件注入视频URL    SubAgent 逐段    │
              模型直接分析        视频分析          │
                    │                    │        │
                    └────────┬───────────┘        │
                             │                    │
                        Agent 输出分析报告 ←───────┘
```

## 两条路径详解

### 路径一：短视频直传（≤400s）

最简单也最有价值的路径。适用于大部分社交媒体短视频（抖音、快手、小红书）。

```
下载视频 → ffmpeg 转码 720p → 上传七牛云 → 获得公网 URL
    → omni_media_injection 中间件注入到 HumanMessage
    → 全模态模型一次性理解视频 → 输出分析
```

**关键组件：**

| 组件 | 职责 |
|------|------|
| `VideoTranscoder.transcode_720p()` | ffmpeg 转码至 720p，平衡质量与体积 |
| `QiniuStorageService.upload_video()` | 上传视频到七牛云，获取 CDN URL，自动过期删除 |
| `omni_media_injection_middleware` | 在 LLM 调用前，将视频 URL 注入到消息内容中 |

**消息注入格式：**

```python
HumanMessage(content=[
    {"type": "text", "text": "请分析这个视频..."},
    {"type": "video_url", "video_url": {"url": "https://cdn.example.com/video.mp4"}},
])
```

### 路径二：长视频分段分析（>400s）

这是一个精心设计的多阶段流水线，解决了全模态模型的输入时长限制。

#### 阶段 1：转录 + 语义分段

长视频仍然需要 ASR 转录——不是为了给模型看文字，而是为了**让 LLM 找到语义上合理的切分点**。

```
长视频 → ASR 转录 → 带时间戳的全文
    → LLM 语义分段（不是硬切！）
    → [{start: 0, end: 185, title: "开场介绍"}, ...]
```

**为什么不硬切？**

硬切（每 5 分钟切一刀）会在句子中间、论点中间、甚至词语中间切断。语义分段让 LLM 理解内容结构，在话题转换、章节过渡等自然断点处切分，每段都是语义完整的单元。

**分段约束：**
- 每段 ≤ 360 秒（配置项 `OMNI_VIDEO_SEGMENT_MAX`，留 40s 余量）
- 段间无间隙、无重叠
- 每段附带 LLM 生成的简短标题

#### 阶段 2：切割 + 转码 + 上传

```
原始视频 + 分段时间点
    → ffmpeg 按时间点切割 + 每段转码 720p
    → 各段上传七牛云 → 获得 URL 列表
```

#### 阶段 3：SubAgent 链式分析

这是整个设计中最有创意的部分——**滚动上下文的 SubAgent 链**。

```
主 Agent
    │
    ├─ 调用 analyze_video_segment(seg1, summary="")
    │   → SubAgent 1 看第 1 段视频 → 分析结果 + 压缩摘要
    │
    ├─ 调用 analyze_video_segment(seg2, summary=summary_1)
    │   → SubAgent 2 看第 2 段视频 + 知道第 1 段讲了什么 → 分析 + 摘要
    │
    ├─ 调用 analyze_video_segment(seg3, summary=summary_1+2)
    │   → SubAgent 3 看第 3 段 + 知道前两段讲了什么 → 分析 + 摘要
    │
    └─ 主 Agent 汇总所有段落 → 输出完整分析报告
```

**为什么需要滚动 summary？**

每个 SubAgent 只能看到自己那段视频，但内容分析需要上下文。比如：
- 第 3 段提到"刚才说的那个方法"，SubAgent 需要知道"那个方法"是什么
- 最后一段做总结时，需要知道前面讲了哪些要点

滚动 summary 解决了这个问题：每个 SubAgent 在分析完成后，自动生成 2-3 句话的压缩摘要，传递给下一个 SubAgent 作为上下文。这样每段分析都能"看到"前面的全局脉络。

**SubAgent 的实现方式：**

不是创建独立的 LangGraph Agent，而是直接调用 LLM（`sub_model.ainvoke(messages)`）。因为 SubAgent 不需要使用工具，只需要理解视频 + 输出分析文字。这大大简化了实现。

## 组件清单

### 服务层

| 服务 | 文件 | 职责 |
|------|------|------|
| VideoTranscoder | `services/video_transcoder.py` | 视频时长检测、720p 转码、按时间点切割 |
| QiniuStorageService | `services/qiniu_storage.py` | 音频/视频上传七牛云，自动过期删除 |

### 工具层

| 工具 | 文件 | 触发条件 |
|------|------|----------|
| `process_video` | `agent/tools/video_processor_tool.py` | 有视频时自动调用，内部判断 omni + 时长分支 |
| `prepare_video_segments` | `agent/tools/video_segment_tool.py` | 长视频场景，Agent 主动调用 |
| `analyze_video_segment` | `agent/tools/video_segment_tool.py` | 逐段分析，Agent 按分段列表依次调用 |

### 中间件层

| 中间件 | 文件 | 说明 |
|--------|------|------|
| `omni_media_injection_middleware` | `agent/middleware/omni_media_injection.py` | 视频/音频 URL 注入到 HumanMessage，视频优先 |

### 状态字段

| 字段 | 类型 | 用途 |
|------|------|------|
| `omni_video_url` | `Optional[str]` | 短视频直传的七牛 URL，注入后自动清除 |
| `omni_audio_url` | `Optional[str]` | 音频直传的七牛 URL，注入后自动清除 |
| `video_segments` | `Optional[List[VideoSegmentDict]]` | 长视频分段信息列表 |

## 配置项

### 七牛云（.env）

```ini
QINIU_ACCESS_KEY=your_access_key
QINIU_SECRET_KEY=your_secret_key
QINIU_BUCKET=your_bucket_name
QINIU_DOMAIN=cdn.example.com
QINIU_DELETE_AFTER_DAYS=1          # 文件自动过期天数
```

### 视频参数（.env）

```ini
OMNI_VIDEO_MAX_DURATION=400        # 直传最大时长（秒），超出则分段
OMNI_VIDEO_SEGMENT_MAX=360         # 分段最大时长（秒），留余量
```

### 模型配置

在前端设置页面创建 LLM 配置时：
1. 勾选"全模态模型"（`support_omni`）
2. 填入全模态模型的 API 配置（如 DashScope 的 Qwen3.5-Omni）
3. 选择该模型后发起分析，即自动走全模态路径

## 使用指南

### 前提条件

1. **七牛云账号**：注册并创建存储空间，获取 AK/SK/Bucket/Domain
2. **全模态模型 API**：如阿里云 DashScope（Qwen3.5-Omni）
3. **ffmpeg**：系统已安装（`brew install ffmpeg` 或 `apt install ffmpeg`）

### 配置步骤

1. 在 `.env` 中配置七牛云参数
2. 运行数据库迁移：`uv run python scripts/run_migrations.py`
3. 在前端"设置 → LLM 配置"中新建配置：
   - Provider: OpenAI / 兼容 API
   - Base URL: `https://dashscope.aliyuncs.com/compatible-mode/v1`
   - Model: `qwen3.5-omni-plus`
   - 勾选"全模态模型"
4. 激活该配置，开始使用

### 使用流程

**短视频：** 用户粘贴链接 → Agent 自动检测为短视频 → 转码上传 → 模型直接分析视频

**长视频：** 用户粘贴链接 → Agent 检测为长视频 → ASR 转录 → Agent 调用分段工具 → 逐段 SubAgent 分析 → 汇总输出

用户无需感知这些差异，Agent 会根据视频时长和模型能力自动选择最优路径。

## 降级策略

| 场景 | 降级行为 |
|------|----------|
| 模型未标记 `support_omni` | 走传统 ASR 流程（完全不变） |
| 七牛云未配置 | 自动降级到 ASR 流程，日志告警 |
| 七牛上传失败 | 自动降级到 ASR 流程 |
| 视频转码失败 | 自动降级到 ASR 流程 |
| 分段 LLM 解析失败 | 抛出错误，提示用户重试 |
| 某段 SubAgent 分析失败 | 该段标记失败，其余段继续 |

## 技术决策记录

### 为什么用七牛云而非 Base64？

- 社交媒体视频通常 10-100MB，Base64 膨胀 33% 后请求体过大
- 七牛 CDN 加速，模型端拉取更快
- 自动过期删除，无需管理临时文件

### 为什么用 Tool 模式而非 SubGraph？

- 与现有 3 个工具（parse_link/fetch_content/process_video）模式一致
- 主 Agent 天然拥有对话上下文，能灵活决定何时调用哪个工具
- 实现简单，不引入 LangGraph SubGraph 概念

### 为什么 SubAgent 直接调用 LLM 而非创建完整 Agent？

- SubAgent 的任务是纯分析（看视频 → 输出文字），不需要调用任何工具
- 创建完整 Agent 需要 checkpointer、tools、middleware 等额外开销
- 直接 `model.ainvoke(messages)` 最简洁高效

### 为什么分段上限设为 360s 而非 400s？

- Qwen-Omni 的视频输入上限是 400s
- 留 40s 余量应对 ffmpeg 切割精度误差和边界效应
- 实际测试中，360s 以内的视频处理稳定性最好
