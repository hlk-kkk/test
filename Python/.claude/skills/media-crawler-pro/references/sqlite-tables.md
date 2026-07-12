# SQLite 表结构参考

数据库路径: `/Users/nanmi/workspace/myself_code/MediaCrawlerPro/MediaCrawlerPro-Python/media_crawler.db`

表结构源文件: `MediaCrawlerPro-Python/schema/tables.sql`

## 表总览

| 平台 | 内容表 | 评论表 | 创作者表 |
|------|--------|--------|----------|
| 小红书 | `xhs_note` | `xhs_note_comment` | `xhs_creator` |
| 抖音 | `douyin_aweme` | `douyin_aweme_comment` | `dy_creator` |
| 快手 | `kuaishou_video` | `kuaishou_video_comment` | `kuaishou_creator` |
| B站 | `bilibili_video` | `bilibili_video_comment` | `bilibili_up_info` |
| 微博 | `weibo_note` | `weibo_note_comment` | `weibo_creator` |
| 贴吧 | `tieba_note` | `tieba_comment` | `tieba_creator` |
| 知乎 | `zhihu_content` | `zhihu_comment` | `zhihu_creator` |

## 小红书 (xhs)

### xhs_note

| 字段 | 类型 | 说明 |
|------|------|------|
| note_id | varchar(64) | 笔记ID |
| title | varchar(255) | 标题 |
| desc | longtext | 描述/正文 |
| type | varchar(16) | 类型 (normal/video) |
| video_url | longtext | 视频地址 |
| time | bigint | 发布时间戳 |
| liked_count | varchar(16) | 点赞数 |
| collected_count | varchar(16) | 收藏数 |
| comment_count | varchar(16) | 评论数 |
| share_count | varchar(16) | 分享数 |
| image_list | longtext | 封面图片列表 |
| tag_list | longtext | 标签列表 |
| note_url | varchar(255) | 详情页URL |
| source_keyword | varchar(255) | 搜索来源关键字 |
| user_id | varchar(64) | 用户ID |
| nickname | varchar(64) | 昵称 |

### xhs_note_comment

| 字段 | 类型 | 说明 |
|------|------|------|
| comment_id | varchar(64) | 评论ID |
| note_id | varchar(64) | 笔记ID |
| content | longtext | 评论内容 |
| create_time | bigint | 评论时间戳 |
| sub_comment_count | varchar(64) | 子评论数 |
| parent_comment_id | varchar(64) | 父评论ID |
| like_count | varchar(255) | 点赞数 |
| nickname | varchar(64) | 昵称 |

### xhs_creator

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | varchar(64) | 用户ID |
| nickname | varchar(64) | 昵称 |
| avatar | varchar(255) | 头像 |
| desc | longtext | 描述 |
| gender | varchar(2) | 性别 |
| follows | varchar(16) | 关注数 |
| fans | varchar(16) | 粉丝数 |
| interaction | varchar(16) | 获赞和收藏数 |
| tag_list | longtext | 标签列表 |

## 抖音 (dy)

### douyin_aweme

| 字段 | 类型 | 说明 |
|------|------|------|
| aweme_id | varchar(64) | 视频ID |
| title | varchar(1024) | 标题 |
| desc | longtext | 描述 |
| aweme_type | varchar(16) | 类型 |
| create_time | bigint | 发布时间戳 |
| liked_count | varchar(16) | 点赞数 |
| comment_count | varchar(16) | 评论数 |
| share_count | varchar(16) | 分享数 |
| collected_count | varchar(16) | 收藏数 |
| aweme_url | varchar(255) | 详情页URL |
| cover_url | varchar(500) | 封面图URL |
| video_download_url | varchar(1024) | 视频下载地址 |
| source_keyword | varchar(255) | 搜索来源关键字 |
| user_id | varchar(64) | 用户ID |
| sec_uid | varchar(128) | 用户sec_uid |
| nickname | varchar(64) | 昵称 |

### douyin_aweme_comment

| 字段 | 类型 | 说明 |
|------|------|------|
| comment_id | varchar(64) | 评论ID |
| aweme_id | varchar(64) | 视频ID |
| content | longtext | 评论内容 |
| create_time | bigint | 评论时间戳 |
| sub_comment_count | varchar(16) | 子评论数 |
| parent_comment_id | varchar(64) | 父评论ID |
| like_count | varchar(255) | 点赞数 |
| nickname | varchar(64) | 昵称 |
| ip_location | varchar(255) | IP地址 |

