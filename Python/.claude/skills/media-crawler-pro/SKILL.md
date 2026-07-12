---
name: media-crawler-pro
description: "自媒体平台数据爬取与分析工具。支持爬取/搜索/查找/抓取/监控/分析小红书(红薯/xhs)、抖音(dy/TikTok)、快手(ks)、B站(哔哩哔哩/bili)、微博(wb)、贴吧(tieba)、知乎(zhihu)的笔记、帖子、视频、评论、创作者/博主/UP主信息。当用户需要搜索平台内容、获取帖子详情、查看博主主页、分析评论舆情、下载视频图片、竞品分析时激活。"
---

# Media Crawler Pro Skill

多平台自媒体爬虫系统，由 4 个子项目组成：签名服务 (SignSrv)、Python 爬虫 (文本数据)、下载服务 (视频/图片)、CookieBridge (Cookie 自动同步)。

## 渐进式加载

本 Skill 仅包含路由决策和快速命令模板。详细文档按需从 references/ 加载，**不要一次性全部读取**。

| 场景 | 读取 |
|------|------|
| 首次运行 / 环境安装 / 服务启动 | `references/setup.md` |
| CookieBridge 配置 / API / 故障排查 | `references/cookie-bridge.md` |
| Cookie 准备 / 有效性验证 / 故障排查 | `references/cookie-preflight.md` |
| 平台 URL/ID 格式确认 | `references/platforms.md` |
| DownloadServer API 调用详情 | `references/downloader.md` |
| 文本爬虫 CLI 详细用法 / 配置项 | `references/text-crawler.md` |
| SQLite 完整表结构 / 字段类型 | `references/sqlite-tables.md` |
| 评论舆情分析流程 | `references/sentiment-analysis.md` |

## 常见意图识别

| 用户说的话 | 对应操作 | 命令模式 |
|-----------|---------|---------|
| "搜一下/查一下/找一下 XX 平台关于 YY 的内容" | 关键词搜索 | CLI `--type search` |
| "看看这个博主/UP主最近发了什么" | 创作者作品列表 | CLI `--type creator` |
| "帮我分析这条帖子/视频的评论" | 详情+评论 | CLI `--type detail` → 查询评论 |
| "这个视频的下载链接是什么" | 内容详情 | DownloadServer API |
| "下载这个视频/图片" | 下载文件 | DownloadServer API + curl |
| "看看首页推荐了什么" | 首页推荐 | CLI `--type homefeed` |
| "帮我做舆情分析/情感分析" | 详情+评论+分析 | → Read `references/sentiment-analysis.md` |

## URL 自动路由

当用户提供 URL 时，按域名和路径自动推断 `--platform` 和 `--type`：

| URL 域名 | platform | URL 路径模式 | type | 传给 `--urls` 的值 |
|----------|----------|-------------|------|-------------------|
| xiaohongshu.com | xhs | `/explore/{id}?xsec_token=...` | detail | 完整 URL |
| xiaohongshu.com | xhs | `/user/profile/{id}?xsec_token=...` | creator | 完整 URL |
| douyin.com | dy | `/video/{aweme_id}` | detail | aweme_id |
| douyin.com | dy | `/user/{sec_uid}` | creator | sec_uid |
| v.douyin.com | dy | `/xxxxx/` | detail | 短链接需先解析 |
| kuaishou.com | ks | `/short-video/{id}` | detail | video_id |
| kuaishou.com | ks | `/profile/{id}` | creator | user_id |
| bilibili.com | bili | `/video/{bvid}` | detail | bvid |
| space.bilibili.com | bili | `/{user_id}` | creator | user_id |
| weibo.com | wb | `/{uid}/{note_id}` | detail | note_id |
| weibo.com | wb | `/u/{uid}` | creator | user_id |
| tieba.baidu.com | tieba | `/p/{note_id}` | detail | note_id |
| zhihu.com | zhihu | `/question/*/answer/*` | detail | 完整 URL |
| zhihu.com | zhihu | `/people/{token}` | creator | 完整 URL |

**小红书注意**: `xsec_token` 有时效性。如果 URL 不含 `xsec_token`，提示用户在浏览器打开该页面，从地址栏复制完整 URL。

## 服务依赖矩阵

根据操作场景只检查必要的服务，避免不必要的 tool calls：

| 操作 | SignSrv (必需) | CookieBridge (推荐) | DownloadServer |
|-----|---------------|-------------------|---------------|
| CLI: search / homefeed | 是 | 是 | **否** |
| CLI: detail / creator | 是 | 是 | **否** |
| DownloadServer API | 是 | 是 | **是** |

