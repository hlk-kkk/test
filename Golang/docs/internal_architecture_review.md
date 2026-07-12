# Internal 架构 Review

更新日期：2026-06-21

## 目标分层

`MediaCrawlerPro-Golang/internal` 目前应该按下面的边界继续收敛：

| 层级 | 目录 | 职责 |
| --- | --- | --- |
| 平台编排层 | `internal/<platform>` | 组装 crawler、注册 handler、处理 crawler type、保存业务数据、管理断点续爬 |
| API 客户端层 | `internal/service/<platform>` | 账号/代理/签名依赖、HTTP 请求、平台 API 响应解析 |
| API DTO 层 | `internal/models/<platform>` | 平台接口响应结构，尽量不混入持久化行为 |
| 基础设施层 | `internal/pkg/*`, `internal/repo`, `internal/crawler`, `internal/worker` | 账号池、HTTP、签名服务、数据库连接、checkpoint、并发池 |
| 测试支撑层 | `internal/testsupport`, `internal/e2e` | mock sign/cookie server、E2E runner、测试数据库/配置辅助 |

平台持久化模型保留在 `internal/<platform>/model.go`。这和 `internal/models/<platform>` 的 API DTO 是两类模型：前者对齐 MySQL/SQLite/CSV/JSON 输出，后者对齐平台接口响应。

## 平台包最低结构

每个平台包应保持一致的文件骨架：

```text
internal/<platform>/
  <platform>.go
  handler.go
  model.go
  store.go
  store_file.go
  store_sqlite.go
```

当前由 `internal/platform_structure_test.go` 固定这个约束：

- `TestPlatformPackagesUseConsistentFileLayout`
- `TestPlatformFileStoreLivesInDedicatedFile`
- `TestPlatformStoreFactoriesUseSharedSaveDataConstants`

已处理：

- `internal/xhs` 的 `FileStore` 已从 `store.go` 拆到 `store_file.go`。
- `internal/xhs` 和 `internal/bilibili` 的 `SaveDataOption` 分支已改用 `constant.SaveDataTo*`，和其它平台一致。

## 当前确认的不一致

### 0. 持久化模型和表名

`internal/<platform>/model.go` 的持久化模型负责 MySQL/SQLite/CSV/JSON 输出结构。当前 Go 表名和 Python `repo/platform_save_data/*/*_store_sql.py` 使用的基础表名一致：

| platform | Go persistence tables |
| --- | --- |
| bilibili | `bilibili_video`, `bilibili_video_comment`, `bilibili_up_info` |
| douyin | `douyin_aweme`, `douyin_aweme_comment`, `dy_creator` |
| kuaishou | `kuaishou_video`, `kuaishou_video_comment`, `kuaishou_creator`, `platform_hot_search_item`, `platform_hot_search_content_rel` |
| tieba | `tieba_note`, `tieba_comment`, `tieba_creator` |
| weibo | `weibo_note`, `weibo_note_comment`, `weibo_creator`, `platform_hot_search_item`, `platform_hot_search_content_rel` |
| xhs | `xhs_note`, `xhs_note_comment`, `xhs_creator` |
| zhihu | `zhihu_content`, `zhihu_comment`, `zhihu_creator`, `platform_hot_search_item`, `platform_hot_search_content_rel` |

`internal/platform_structure_test.go` 会从 Go `TableName()` 方法解析表名，并校验上表同步更新。

字段级对照结果如下。`Python-only fields` 为空表示 Go 持久化模型没有漏 Python 输出字段；`Go-only fields` 表示 Go 额外写入的字段。`id` 是 Go/GORM 内部主键，不列入差异；Python `WeiboComment.sub_comments` 只用于内存传递，也不列入持久化差异。

