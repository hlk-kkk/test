# PicTacticAgent

[English](README_en.md) | **中文**

AI 图片生成工具 — 多轮生成 + 自动评估 + 迭代优化，告别文生图"抽卡"。内置 **199 条精选提示词模板**，支持作为 Skill 集成到 Claude Code / OpenClaw 等 AI Agent 中。

<table>
  <tr>
    <td><img src="docs/screenshots/workspace-with-history.png" alt="工作台" /></td>
    <td><img src="docs/screenshots/template-library.png" alt="模板库" /></td>
  </tr>
  <tr>
    <td align="center"><b>工作台</b> — 生成 + 评分 + 历史</td>
    <td align="center"><b>模板库</b> — 199 条精选提示词，一键生成</td>
  </tr>
</table>

## 亮点

- **199 条精选模板** — 从 2,000+ 条 X/Twitter 提示词中通过 Vision LLM 锦标赛筛选，覆盖电影叙事、时尚编辑、人像、产品、艺术风格、超现实等 6 大类
- **多轮迭代生成** — 每轮生成多张候选，AI 六维度评分，自动迭代至满意为止
- **Skill 集成** — 一条命令变身 Claude Code / OpenClaw 的图片生成能力，对 AI 说"帮我画张图"即可
- **多种使用方式** — Web 界面 / CLI 命令行 / REST API，灵活接入
- **多 Provider** — Gemini Web（免费 Cookie 认证）、Gemini、Jiekouai、Antigravity，一键切换

## 快速开始

```bash
# 1. 克隆
git clone https://github.com/NanmiCoder/PicTacticAgent.git
cd PicTacticAgent

# 2. 配置 API Key（编辑 .env，选一个 Provider 填入 Key 即可）
cp .env.example .env

# 3. 一键启动
./start.sh
```

启动后访问 http://localhost:3019 开始使用。

> Provider 推荐：[接口AI](https://jiekou.ai/referral?invited_code=3CF8T0) 国内中转站，一个 Key 搞定生图 + LLM 评估，注册绑定 GitHub 送 3 刀。

## Skill 集成

PicTacticAgent 可以作为 **Skill** 集成到 AI Agent 中，让 Agent 具备图片生成和编辑能力。

### Claude Code

让 Claude Code 阅读安装文档，自动完成配置：

```
@docs/skill-installation.md 按照文档帮我安装 PicTacticAgent Skill
```

安装后直接说「帮我生成一张赛博朋克风格的封面图」就能用。

### OpenClaw

项目根目录已提供 `SKILL.md`，支持 OpenClaw 技能发现：

```bash
# 在 OpenClaw 中安装
clawhub install pictactic

# 或手动复制 SKILL.md 到你的 OpenClaw 工作区
```

安装后通过任意消息渠道（Telegram / Slack / Discord 等）对 OpenClaw 说「generate an image of ...」即可触发。

### 其他 Agent

任何支持 CLI 调用的 Agent 都可以通过 `uv run pictactic` 命令集成：

```bash
uv run pictactic generate "一张科技感十足的封面图"
uv run pictactic edit --source ./image.png "将背景改为深蓝色"
```

## CLI 快速参考

```bash
# 文生图
uv run pictactic generate "prompt here"

# 带参数
uv run pictactic generate --rounds 3 --images 5 --provider gemini "prompt"

# 编辑图片
uv run pictactic edit --source ./img.png "修改指令"

# 异步模式（推荐 Agent 使用）
uv run pictactic submit generate --format quiet "prompt"
uv run pictactic status <task_id>
uv run pictactic result <task_id> --output-dir ./output

# 查看 Provider
uv run pictactic providers
```

> 注意：prompt 参数必须放在所有选项之后。

## 技术栈

FastAPI + LangGraph (StateGraph) | React 19 + TailwindCSS 4 | SQLite | Gemini Web / Gemini / Jiekouai / Antigravity | Typer CLI

## 文档

| 文档 | 说明 |
|------|------|
| [配置指南](docs/CONFIGURATION.md) | Provider 选择、API Key 配置 |
| [部署指南](docs/DEPLOYMENT.md) | Docker / Linux 生产部署 |
| [Skill 安装](docs/skill-installation.md) | Claude Code Skill 详细安装步骤 |
| [技术设计](docs/TECHNICAL_DESIGN.md) | 架构设计、工作流、数据模型 |
| [API 文档](http://localhost:8019/docs) | 启动后可访问的交互式 API 文档 |

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request!