一行检查所有服务状态：
```bash
echo "SignSrv: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8989/signsrv/pong 2>/dev/null || echo 'down'), CookieBridge: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8274/ping 2>/dev/null || echo 'down'), DownloadServer: $(curl -s -o /dev/null -w '%{http_code}' http://localhost:8205/ping 2>/dev/null || echo 'down')"
```

> 首次运行或服务未启动时，读取 `references/setup.md` 获取安装和启动指引。

## Cookie 前置检查

**所有爬虫操作都需要有效的平台登录 Cookie。** 没有 Cookie 会报错："账号池中没有可用的账号"。执行爬虫前，先检查 Cookie 是否可用：

```bash
# CookieBridge 模式：检查目标平台 Cookie
curl -s http://localhost:8274/api/cookies/{platform} | python3 -c "
import sys,json; d=json.load(sys.stdin)
if d.get('isok') and d.get('data',{}).get('cookies'):
    print(f'OK: {d[\"data\"][\"platform\"]} Cookie 可用 (来源: {d[\"data\"].get(\"source\",\"unknown\")})')
else:
    print('FAIL: Cookie 不可用，请在浏览器登录该平台')
"

# Excel 模式：检查 Excel 中是否有该平台的 Cookie
python3 -c "
import openpyxl; wb=openpyxl.load_workbook('$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx')
p='{platform}'
ws=wb[p] if p in wb.sheetnames else None
print('OK' if ws and any(ws.cell(r,3).value for r in range(2,ws.max_row+1)) else 'FAIL: Cookie 缺失')
"

# 或使用 preflight 脚本一键检查（含服务 + Cookie）
bash "$MCR_ROOT/MediaCrawlerPro-Python/scripts/preflight.sh" {platform}
```

> Cookie 不可用？完整准备指南（方案选择、平台特殊要求、故障排查）见 `references/cookie-preflight.md`。

## Cookie 决策流程

```
CookieBridge 在线？ (curl -s http://localhost:8274/ping)
  ├─ 是 → ACCOUNT_POOL_SAVE_TYPE=cookie_bridge
  └─ 否 → Excel 有该平台 cookies？
              ├─ 是 → ACCOUNT_POOL_SAVE_TYPE=xlsx
              └─ 否 → 要求用户提供 cookies 字符串，写入 Excel
```

DownloadServer API 需在每个请求体中传入 `cookies` 字段，可从 CookieBridge 获取：
```bash
COOKIES=$(curl -s http://localhost:8274/api/cookies/{platform} | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['cookies'])")
```

> Cookie 准备完整指南见 `references/cookie-preflight.md`。CookieBridge API 详情见 `references/cookie-bridge.md`，Excel 写入方法见 `references/text-crawler.md`。

## CLI 参数与环境变量

完整参数列表通过 `uv run main.py --help` 查看。关键环境变量（放在命令前面）：

| 环境变量 | 说明 | 常用值 |
|----------|------|--------|
| `DB_TYPE` | 数据库类型 | `sqlite`（推荐） / `mysql` |
| `ACCOUNT_POOL_SAVE_TYPE` | Cookie 来源 | `cookie_bridge`（推荐） / `xlsx` / `mysql` |

### 评论爬取控制

默认开启评论爬取。**不涉及评论的任务务必加 `--no-enable_comments` 加速**：

| 场景 | 需要评论？ |
|------|-----------|
| search / homefeed / creator（仅看列表） | 否 → 加 `--no-enable_comments` |
| detail（看帖子+评论）/ 舆情分析 | 是 → 默认即可 |

## 快速命令模板

### 定位项目根目录

```bash
# 自动检测：当前目录 → 上级目录 → 常见安装位置
if [ -d "./MediaCrawlerPro-Python" ]; then MCR_ROOT="$(pwd)"
elif [ -d "../MediaCrawlerPro-Python" ]; then MCR_ROOT="$(cd .. && pwd)"
elif [ -d "$HOME/MediaCrawlerPro/MediaCrawlerPro-Python" ]; then MCR_ROOT="$HOME/MediaCrawlerPro"
elif [ -d "$HOME/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-Python" ]; then MCR_ROOT="$HOME/workspace/myself_code/MediaCrawlerPro"
else echo "ERROR: 无法自动检测 MCR_ROOT，请手动设置: export MCR_ROOT=/path/to/MediaCrawlerPro"; fi
echo "MCR_ROOT=$MCR_ROOT"
```

> 如果自动检测失败，手动设置：`export MCR_ROOT=/你的路径/MediaCrawlerPro`
> 安装指引见 `docs/9.Skill安装文档.md`，所有子项目应位于同一父目录下。

### 文本爬虫 CLI

**重要：必须先 `cd` 进项目目录再执行，否则相对路径会出错。**