| platform | Python model -> Go model | Python-only fields | Go-only fields |
| --- | --- | --- | --- |
| xhs | XhsNote -> Note | - | `add_ts`, `last_modify_ts` |
| xhs | XhsComment -> NoteComment | - | `add_ts`, `last_modify_ts` |
| xhs | XhsCreator -> Creator | - | `add_ts`, `last_modify_ts` |
| bilibili | BilibiliVideo -> Video | - | - |
| bilibili | BilibiliComment -> VideoComment | - | - |
| bilibili | BilibiliUpInfo -> UpInfo | - | - |
| douyin | DouyinAweme -> Aweme | - | `add_ts`, `last_modify_ts` |
| douyin | DouyinAwemeComment -> AwemeComment | - | `add_ts`, `last_modify_ts` |
| douyin | DouyinCreator -> Creator | - | `add_ts`, `last_modify_ts` |
| kuaishou | KuaishouVideo -> Video | - | `add_ts`, `last_modify_ts` |
| kuaishou | KuaishouVideoComment -> VideoComment | - | `add_ts`, `last_modify_ts` |
| kuaishou | KuaishouCreator -> Creator | - | `add_ts`, `last_modify_ts` |
| kuaishou | HotSearchItem -> HotSearchItem | - | - |
| kuaishou | HotSearchContentRelation -> HotSearchContentRelation | - | - |
| weibo | WeiboNote -> Note | - | - |
| weibo | WeiboComment -> Comment | - | - |
| weibo | WeiboCreator -> Creator | - | - |
| weibo | HotSearchItem -> HotSearchItem | - | - |
| weibo | HotSearchContentRelation -> HotSearchContentRelation | - | - |
| tieba | TiebaNote -> Note | - | `add_ts`, `last_modify_ts`, `tieba_id` |
| tieba | TiebaComment -> Comment | - | `add_ts`, `last_modify_ts` |
| tieba | TiebaCreator -> Creator | - | `add_ts`, `last_modify_ts` |
| zhihu | ZhihuContent -> Content | - | `add_ts`, `last_modify_ts` |
| zhihu | ZhihuComment -> Comment | - | `add_ts`, `last_modify_ts` |
| zhihu | ZhihuCreator -> Creator | - | `add_ts`, `last_modify_ts` |
| zhihu | HotSearchItem -> HotSearchItem | - | - |
| zhihu | HotSearchContentRelation -> HotSearchContentRelation | - | - |

结论：

- Go 没有缺失 Python 持久化模型字段。
- 多数 Go-only `add_ts`/`last_modify_ts` 是 DB 审计字段，和部分 Python 模型已经显式包含的字段语义一致。
- `tieba.Note.tieba_id` 是 Go 额外字段；它不来自 Python `TiebaNote`，如果后续要求文件输出完全等同 Python，需要评估是否只保留 DB 层或从文件输出中过滤。

### 1. Processor/Worker 层只覆盖部分平台

`internal/xhs` 和 `internal/bilibili` 使用：

- `processor.go`
- `worker.Pool`
- `CrawlerOptions.WorkerSize`
- handler 通过 `Processor` 保存 note/comment/creator

`douyin/kuaishou/weibo/tieba/zhihu` 目前是 handler 直接调用 `Store`。

当前 Go 代码矩阵：

| platform | processor.go | worker pool in crawler |
| --- | --- | --- |
| bilibili | Y | Y |
| douyin | - | - |
| kuaishou | - | - |
| tieba | - | - |
| weibo | - | - |
| xhs | Y | Y |
| zhihu | - | - |

Python 版本每个平台都有 `processors/*_processor.py`，所以这不是 Python 业务结构的自然差异，而是 Go 迁移过程中的架构差异。后续需要做一次业务判断：

Decision: keep `processor.go` as an optional platform adapter for now.

Do not require every platform package to add `processor.go` until a shared processor contract is extracted.

当前判断：

- Python 每个平台都有 processor，但 Go 的非 `xhs/bilibili` 平台已经通过 handler 直接保存数据，并且现有 E2E 覆盖了对应 CLI 行为。
- 现在强制所有平台补 `processor.go` 会变成大范围搬运重构，但还没有清晰的 Go 公共 processor contract。
- `xhs/bilibili` 的 `processor.go` 先保留为平台可选 adapter，用来承接 worker pool、并发保存和平台特定的数据处理。
- 后续如果要收敛结构，应先提取共享 contract，再按平台用 TDD + E2E 逐个迁移，避免只为了文件形状一致而改变行为。