### dy_creator

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | varchar(128) | 用户ID |
| nickname | varchar(64) | 昵称 |
| avatar | varchar(255) | 头像 |
| desc | longtext | 描述 |
| gender | varchar(2) | 性别 |
| follows | varchar(16) | 关注数 |
| fans | varchar(16) | 粉丝数 |
| interaction | varchar(16) | 获赞数 |
| videos_count | varchar(16) | 作品数 |

## 快手 (ks)

### kuaishou_video

| 字段 | 类型 | 说明 |
|------|------|------|
| video_id | varchar(64) | 视频ID |
| title | varchar(500) | 标题 |
| desc | longtext | 描述 |
| video_type | varchar(16) | 类型 |
| create_time | bigint | 发布时间戳 |
| liked_count | varchar(16) | 点赞数 |
| viewd_count | varchar(16) | 浏览数 |
| video_url | varchar(512) | 详情URL |
| video_cover_url | text | 封面图URL |
| video_play_url | text | 播放URL |
| source_keyword | varchar(255) | 搜索来源关键字 |
| nickname | varchar(64) | 昵称 |

### kuaishou_video_comment

| 字段 | 类型 | 说明 |
|------|------|------|
| comment_id | varchar(64) | 评论ID |
| video_id | varchar(64) | 视频ID |
| content | longtext | 评论内容 |
| create_time | bigint | 评论时间戳 |
| sub_comment_count | varchar(16) | 子评论数 |
| parent_comment_id | varchar(64) | 父评论ID |
| like_count | varchar(255) | 点赞数 |
| nickname | varchar(64) | 昵称 |

### kuaishou_creator

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | varchar(64) | 用户ID |
| nickname | varchar(64) | 昵称 |
| avatar | varchar(255) | 头像 |
| desc | longtext | 描述 |
| gender | varchar(2) | 性别 |
| follows | varchar(16) | 关注数 |
| fans | varchar(16) | 粉丝数 |
| videos_count | varchar(16) | 作品数 |

## B站 (bili)

### bilibili_video

| 字段 | 类型 | 说明 |
|------|------|------|
| video_id | varchar(64) | 视频ID (aid) |
| bvid | varchar(64) | 视频ID (bvid) |
| title | varchar(500) | 标题 |
| desc | longtext | 描述 |
| video_type | varchar(16) | 类型 |
| create_time | bigint | 发布时间戳 |
| liked_count | varchar(16) | 点赞数 |
| video_play_count | varchar(16) | 播放数 |
| video_danmaku | varchar(16) | 弹幕数 |
| video_comment | varchar(16) | 评论数 |
| video_url | varchar(512) | 详情URL |
| video_cover_url | varchar(512) | 封面图URL |
| duration | varchar(16) | 时长 |
| source_keyword | varchar(255) | 搜索来源关键字 |
| nickname | varchar(64) | 昵称 |

### bilibili_video_comment

| 字段 | 类型 | 说明 |
|------|------|------|
| comment_id | varchar(64) | 评论ID |
| video_id | varchar(64) | 视频ID |
| content | longtext | 评论内容 |
| create_time | bigint | 评论时间戳 |
| sub_comment_count | varchar(16) | 子评论数 |
| parent_comment_id | varchar(64) | 父评论ID |
| like_count | varchar(255) | 点赞数 |
| nickname | varchar(64) | 昵称 |

### bilibili_up_info

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | varchar(64) | 用户ID |
| nickname | varchar(64) | 昵称 |
| avatar | varchar(255) | 头像 |
| follower_count | bigint | 粉丝数 |
| following_count | bigint | 关注数 |
| content_count | bigint | 作品数 |
| description | longtext | 描述 |

## 微博 (wb)

### weibo_note

| 字段 | 类型 | 说明 |
|------|------|------|
| note_id | varchar(64) | 帖子ID |
| content | longtext | 正文 |
| create_time | bigint | 发布时间戳 |
| create_date_time | varchar(32) | 发布日期时间 |
| liked_count | varchar(16) | 点赞数 |
| comments_count | varchar(16) | 评论数 |
| shared_count | varchar(16) | 转发数 |
| note_url | varchar(512) | 详情URL |
| image_list | longtext | 封面图列表 |
| video_url | longtext | 视频地址 |
| source_keyword | varchar(255) | 搜索来源关键字 |
| nickname | varchar(64) | 昵称 |
| profile_url | varchar(255) | 主页地址 |

