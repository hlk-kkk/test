# 下载器详细指南 (DownloadServer)

## 项目路径

```
$MCR_ROOT/MediaCrawlerPro-Downloader/DownloadServer
```

其中 `$MCR_ROOT` 在 SKILL.md 第零步中确定。

## 启动服务

```bash
cd "$MCR_ROOT/MediaCrawlerPro-Downloader/DownloadServer"
source .venv/bin/activate
python app.py &

# 验证服务启动
curl -s http://localhost:8205/ping
# 返回: {"biz_code":0,"msg":"OK!","isok":true,"data":{"message":"pong"}}
```

默认端口 `8205`，依赖 SignSrv（端口 `8989`）。

## 支持平台

DownloadServer 支持 4 个平台：`xhs`（小红书）、`dy`（抖音）、`ks`（快手）、`bili`（B站）。

## API 接口

### 1. 获取内容详情 `POST /api/v1/content_detail`

获取单个内容的详细信息，包含视频下载地址、封面图、图文列表等。

**请求：**

```json
{
  "platform": "dy",
  "content_url": "https://www.douyin.com/video/7566756334578830627",
  "cookies": "你的cookies字符串"
}
```

- `platform`: 枚举值 `xhs` | `dy` | `ks` | `bili`
- `content_url`: 内容页面 URL
- `cookies`: 对应平台的登录 cookies

**响应：**

```json
{
  "biz_code": 0,
  "msg": "OK!",
  "isok": true,
  "data": {
    "content": {
      "id": "内容ID",
      "url": "内容页面URL",
      "title": "标题",
      "desc": "描述/正文",
      "content_type": "video 或 note",
      "cover_url": "封面图URL",
      "image_urls": ["图片URL1", "图片URL2"],
      "video_download_url": "视频下载直链",
      "author": {
        "user_id": "作者ID",
        "sec_uid": "加密用户ID（抖音）",
        "nickname": "昵称",
        "avatar": "头像URL",
        "profile_url": "主页链接"
      },
      "interaction": {
        "liked_count": "点赞数",
        "collected_count": "收藏数",
        "comment_count": "评论数",
        "share_count": "分享数",
        "play_count": "播放数",
        "danmaku_count": "弹幕数"
      }
    }
  }
}
```

### 2. 查询创作者信息 `POST /api/v1/creator_query`

**请求：**

```json
{
  "platform": "dy",
  "creator_url": "https://www.douyin.com/user/MS4wLjABAAAATJPY7LAlaa5X-c8uNdWkvz0jUGgpw4eeXIwu_8BhvqE",
  "cookies": "你的cookies字符串"
}
```

**响应：**

```json
{
  "biz_code": 0,
  "msg": "OK!",
  "isok": true,
  "data": {
    "nickname": "昵称",
    "avatar": "头像URL",
    "description": "个人简介",
    "user_id": "用户ID",
    "follower_count": "粉丝数",
    "following_count": "关注数",
    "content_count": "作品数"
  }
}
```

### 3. 查询创作者内容列表 `POST /api/v1/creator_contents`

**请求：**

```json
{
  "platform": "dy",
  "creator_id": "创作者ID",
  "cursor": "",
  "cookies": "你的cookies字符串"
}
```

- `creator_id`: 从 `creator_query` 接口获取的 `user_id`
- `cursor`: 分页游标，首次为空，后续传入上一次响应的 `next_cursor`

**响应：**

```json
{
  "biz_code": 0,
  "msg": "OK!",
  "isok": true,
  "data": {
    "contents": [
      {
        "id": "内容ID",
        "url": "内容URL",
        "title": "标题",
        "desc": "描述",
        "content_type": "video",
        "cover_url": "封面URL",
        "image_urls": [],
        "video_download_url": "下载URL"
      }
    ],
    "has_more": true,
    "next_cursor": "下一页游标"
  }
}
```

## 下载文件

从 API 拿到 `video_download_url` / `cover_url` / `image_urls` 后，用 curl 下载：

```bash
# 下载视频
curl -L -o output.mp4 "VIDEO_DOWNLOAD_URL"

# 下载封面图
curl -L -o cover.jpg "COVER_URL"

# 下载笔记图片
curl -L -o image1.jpg "IMAGE_URL"

# B站视频需要 Referer 头
curl -L -H "Referer: https://www.bilibili.com/" -o video.mp4 "BILI_VIDEO_URL"
```

## Cookie 获取方式

### 方式1：用户直接提供

用户提供 cookies 字符串，直接传入 API 的 `cookies` 字段。

### 方式2：从 Excel 文件读取

Python 爬虫的 cookies 存储在 Excel 文件中，可以用脚本读取后传给 DownloadServer：

```bash
python3 -c "
import openpyxl
wb = openpyxl.load_workbook('$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx')
for ws in wb.worksheets:
    print(f'=== Sheet: {ws.title} ===')
    for row in ws.iter_rows(min_row=2, values_only=True):
        if row[2]:  # cookies column
            print(f'Platform: {row[3]}, Cookies: {str(row[2])[:80]}...')
"
```

Excel 文件路径: `$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx`

## 典型工作流

### 获取内容详情并下载视频

```bash
# 1. 获取视频详情
RESP=$(curl -s -X POST http://localhost:8205/api/v1/content_detail \
  -H 'Content-Type: application/json' \
  -d '{"platform": "dy", "content_url": "https://www.douyin.com/video/xxx", "cookies": "YOUR_COOKIES"}')

# 2. 提取下载URL
VIDEO_URL=$(echo $RESP | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['content']['video_download_url'])")

# 3. 下载视频
curl -L -o video.mp4 "$VIDEO_URL"
```

### 批量下载创作者所有视频

```bash
# 1. 查询创作者信息
curl -s -X POST http://localhost:8205/api/v1/creator_query \
  -H 'Content-Type: application/json' \
  -d '{"platform": "dy", "creator_url": "https://...", "cookies": "..."}'

# 2. 遍历内容列表（分页）
curl -s -X POST http://localhost:8205/api/v1/creator_contents \
  -H 'Content-Type: application/json' \
  -d '{"platform": "dy", "creator_id": "用户ID", "cursor": "", "cookies": "..."}'

# 3. 对每个内容获取详情并下载
# （循环处理 contents 列表中的 video_download_url）
```