### 2. `internal/service/client.go` 依赖组装

所有 `New<Platform>Client()` 都需要相同的账号池基础依赖：

- `config.GetConfig()`
- `tools.GetLogger()`
- `accounts_cookies.CookiesAccountRepo`
- `account_pool.NewAccountPool`
- `account_pool.NewAccountWithIpPool`

已处理：

- `internal/service/client.go` 已抽出私有 `newClientBaseDeps()`，统一账号池依赖组装。
- 各平台 `New<Platform>Client()` public constructor 保持不变。
- `xhs/bilibili/douyin/zhihu` 仍按原有行为创建 `signsrv.NewSignClient`，不影响 `kuaishou/weibo/tieba`。

### 3. Checkpoint platform id 写法混杂

Checkpoint platform id 已统一为显式常量，默认对齐 Python 版本的 checkpoint platform 字段：

| platform package | canonical checkpoint platform | legacy readable platform |
| --- | --- | --- |
| `xhs` | `xhs` | - |
| `douyin` | `dy` | - |
| `kuaishou` | `ks` | - |
| `bilibili` | `bili` | `bilibili` |
| `weibo` | `wb` | - |
| `tieba` | `tieba` | - |
| `zhihu` | `zhihu` | - |

已处理：

- `constant.CheckpointPlatformValues` 固定 canonical checkpoint platform 值。
- 各平台 `NewCheckpointManager` 调用改为 `constant.<Platform>CheckpointPlatform`。
- B站新写 checkpoint 使用 Python-compatible `bili` 前缀。
- `CheckpointManager.Load` 支持 legacy platform alias，B站可以读取旧 Go 版 `bilibili_*` checkpoint 文件。

### 4. Handler 支持矩阵不完全一致

各平台 crawler type 注册不是完全相同：

- `xhs/bilibili/douyin/kuaishou/zhihu` 注册 `search/detail/creator/homefeed/hot_search`
- `weibo/tieba` 暂未注册 `homefeed`

这是业务能力差异，不应该强行补空 handler。文档和 E2E 应明确哪些平台支持哪些 crawler type。

当前代码注册矩阵：

| platform | search | detail | creator | homefeed | hot_search |
| --- | --- | --- | --- | --- | --- |
| bilibili | Y | Y | Y | Y | Y |
| douyin | Y | Y | Y | Y | Y |
| kuaishou | Y | Y | Y | Y | Y |
| tieba | Y | Y | Y | - | Y |
| weibo | Y | Y | Y | - | Y |
| xhs | Y | Y | Y | Y | Y |
| zhihu | Y | Y | Y | Y | Y |

`internal/platform_structure_test.go` 会从各平台 `<platform>.go` 解析 handler 注册，并校验上表同步更新。

## 迁移 Checklist

- [x] 平台 E2E 测试从根目录迁到 `tests/e2e`。
- [x] 平台包最低文件结构增加静态测试保护。
- [x] `internal/xhs` file store 拆到 `store_file.go`。
- [x] `internal/xhs` 和 `internal/bilibili` store factory 改用 `constant.SaveDataTo*`。
- [x] 盘点 Go 持久化表名，并和 Python store SQL 基础表名核对。
- [x] 为平台 crawler type 支持矩阵增加测试和文档表。
- [x] 评估 `Processor` 是否成为所有平台公共结构，或者只作为 `xhs/bilibili` 特化实现保留。
- [x] 收敛 `internal/service/client.go` 的账号池依赖组装重复。
- [x] 固定 checkpoint platform id 的兼容策略，并验证断点续爬覆盖。
- [x] 对比 Python 版本各平台输出字段，确认 `internal/<platform>/model.go` 的持久化结构没有漏字段。

## 验证命令

```bash
go test ./constant -count=1
go test ./internal -count=1
go test ./internal/crawler -count=1
go test ./internal/... -count=1
go test ./... -run '^$' -count=1
go test ./tests/e2e -run 'TestCLIBilibiliSearchCheckpointResumeSkipsCompletedPages' -count=1 -timeout 120s
go test ./tests/e2e -run 'TestCLI.*WithUnifiedRunner' -count=1 -timeout 420s
```
