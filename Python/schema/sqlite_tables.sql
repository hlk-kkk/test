-- MediaCrawlerPro SQLite 数据库表结构
-- 由 schema/tables.sql 转换而来，兼容 SQLite 语法
-- 转换规则：AUTO_INCREMENT→AUTOINCREMENT，varchar/longtext→TEXT，bigint/int→INTEGER
--           删除 ENGINE/CHARSET/COLLATE/COMMENT，KEY 转为独立 CREATE INDEX 语句

DROP TABLE IF EXISTS `bilibili_video`;
CREATE TABLE `bilibili_video`
(
    `id`               INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`         TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`           INTEGER NOT NULL,
    `last_modify_ts`   INTEGER NOT NULL,
    `video_id`         TEXT NOT NULL,
    `bvid`             TEXT DEFAULT NULL,
    `video_type`       TEXT NOT NULL,
    `title`            TEXT DEFAULT NULL,
    `desc`             TEXT,
    `create_time`      INTEGER NOT NULL,
    `liked_count`      TEXT DEFAULT NULL,
    `video_play_count` TEXT DEFAULT NULL,
    `video_danmaku`    TEXT DEFAULT NULL,
    `video_comment`    TEXT DEFAULT NULL,
    `video_url`        TEXT DEFAULT NULL,
    `video_cover_url`  TEXT DEFAULT NULL,
    `source_keyword`   TEXT DEFAULT '',
    `duration`         TEXT DEFAULT NULL
);
CREATE INDEX IF NOT EXISTS idx_bilibili_vi_video_i_31c36e ON bilibili_video (`video_id`);
CREATE INDEX IF NOT EXISTS idx_bilibili_vi_create__73e0ec ON bilibili_video (`create_time`);


DROP TABLE IF EXISTS `bilibili_video_comment`;
CREATE TABLE `bilibili_video_comment`
(
    `id`                INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`          TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`            INTEGER NOT NULL,
    `last_modify_ts`    INTEGER NOT NULL,
    `comment_id`        TEXT NOT NULL,
    `video_id`          TEXT NOT NULL,
    `content`           TEXT,
    `create_time`       INTEGER NOT NULL,
    `sub_comment_count` TEXT NOT NULL,
    `parent_comment_id` TEXT DEFAULT NULL,
    `like_count`        TEXT NOT NULL DEFAULT '0'
);
CREATE INDEX IF NOT EXISTS idx_bilibili_vi_comment_41c34e ON bilibili_video_comment (`comment_id`);
CREATE INDEX IF NOT EXISTS idx_bilibili_vi_video_i_f22873 ON bilibili_video_comment (`video_id`);