```bash
cd "$MCR_ROOT/MediaCrawlerPro-Python"

# 搜索（不需要评论时加 --no-enable_comments 加速）
DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run main.py \
  --platform {platform} --type search --keywords "关键词1,关键词2" --no-enable_comments

# 详情（默认带评论）
DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run main.py \
  --platform {platform} --type detail --urls "url_or_id1,url_or_id2"

# 创作者（不需要评论时加 --no-enable_comments）
DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run main.py \
  --platform {platform} --type creator --urls "creator_url_or_id" --no-enable_comments

# 首页推荐
DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run main.py \
  --platform {platform} --type homefeed --no-enable_comments
```

> 如果 CookieBridge 不可用，将 `cookie_bridge` 换为 `xlsx`。
> 各平台 `--urls` 参数格式见 `references/platforms.md`。

### DownloadServer API

```bash
# 获取内容详情（含视频下载URL、封面图、互动数据）
curl -s -X POST http://localhost:8205/api/v1/content_detail \
  -H 'Content-Type: application/json' \
  -d '{"platform": "{platform}", "content_url": "URL", "cookies": "COOKIES"}'

# 下载视频/图片
curl -L -o output.mp4 "$VIDEO_DOWNLOAD_URL"
# B站视频需要 Referer
curl -L -H "Referer: https://www.bilibili.com/" -o video.mp4 "$BILI_URL"
```

> 完整 API 文档（creator_query、creator_contents 等）见 `references/downloader.md`。

### SQLite 查询

```bash
# 在项目目录下执行（cd "$MCR_ROOT/MediaCrawlerPro-Python" 后）
DB_PATH="./media_crawler.db"

# 查看最近爬取的内容
sqlite3 -header -column "$DB_PATH" "SELECT * FROM {content_table} ORDER BY add_ts DESC LIMIT 20;"

# 查看评论（按点赞排序）
sqlite3 -header -column "$DB_PATH" "SELECT content, nickname, like_count FROM {comment_table} WHERE {id_field}='{content_id}' ORDER BY CAST(like_count AS INTEGER) DESC LIMIT 50;"
```

> **注意**: 数值排序需 `CAST(xxx AS INTEGER)`。tieba/zhihu 评论表昵称字段为 `user_nickname`，时间字段为 `publish_time`。
> 完整表结构见 `references/sqlite-tables.md`。

## 平台-表名速查

| platform | 内容表 | 内容ID字段 | 评论表 | 评论关联字段 | 创作者表 |
|----------|--------|-----------|--------|------------|---------|
| xhs | `xhs_note` | `note_id` | `xhs_note_comment` | `note_id` | `xhs_creator` |
| dy | `douyin_aweme` | `aweme_id` | `douyin_aweme_comment` | `aweme_id` | `dy_creator` |
| ks | `kuaishou_video` | `video_id` | `kuaishou_video_comment` | `video_id` | `kuaishou_creator` |
| bili | `bilibili_video` | `video_id` | `bilibili_video_comment` | `video_id` | `bilibili_up_info` |
| wb | `weibo_note` | `note_id` | `weibo_note_comment` | `note_id` | `weibo_creator` |
| tieba | `tieba_note` | `note_id` | `tieba_comment` | `note_id` | `tieba_creator` |
| zhihu | `zhihu_content` | `content_id` | `zhihu_comment` | `content_id` | `zhihu_creator` |

## 任务决策流程

| 用户需求 | 使用工具 | 需改代码 |
|---------|---------|---------|
| 获取单条内容详情/视频URL/封面图 | DownloadServer API | 否 |
| 下载视频/图片 | DownloadServer API + curl | 否 |
| 获取创作者信息 | DownloadServer API | 否 |
| 关键词搜索 | Python CLI `--type search` | 否 |
| 首页推荐 | Python CLI `--type homefeed` | 否 |
| 爬取评论 | Python CLI `--type detail --urls "..."` | 否 |
| 爬取创作者作品列表+评论 | Python CLI `--type creator --urls "..."` | 否 |

## 端口速查

| 服务 | 端口 | 健康检查 |
|------|------|---------|
| SignSrv | 8989 | `curl http://localhost:8989/signsrv/pong` |
| CookieBridge | 8274 | `curl http://localhost:8274/ping` |
| DownloadServer | 8205 | `curl http://localhost:8205/ping` |

## 注意事项

- 仅供学习和研究目的使用，不得用于商业用途
- 遵守目标平台的使用条款和 robots.txt 规则
- 控制爬取频率（`MAX_CONCURRENCY_NUM=1`, `CRAWLER_TIME_SLEEP=1`）
- 不得进行大规模爬取或对平台造成运营干扰
