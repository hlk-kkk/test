# 文本爬虫详细指南

## 项目路径

```
$MCR_ROOT/MediaCrawlerPro-Python
```

其中 `$MCR_ROOT` 在 SKILL.md 第零步中确定。

## 四种爬虫模式

### 1. 搜索模式 (search)

按关键词搜索平台内容，关键词通过 `--keywords` 参数传入，无需修改代码。

```bash
DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run --project "$MCR_ROOT/MediaCrawlerPro-Python" \
  "$MCR_ROOT/MediaCrawlerPro-Python/main.py" \
  --platform xhs --type search --keywords "AI绘画,Stable Diffusion"
```

相关配置（`config/base_config.py`）：
- `KEYWORDS` - 默认关键词（CLI `--keywords` 会覆盖）
- `CRAWLER_MAX_NOTES_COUNT` - 最大爬取数量（默认 40）
- `SORT_TYPE` - 排序方式（暂只支持小红书，如 `popularity_descending`）
- `PUBLISH_TIME_TYPE` - 发布时间过滤（暂只支持抖音）

### 2. 详情模式 (detail)

爬取指定帖子/笔记的详情和评论。通过 `--urls` 参数传入 URL 或 ID，多个用逗号分隔。

```bash
# 小红书（传入完整URL，需携带 xsec_token）
DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run --project "$MCR_ROOT/MediaCrawlerPro-Python" \
  "$MCR_ROOT/MediaCrawlerPro-Python/main.py" \
  --platform xhs --type detail \
  --urls "https://www.xiaohongshu.com/explore/xxx?xsec_token=xxx&xsec_source=pc_user"

# 抖音（传入 aweme_id）
--platform dy --type detail --urls "7566756334578830627,7525538910311632128"

# B站（传入 bvid）
--platform bili --type detail --urls "BV1d54y1g7db"

# 知乎（传入完整URL）
--platform zhihu --type detail --urls "https://www.zhihu.com/question/826896610/answer/4885821440"
```

### 3. 创作者模式 (creator)

爬取创作者主页信息和作品列表。通过 `--urls` 参数传入创作者 URL 或 ID。

```bash
# 小红书（传入主页URL，需携带 xsec_token）
--platform xhs --type creator --urls "https://www.xiaohongshu.com/user/profile/xxx?xsec_token=xxx"

# 抖音（传入 sec_uid）
--platform dy --type creator --urls "MS4wLjABAAAATJPY7LAlaa5X-c8uNdWkvz0jUGgpw4eeXIwu_8BhvqE"
```

各平台 URL/ID 格式见 [platforms.md](platforms.md)

### 4. 首页推荐 (homefeed)

爬取平台首页推荐内容，无需修改代码。

```bash
DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run --project "$MCR_ROOT/MediaCrawlerPro-Python" \
  "$MCR_ROOT/MediaCrawlerPro-Python/main.py" \
  --platform xhs --type homefeed
```

## 公共配置项

在 `config/base_config.py` 中：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `ENABLE_GET_COMMENTS` | `True` | 是否爬取评论 |
| `ENABLE_GET_SUB_COMMENTS` | `False` | 是否爬取二级评论 |
| `PER_NOTE_MAX_COMMENTS_COUNT` | `0` | 每篇帖子最大评论数（0=不限制） |
| `MAX_CONCURRENCY_NUM` | `1` | 并发数 |
| `CRAWLER_TIME_SLEEP` | `1` | 请求间隔（秒） |
| `CRAWLER_MAX_NOTES_COUNT` | `40` | 搜索最大帖子数 |
| `ENABLE_CHECKPOINT` | `True` | 断点续爬 |

## 环境变量

### Cookie 来源（三选一）

```bash
# 方式1：CookieBridge 自动获取（推荐，需 CookieBridge Server 运行）
ACCOUNT_POOL_SAVE_TYPE=cookie_bridge

# 方式2：从 Excel 读取
ACCOUNT_POOL_SAVE_TYPE=xlsx

# 方式3：从 MySQL 读取
ACCOUNT_POOL_SAVE_TYPE=mysql
```

### 数据库类型

```bash
DB_TYPE=sqlite   # 使用 SQLite（推荐，无需额外数据库）
DB_TYPE=mysql    # 使用 MySQL
```

### CookieBridge 配置

```bash
COOKIE_BRIDGE_URL=http://localhost:8274   # CookieBridge 服务地址（默认值）
```

**注意**：shell 环境变量会覆盖 `config/base_config.py` 中的默认值。如果遇到配置不生效的问题，先检查 `echo $ACCOUNT_POOL_SAVE_TYPE`。

## Cookie 写入 Excel

如果用户提供了 cookies 字符串且不使用 CookieBridge，需要写入 Excel 供爬虫使用：

```python
import openpyxl
wb = openpyxl.load_workbook('$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx')
ws = wb['xhs']  # sheet 名 = 平台 ID (xhs/dy/ks/bili/wb/tieba/zhihu)
ws['A2'] = 1                    # id
ws['B2'] = 'account_1'         # account_name
ws['C2'] = 'cookies_string'    # cookies
ws['D2'] = 'xhs'               # platform_name
ws['E2'] = 'normal'            # status
wb.save('$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx')
```

## SQLite 查询结果

数据库路径: `$MCR_ROOT/MediaCrawlerPro-Python/media_crawler.db`

### 常用查询示例

```sql
-- 搜索结果：查最新的小红书笔记
SELECT note_id, title, desc, liked_count, collected_count, comment_count, source_keyword
FROM xhs_note ORDER BY add_ts DESC LIMIT 20;

-- 搜索结果：按关键词筛选
SELECT note_id, title, liked_count FROM xhs_note
WHERE source_keyword = 'AI绘画' ORDER BY add_ts DESC;

-- 查看抖音视频
SELECT aweme_id, title, liked_count, comment_count, share_count, aweme_url
FROM douyin_aweme ORDER BY create_time DESC LIMIT 20;

-- 查看评论
SELECT content, nickname, like_count FROM xhs_note_comment
WHERE note_id = 'xxx' ORDER BY create_time DESC;

-- 查看创作者信息
SELECT user_id, nickname, fans, follows, interaction FROM xhs_creator;

-- 查看知乎内容
SELECT content_id, content_type, title, voteup_count, comment_count, content_url
FROM zhihu_content ORDER BY created_time DESC LIMIT 20;
```

表结构详情见 [sqlite-tables.md](sqlite-tables.md)

## 注意事项

- 小红书 detail 模式需要带 `xsec_token` 的 URL（有时效性，需从浏览器地址栏获取）
- 小红书 creator 模式同样需要带 `xsec_token` 的主页 URL
- 知乎 detail 模式支持 4 种 URL 格式：回答、文章、视频、问题页
- 爬虫是批量运行模式，运行结束后查询 SQLite 获取结果
- 贴吧还支持通过 `TIEBA_NAME_LIST` 指定贴吧名称来爬取