DROP TABLE IF EXISTS `douyin_aweme`;
CREATE TABLE `douyin_aweme`
(
    `id`                 INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`           TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`             INTEGER NOT NULL,
    `last_modify_ts`     INTEGER NOT NULL,
    `aweme_id`           TEXT NOT NULL,
    `aweme_type`         TEXT NOT NULL,
    `title`              TEXT DEFAULT NULL,
    `desc`               TEXT,
    `create_time`        INTEGER NOT NULL,
    `liked_count`        TEXT DEFAULT NULL,
    `comment_count`      TEXT DEFAULT NULL,
    `share_count`        TEXT DEFAULT NULL,
    `collected_count`    TEXT DEFAULT NULL,
    `aweme_url`          TEXT DEFAULT NULL,
    `cover_url`          TEXT DEFAULT NULL,
    `video_download_url` TEXT DEFAULT NULL,
    `source_keyword`     TEXT DEFAULT '',
    `is_ai_generated`    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_douyin_awem_aweme_i_6f7bc6 ON douyin_aweme (`aweme_id`);
CREATE INDEX IF NOT EXISTS idx_douyin_awem_create__299dfe ON douyin_aweme (`create_time`);


DROP TABLE IF EXISTS `douyin_aweme_comment`;
CREATE TABLE `douyin_aweme_comment`
(
    `id`                INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`          TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`            INTEGER NOT NULL,
    `last_modify_ts`    INTEGER NOT NULL,
    `comment_id`        TEXT NOT NULL,
    `aweme_id`          TEXT NOT NULL,
    `content`           TEXT,
    `create_time`       INTEGER NOT NULL,
    `sub_comment_count` TEXT NOT NULL,
    `parent_comment_id` TEXT DEFAULT NULL,
    `like_count`        TEXT NOT NULL DEFAULT '0',
    `pictures`          TEXT NOT NULL DEFAULT '',
    `reply_to_reply_id` TEXT DEFAULT NULL
);
CREATE INDEX IF NOT EXISTS idx_douyin_awem_comment_fcd7e4 ON douyin_aweme_comment (`comment_id`);
CREATE INDEX IF NOT EXISTS idx_douyin_awem_aweme_i_c50049 ON douyin_aweme_comment (`aweme_id`);




DROP TABLE IF EXISTS `kuaishou_video`;
CREATE TABLE `kuaishou_video`
(
    `id`              INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`        TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`          INTEGER NOT NULL,
    `last_modify_ts`  INTEGER NOT NULL,
    `video_id`        TEXT NOT NULL,
    `video_type`      TEXT NOT NULL,
    `title`           TEXT DEFAULT NULL,
    `desc`            TEXT,
    `create_time`     INTEGER NOT NULL,
    `liked_count`     TEXT DEFAULT NULL,
    `viewd_count`     TEXT DEFAULT NULL,
    `video_url`       TEXT DEFAULT NULL,
    `video_cover_url` TEXT DEFAULT NULL,
    `video_play_url`  TEXT DEFAULT NULL,
    `source_keyword`  TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_video_i_c5c6a6 ON kuaishou_video (`video_id`);
CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_create__a10dee ON kuaishou_video (`create_time`);


DROP TABLE IF EXISTS `kuaishou_video_comment`;
CREATE TABLE `kuaishou_video_comment`
(
    `id`                INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`          TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`            INTEGER NOT NULL,
    `last_modify_ts`    INTEGER NOT NULL,
    `comment_id`        TEXT NOT NULL,
    `parent_comment_id` TEXT DEFAULT NULL,
    `video_id`          TEXT NOT NULL,
    `content`           TEXT,
    `create_time`       INTEGER NOT NULL,
    `sub_comment_count` TEXT NOT NULL,
    `like_count`        TEXT NOT NULL DEFAULT '0'
);
CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_comment_ed48fa ON kuaishou_video_comment (`comment_id`);
CREATE INDEX IF NOT EXISTS idx_kuaishou_vi_video_i_e50914 ON kuaishou_video_comment (`video_id`);




DROP TABLE IF EXISTS `weibo_note`;
CREATE TABLE `weibo_note`
(
    `id`               INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`         TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`           INTEGER NOT NULL,
    `last_modify_ts`   INTEGER NOT NULL,
    `note_id`          TEXT NOT NULL,
    `content`          TEXT,
    `create_time`      INTEGER NOT NULL,
    `create_date_time` TEXT NOT NULL,
    `liked_count`      TEXT DEFAULT NULL,
    `comments_count`   TEXT DEFAULT NULL,
    `shared_count`     TEXT DEFAULT NULL,
    `note_url`         TEXT DEFAULT NULL,
    `image_list`       TEXT,
    `video_url`        TEXT,
    `source_keyword`   TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_weibo_note_note_id_f95b1a ON weibo_note (`note_id`);
CREATE INDEX IF NOT EXISTS idx_weibo_note_create__692709 ON weibo_note (`create_time`);
CREATE INDEX IF NOT EXISTS idx_weibo_note_create__d05ed2 ON weibo_note (`create_date_time`);


DROP TABLE IF EXISTS `weibo_note_comment`;
CREATE TABLE `weibo_note_comment`
(
    `id`                INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`          TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`            INTEGER NOT NULL,
    `last_modify_ts`    INTEGER NOT NULL,
    `comment_id`        TEXT NOT NULL,
    `note_id`           TEXT NOT NULL,
    `content`           TEXT,
    `create_time`       INTEGER NOT NULL,
    `create_date_time`  TEXT NOT NULL,
    `sub_comment_count` TEXT NOT NULL,
    `parent_comment_id` TEXT DEFAULT NULL,
    `like_count`        TEXT NOT NULL DEFAULT '0'
);
CREATE INDEX IF NOT EXISTS idx_weibo_note__comment_c7611c ON weibo_note_comment (`comment_id`);
CREATE INDEX IF NOT EXISTS idx_weibo_note__note_id_24f108 ON weibo_note_comment (`note_id`);
CREATE INDEX IF NOT EXISTS idx_weibo_note__create__667fe3 ON weibo_note_comment (`create_date_time`);




DROP TABLE IF EXISTS `xhs_note`;
CREATE TABLE `xhs_note`
(
    `id`               INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`         TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`           INTEGER NOT NULL,
    `last_modify_ts`   INTEGER NOT NULL,
    `note_id`          TEXT NOT NULL,
    `type`             TEXT DEFAULT NULL,
    `title`            TEXT DEFAULT NULL,
    `desc`             TEXT,
    `video_url`        TEXT,
    `time`             INTEGER NOT NULL,
    `last_update_time` INTEGER NOT NULL,
    `liked_count`      TEXT DEFAULT NULL,
    `collected_count`  TEXT DEFAULT NULL,
    `comment_count`    TEXT DEFAULT NULL,
    `share_count`      TEXT DEFAULT NULL,
    `image_list`       TEXT,
    `tag_list`         TEXT,
    `note_url`         TEXT DEFAULT NULL,
    `source_keyword`   TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_xhs_note_note_id_209457 ON xhs_note (`note_id`);
CREATE INDEX IF NOT EXISTS idx_xhs_note_time_eaa910 ON xhs_note (`time`);


DROP TABLE IF EXISTS `xhs_note_comment`;
CREATE TABLE `xhs_note_comment`
(
    `id`                INTEGER PRIMARY KEY AUTOINCREMENT,
    `nickname`          TEXT DEFAULT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`            INTEGER NOT NULL,
    `last_modify_ts`    INTEGER NOT NULL,
    `comment_id`        TEXT NOT NULL,
    `create_time`       INTEGER NOT NULL,
    `note_id`           TEXT NOT NULL,
    `content`           TEXT NOT NULL,
    `sub_comment_count` TEXT NOT NULL,
    `pictures`          TEXT DEFAULT NULL,
    `parent_comment_id` TEXT DEFAULT NULL,
    `like_count`        TEXT NOT NULL DEFAULT '0',
    `note_url`          TEXT NOT NULL DEFAULT '',
    `target_comment_id` TEXT DEFAULT NULL
);
CREATE INDEX IF NOT EXISTS idx_xhs_note_co_comment_8e8349 ON xhs_note_comment (`comment_id`);
CREATE INDEX IF NOT EXISTS idx_xhs_note_co_create__204f8d ON xhs_note_comment (`create_time`);


DROP TABLE IF EXISTS `tieba_note`;
CREATE TABLE tieba_note
(
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    note_id           TEXT    NOT NULL,
    title             TEXT    NOT NULL,
    `desc`            TEXT,
    note_url          TEXT    NOT NULL,
    publish_time      TEXT    NOT NULL,
    user_nickname     TEXT    DEFAULT '',
    `creator_hash`     TEXT DEFAULT '',
    tieba_id          TEXT    DEFAULT '',
    tieba_name        TEXT    NOT NULL,
    tieba_link        TEXT    NOT NULL,
    total_replay_num  INTEGER DEFAULT 0,
    total_replay_page INTEGER DEFAULT 0,
    add_ts            INTEGER NOT NULL,
    last_modify_ts    INTEGER NOT NULL,
    source_keyword    TEXT    DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_tieba_note_note_id ON tieba_note (note_id);
CREATE INDEX IF NOT EXISTS idx_tieba_note_publish_time ON tieba_note (publish_time);


DROP TABLE IF EXISTS `tieba_comment`;
CREATE TABLE tieba_comment
(
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    comment_id        TEXT    NOT NULL,
    parent_comment_id TEXT    DEFAULT '',
    content           TEXT    NOT NULL,
    user_nickname     TEXT    DEFAULT '',
    `creator_hash`     TEXT DEFAULT '',
    tieba_id          TEXT    DEFAULT '',
    tieba_name        TEXT    NOT NULL,
    tieba_link        TEXT    NOT NULL,
    publish_time      TEXT    DEFAULT '',
    sub_comment_count INTEGER DEFAULT 0,
    note_id           TEXT    NOT NULL,
    note_url          TEXT    NOT NULL,
    add_ts            INTEGER NOT NULL,
    last_modify_ts    INTEGER NOT NULL
);
-- 注：MySQL 原版两个索引均建在 note_id 列，保持一致（idx_tieba_comment_comment_id 命名是 MySQL 版的历史 bug）
CREATE INDEX IF NOT EXISTS idx_tieba_comment_comment_id ON tieba_comment (note_id);
CREATE INDEX IF NOT EXISTS idx_tieba_comment_note_id ON tieba_comment (note_id);
CREATE INDEX IF NOT EXISTS idx_tieba_comment_publish_time ON tieba_comment (publish_time);


DROP TABLE IF EXISTS `crawler_cookies_account`;
CREATE TABLE `crawler_cookies_account`
(
    `id`                INTEGER PRIMARY KEY AUTOINCREMENT,
    `account_name`      TEXT    NOT NULL DEFAULT '',
    `platform_name`     TEXT    NOT NULL DEFAULT '',
    `cookies`           TEXT,
    `create_time`       TEXT    DEFAULT CURRENT_TIMESTAMP,
    `update_time`       TEXT    DEFAULT CURRENT_TIMESTAMP,
    `invalid_timestamp` INTEGER NOT NULL DEFAULT 0,
    `status`            INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_crawler_cookies_account_01 ON crawler_cookies_account (`update_time`);

-- SQLite 不支持 ON UPDATE CURRENT_TIMESTAMP，用 trigger 模拟
-- WHEN 条件防止 trigger 递归触发自身
CREATE TRIGGER IF NOT EXISTS trg_cookies_account_update_time
    AFTER UPDATE ON crawler_cookies_account
    FOR EACH ROW
    WHEN NEW.update_time = OLD.update_time
BEGIN
    UPDATE crawler_cookies_account SET update_time = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;






DROP TABLE IF EXISTS `zhihu_content`;
CREATE TABLE `zhihu_content`
(
    `id`              INTEGER PRIMARY KEY AUTOINCREMENT,
    `content_id`      TEXT    NOT NULL,
    `content_type`    TEXT    NOT NULL,
    `content_text`    TEXT,
    `content_url`     TEXT    NOT NULL,
    `question_id`     TEXT    DEFAULT NULL,
    `title`           TEXT    NOT NULL,
    `desc`            TEXT,
    `created_time`    TEXT    NOT NULL,
    `updated_time`    TEXT    NOT NULL,
    `voteup_count`    INTEGER NOT NULL DEFAULT 0,
    `comment_count`   INTEGER NOT NULL DEFAULT 0,
    `source_keyword`  TEXT    DEFAULT NULL,
    `user_nickname`   TEXT    NOT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`          INTEGER NOT NULL,
    `last_modify_ts`  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_zhihu_content_content_id ON zhihu_content (`content_id`);
CREATE INDEX IF NOT EXISTS idx_zhihu_content_created_time ON zhihu_content (`created_time`);


DROP TABLE IF EXISTS `zhihu_comment`;
CREATE TABLE `zhihu_comment`
(
    `id`                INTEGER PRIMARY KEY AUTOINCREMENT,
    `comment_id`        TEXT    NOT NULL,
    `parent_comment_id` TEXT    DEFAULT NULL,
    `content`           TEXT    NOT NULL,
    `publish_time`      TEXT    NOT NULL,
    `sub_comment_count` INTEGER NOT NULL DEFAULT 0,
    `like_count`        INTEGER NOT NULL DEFAULT 0,
    `dislike_count`     INTEGER NOT NULL DEFAULT 0,
    `content_id`        TEXT    NOT NULL,
    `content_type`      TEXT    NOT NULL,
    `user_nickname`     TEXT    NOT NULL,
    `creator_hash`     TEXT DEFAULT '',
    `add_ts`            INTEGER NOT NULL,
    `last_modify_ts`    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_zhihu_comment_comment_id ON zhihu_comment (`comment_id`);
CREATE INDEX IF NOT EXISTS idx_zhihu_comment_content_id ON zhihu_comment (`content_id`);
CREATE INDEX IF NOT EXISTS idx_zhihu_comment_publish_time ON zhihu_comment (`publish_time`);




DROP TABLE IF EXISTS `platform_hot_search_item`;
CREATE TABLE `platform_hot_search_item`
(
    `id`             INTEGER PRIMARY KEY AUTOINCREMENT,
    `hot_item_id`    TEXT    NOT NULL,
    `platform`       TEXT    NOT NULL,
    `board_name`     TEXT    NOT NULL DEFAULT 'hot_search',
    `board_type`     TEXT    NOT NULL DEFAULT '',
    `snapshot_date`  TEXT    NOT NULL,
    `snapshot_ts`    INTEGER NOT NULL,
    `rank_no`        INTEGER NOT NULL DEFAULT 0,
    `title`          TEXT    NOT NULL DEFAULT '',
    `keyword`        TEXT    NOT NULL DEFAULT '',
    `hot_value`      TEXT    NOT NULL DEFAULT '',
    `badge`          TEXT    NOT NULL DEFAULT '',
    `topic_id`       TEXT    NOT NULL DEFAULT '',
    `topic_url`      TEXT    NOT NULL DEFAULT '',
    `raw_json`       TEXT,
    `add_ts`         INTEGER NOT NULL,
    `last_modify_ts` INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_hot_search_item_id ON platform_hot_search_item (`hot_item_id`);
CREATE INDEX IF NOT EXISTS idx_hot_search_item_board ON platform_hot_search_item (`platform`, `board_name`, `snapshot_date`, `rank_no`);
CREATE INDEX IF NOT EXISTS idx_hot_search_item_title ON platform_hot_search_item (`platform`, `title`);


DROP TABLE IF EXISTS `platform_hot_search_content_rel`;
CREATE TABLE `platform_hot_search_content_rel`
(
    `id`             INTEGER PRIMARY KEY AUTOINCREMENT,
    `rel_id`         TEXT    NOT NULL,
    `hot_item_id`    TEXT    NOT NULL,
    `platform`       TEXT    NOT NULL,
    `content_id`     TEXT    NOT NULL,
    `content_type`   TEXT    NOT NULL DEFAULT '',
    `content_table`  TEXT    NOT NULL DEFAULT '',
    `rank_no`        INTEGER NOT NULL DEFAULT 0,
    `source_keyword` TEXT    NOT NULL DEFAULT '',
    `add_ts`         INTEGER NOT NULL,
    `last_modify_ts` INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_hot_search_rel_id ON platform_hot_search_content_rel (`rel_id`);
CREATE INDEX IF NOT EXISTS idx_hot_search_rel_item ON platform_hot_search_content_rel (`hot_item_id`);
CREATE INDEX IF NOT EXISTS idx_hot_search_rel_content ON platform_hot_search_content_rel (`platform`, `content_table`, `content_id`);
