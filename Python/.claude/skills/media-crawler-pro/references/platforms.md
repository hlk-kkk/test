# 平台 & URL 模式参考

## 平台 ID 映射

| 平台 | Python CLI `--platform` | DownloadServer `platform` | 支持的爬虫类型 |
|------|------------------------|---------------------------|---------------|
| 小红书 | `xhs` | `xhs` | search, detail, creator, homefeed |
| 抖音 | `dy` | `dy` | search, detail, creator, homefeed |
| 快手 | `ks` | `ks` | search, detail, creator, homefeed |
| B站 | `bili` | `bili` | search, detail, creator, homefeed |
| 微博 | `wb` | 不支持 | search, detail, creator |
| 贴吧 | `tieba` | 不支持 | search, detail, creator |
| 知乎 | `zhihu` | 不支持 | search, detail, creator |

## URL 格式参考

### 小红书

```
# 笔记（需携带 xsec_token，有时效）
https://www.xiaohongshu.com/explore/{note_id}?xsec_token=xxx&xsec_source=pc_feed

# 创作者主页（需携带 xsec_token）
https://www.xiaohongshu.com/user/profile/{user_id}?xsec_token=xxx&xsec_source=pc_search
```

**重要**: 小红书的 `xsec_token` 有时效性，需要在浏览器打开对应页面，从地址栏复制完整 URL。旧的 `XHS_SPECIFIED_ID_LIST` 和 `XHS_CREATOR_ID_LIST` 已废弃。

### 抖音

```
# 视频详情
https://www.douyin.com/video/{aweme_id}

# 短链接（也支持）
https://v.douyin.com/xxxxx/

# 创作者主页（使用 sec_uid）
https://www.douyin.com/user/{sec_uid}
```

detail 模式使用 `DY_SPECIFIED_ID_LIST`（aweme_id），creator 模式使用 `DY_CREATOR_ID_LIST`（sec_uid）。

### 快手

```
# 视频详情
https://www.kuaishou.com/short-video/{video_id}

# 创作者主页
https://www.kuaishou.com/profile/{user_id}
```

### B站

```
# 视频详情（使用 bvid）
https://www.bilibili.com/video/{bvid}

# UP主主页（使用数字ID）
https://space.bilibili.com/{user_id}
```

**下载注意**: B站视频下载需要 `Referer: https://www.bilibili.com/` 请求头。

### 微博

```
# 帖子详情（使用帖子ID）
https://weibo.com/{user_id}/{note_id}

# 创作者主页（使用用户ID）
https://weibo.com/u/{user_id}
```

### 贴吧

```
# 帖子详情
https://tieba.baidu.com/p/{note_id}

# 创作者主页
https://tieba.baidu.com/home/main/?id={user_id}&fr=frs
```

贴吧还支持通过 `TIEBA_NAME_LIST` 配置项指定贴吧名称来爬取。

### 知乎

```
# 回答
https://www.zhihu.com/question/{question_id}/answer/{answer_id}

# 文章
https://zhuanlan.zhihu.com/p/{article_id}

# 视频
https://www.zhihu.com/zvideo/{video_id}

# 问题（爬取该问题下的答案列表）
https://www.zhihu.com/question/{question_id}

# 创作者主页
https://www.zhihu.com/people/{url_token}
```

## 配置项速查

### detail 模式：指定内容 ID/URL

| 平台 | 配置变量 | 值类型 |
|------|---------|--------|
| 小红书 | `XHS_SPECIFIED_NOTE_URL_LIST` | URL 列表（带 xsec_token） |
| 抖音 | `DY_SPECIFIED_ID_LIST` | aweme_id 列表 |
| 快手 | `KS_SPECIFIED_ID_LIST` | video_id 列表 |
| B站 | `BILI_SPECIFIED_ID_LIST` | bvid 列表 |
| 微博 | `WEIBO_SPECIFIED_ID_LIST` | note_id 列表 |
| 贴吧 | `TIEBA_SPECIFIED_ID_LIST` | note_id 列表 |
| 知乎 | `ZHIHU_SPECIFIED_ID_LIST` | URL 列表 |

### creator 模式：指定创作者 ID/URL

| 平台 | 配置变量 | 值类型 |
|------|---------|--------|
| 小红书 | `XHS_CREATOR_URL_LIST` | URL 列表（带 xsec_token） |
| 抖音 | `DY_CREATOR_ID_LIST` | sec_uid 列表 |
| 快手 | `KS_CREATOR_ID_LIST` | user_id 列表 |
| B站 | `BILI_CREATOR_ID_LIST` | 数字 user_id 列表 |
| 微博 | `WEIBO_CREATOR_ID_LIST` | user_id 列表 |
| 贴吧 | `TIEBA_CREATOR_URL_LIST` | URL 列表 |
| 知乎 | `ZHIHU_CREATOR_URL_LIST` | URL 列表 |

### 配置文件路径

```
$MCR_ROOT/MediaCrawlerPro-Python/config/base_config.py     # 主配置（平台、关键词、ID列表等）
$MCR_ROOT/MediaCrawlerPro-Python/config/db_config.py       # 数据库配置（DB_TYPE、SQLITE_DB_PATH）
$MCR_ROOT/MediaCrawlerPro-Python/config/sign_srv_config.py # 签名服务配置（SIGN_SRV_HOST、SIGN_SRV_PORT）
```