### weibo_note_comment

| 字段 | 类型 | 说明 |
|------|------|------|
| comment_id | varchar(64) | 评论ID |
| note_id | varchar(64) | 帖子ID |
| content | longtext | 评论内容 |
| create_time | bigint | 评论时间戳 |
| sub_comment_count | varchar(16) | 子评论数 |
| parent_comment_id | varchar(64) | 父评论ID |
| like_count | varchar(255) | 点赞数 |
| nickname | varchar(64) | 昵称 |

### weibo_creator

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | varchar(64) | 用户ID |
| nickname | varchar(64) | 昵称 |
| avatar | varchar(255) | 头像 |
| desc | longtext | 描述 |
| gender | varchar(2) | 性别 |
| follows | varchar(16) | 关注数 |
| fans | varchar(16) | 粉丝数 |
| tag_list | longtext | 标签列表 |

## 贴吧 (tieba)

### tieba_note

| 字段 | 类型 | 说明 |
|------|------|------|
| note_id | varchar(644) | 帖子ID |
| title | varchar(255) | 标题 |
| desc | text | 描述 |
| note_url | varchar(255) | 帖子链接 |
| publish_time | varchar(255) | 发布时间 |
| user_nickname | varchar(255) | 昵称 |
| tieba_name | varchar(255) | 贴吧名称 |
| total_replay_num | int | 回复总数 |
| source_keyword | varchar(255) | 搜索来源关键字 |

### tieba_comment

| 字段 | 类型 | 说明 |
|------|------|------|
| comment_id | varchar(255) | 评论ID |
| note_id | varchar(255) | 帖子ID |
| content | text | 评论内容 |
| publish_time | varchar(255) | 发布时间 |
| sub_comment_count | int | 子评论数 |
| parent_comment_id | varchar(255) | 父评论ID |
| user_nickname | varchar(255) | 昵称 |

### tieba_creator

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | varchar(64) | 用户ID |
| user_name | varchar(64) | 用户名 |
| nickname | varchar(64) | 昵称 |
| gender | varchar(2) | 性别 |
| follows | varchar(16) | 关注数 |
| fans | varchar(16) | 粉丝数 |
| registration_duration | varchar(16) | 吧龄 |

## 知乎 (zhihu)

### zhihu_content

| 字段 | 类型 | 说明 |
|------|------|------|
| content_id | varchar(64) | 内容ID |
| content_type | varchar(16) | 类型 (article/answer/zvideo) |
| content_text | longtext | 内容文本 |
| content_url | varchar(255) | 落地链接 |
| question_id | varchar(64) | 问题ID（answer类型） |
| title | varchar(255) | 标题 |
| desc | longtext | 描述 |
| created_time | varchar(32) | 创建时间 |
| voteup_count | int | 赞同数 |
| comment_count | int | 评论数 |
| source_keyword | varchar(64) | 来源关键词 |
| user_id | varchar(64) | 用户ID |
| user_nickname | varchar(64) | 昵称 |

### zhihu_comment

| 字段 | 类型 | 说明 |
|------|------|------|
| comment_id | varchar(64) | 评论ID |
| content_id | varchar(64) | 内容ID |
| content_type | varchar(16) | 内容类型 |
| content | text | 评论内容 |
| publish_time | varchar(32) | 发布时间 |
| sub_comment_count | int | 子评论数 |
| like_count | int | 点赞数 |
| dislike_count | int | 踩数 |
| parent_comment_id | varchar(64) | 父评论ID |
| user_nickname | varchar(64) | 昵称 |

### zhihu_creator

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | varchar(64) | 用户ID |
| user_nickname | varchar(64) | 昵称 |
| url_token | varchar(64) | URL Token |
| gender | varchar(16) | 性别 |
| follows | int | 关注数 |
| fans | int | 粉丝数 |
| anwser_count | int | 回答数 |
| video_count | int | 视频数 |
| question_count | int | 问题数 |
| article_count | int | 文章数 |
| get_voteup_count | int | 获赞数 |
