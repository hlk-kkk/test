CREATE TABLE IF NOT EXISTS `platform_hot_search_item`
(
    `id`               bigint       NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `hot_item_id`      varchar(64)  NOT NULL COMMENT '热搜榜单项快照ID',
    `platform`         varchar(32)  NOT NULL COMMENT '平台名称',
    `board_name`       varchar(64)  NOT NULL DEFAULT 'hot_search' COMMENT '榜单名称',
    `board_type`       varchar(64)  NOT NULL DEFAULT '' COMMENT '榜单类型',
    `snapshot_date`    varchar(16)  NOT NULL COMMENT '采集日期',
    `snapshot_ts`      bigint       NOT NULL COMMENT '采集时间戳',
    `rank_no`          int          NOT NULL DEFAULT 0 COMMENT '榜单排名',
    `title`            varchar(500) NOT NULL DEFAULT '' COMMENT '热搜标题',
    `keyword`          varchar(500) NOT NULL DEFAULT '' COMMENT '检索关键词',
    `hot_value`        varchar(64)  NOT NULL DEFAULT '' COMMENT '热度值',
    `badge`            varchar(64)  NOT NULL DEFAULT '' COMMENT '热搜标签',
    `topic_id`         varchar(128) NOT NULL DEFAULT '' COMMENT '平台侧话题ID',
    `topic_url`        varchar(512) NOT NULL DEFAULT '' COMMENT '平台侧话题链接',
    `raw_json`         longtext COMMENT '平台原始榜单项JSON',
    `add_ts`           bigint       NOT NULL COMMENT '记录添加时间戳',
    `last_modify_ts`   bigint       NOT NULL COMMENT '记录最后修改时间戳',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_hot_search_item_id` (`hot_item_id`),
    KEY `idx_hot_search_item_board` (`platform`, `board_name`, `snapshot_date`, `rank_no`),
    KEY `idx_hot_search_item_title` (`platform`, `title`(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='跨平台热搜榜单项快照';

CREATE TABLE IF NOT EXISTS `platform_hot_search_content_rel`
(
    `id`              bigint       NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `rel_id`          varchar(64)  NOT NULL COMMENT '热搜内容关系ID',
    `hot_item_id`     varchar(64)  NOT NULL COMMENT '热搜榜单项快照ID',
    `platform`        varchar(32)  NOT NULL COMMENT '平台名称',
    `content_id`      varchar(128) NOT NULL COMMENT '内容ID',
    `content_type`    varchar(64)  NOT NULL DEFAULT '' COMMENT '内容类型',
    `content_table`   varchar(64)  NOT NULL DEFAULT '' COMMENT '内容表名',
    `rank_no`         int          NOT NULL DEFAULT 0 COMMENT '榜单排名',
    `source_keyword`  varchar(500) NOT NULL DEFAULT '' COMMENT '来源关键词',
    `add_ts`          bigint       NOT NULL COMMENT '记录添加时间戳',
    `last_modify_ts`  bigint       NOT NULL COMMENT '记录最后修改时间戳',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_hot_search_rel_id` (`rel_id`),
    KEY `idx_hot_search_rel_item` (`hot_item_id`),
    KEY `idx_hot_search_rel_content` (`platform`, `content_table`, `content_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='热搜榜单项与内容关系';
