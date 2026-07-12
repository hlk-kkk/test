# MediaCrawlerPro-Golang 运行手册

本文说明如何在本地运行 Go 版本爬虫、如何配置依赖服务、如何选择存储后端，以及如何执行 `/tmp` mock E2E 验证。

## 1. 基本准备

在 `MediaCrawlerPro-Golang` 目录下执行：

```bash
go mod download
go build -o /tmp/mcpro-go .
```

常用检查命令：

```bash
go test ./cmd ./config ./constant -count=1
go test -p 1 -count=1 -timeout 420s ./...
```

全量 E2E 会真实编译并启动 CLI，耗时通常是数分钟。日常开发优先跑聚焦测试，最后再跑全量命令。

## 2. CLI 入口

Go CLI 尽量保持 Python 入口兼容：

```bash
go run . \
  --platform xhs|dy|ks|bili|wb|tieba|zhihu \
  --type search|detail|creator|homefeed|hot_search \
  --keywords "AI,deepseek" \
  --save_data_option db|sqlite|json|csv
```

也可以使用编译后的二进制：

```bash
/tmp/mcpro-go --platform xhs --type search --keywords "deepseek" --save_data_option json
```

支持的平台和 Python 兼容别名：

| 平台 | CLI 值 |
| --- | --- |
| 小红书 | `xhs` |
| 抖音 | `dy` |
| 快手 | `ks` |
| Bilibili | `bili` |
| 微博 | `wb` |
| 贴吧 | `tieba` |
| 知乎 | `zhihu` |

支持的 crawler type：

| 类型 | 用途 |
| --- | --- |
| `search` | 按关键词搜索内容 |
| `detail` | 按指定内容 URL 或 ID 抓详情 |
| `creator` | 按创作者主页 URL 或 ID 抓创作者信息和作品 |
| `homefeed` | 抓推荐流，平台支持范围与 Python 版本保持一致 |
| `hot_search` | 抓热榜/热搜；XHS、Tieba 当前按 Python placeholder 行为处理 |

常用 flags：

```bash
--platform <xhs|dy|ks|bili|wb|tieba|zhihu>
--type <search|detail|creator|homefeed|hot_search>
--keywords "kw1,kw2"
--urls "url_or_id_1,url_or_id_2"
--save_data_option <db|sqlite|json|csv>
--enable_comments
--no-enable_comments
--enable_sub_comments
--no-enable_sub_comments
--enable_checkpoint
--no-enable_checkpoint
--checkpoint_id <stable-id>
--hot_search_max_items <n>
--hot_search_per_item_count <n>
--start <page>
```

## 3. 配置文件

程序启动时会读取 `config.yml`，查找顺序是：

1. `config/config.yml`
2. `../config/config.yml`
3. 当前工作目录下的 `config.yml`

推荐保留 `config/config.yml` 作为默认配置，通过环境变量覆盖本地敏感信息和测试路径。

第一次运行可以从示例文件复制：

```bash
cp config/config.yml.example config/config.yml
```

然后按下面的最小配置调整 `platform`、`crawlerType`、`saveDataOption`、`accountPoolSaveType` 和依赖服务地址。

最小可运行配置示例：

```yaml
platform: "xhs"
keywords:
  - "deepseek"
crawlerType: "search"
saveDataOption: "json"
accountPoolSaveType: "cookie_bridge"
cookieBridgeURL: "http://localhost:8274"
dataDir: "data"
sqliteDbPath: "media_crawler.db"
startPage: 1
crawlerMaxNotesCount: 20
maxConcurrencyNum: 1
requestTimeOut: 10
enableGetComments: false
enableGetSubComments: false
enableCheckpoint: true
crawlerTimeSleep: 1
perNoteMaxComments: 20
hotSearchMaxItemsCount: 50
hotSearchPerItemContentCount: 4
debugRequest: false

proxyConfig:
  enableIpProxy: false

signSrvConfig:
  signSrvHost: "127.0.0.1"
  signSrvPort: 8989

platformEndpoints: {}
```

`detail` 和 `creator` 模式可以在配置中写 URL，也可以直接用 `--urls` 覆盖：

```yaml
specifiedNoteURLs:
  - "https://www.xiaohongshu.com/explore/NOTE_ID?xsec_token=...&xsec_source=pc_search"

specifiedCreatorURLs:
  - "https://www.xiaohongshu.com/user/profile/CREATOR_ID?xsec_token=...&xsec_source=pc_user"
```

## 4. 环境变量

数据库连接优先用环境变量覆盖，避免把本地密码写进配置文件：

```bash
export RELATION_DB_HOST=127.0.0.1
export RELATION_DB_PORT=3306
export RELATION_DB_USER=root
export RELATION_DB_PWD=your_password
export RELATION_DB_NAME=media_crawler_pro
```

SQLite 和数据目录：

```bash
export MCPRO_DATA_DIR=/tmp/mcpro-data
export MCPRO_SQLITE_DB_PATH=/tmp/mcpro-media-crawler.db
# 兼容旧变量
export SQLITE_DB_PATH=/tmp/mcpro-media-crawler.db
```

CookieBridge：

```bash
export COOKIE_BRIDGE_URL=http://localhost:8274
```

热搜 checkpoint 快照最大可恢复时间：

```bash
export HOT_SEARCH_CHECKPOINT_MAX_AGE_SECONDS=21600
```

平台 endpoint 可被环境变量覆盖，主要用于 mock server 和 E2E：

```bash
export MCPRO_XHS_API_URL=http://127.0.0.1:18080
export MCPRO_XHS_INDEX_URL=http://127.0.0.1:18080
export MCPRO_BILIBILI_API_URL=http://127.0.0.1:18080
export MCPRO_BILIBILI_INDEX_URL=http://127.0.0.1:18080
export MCPRO_BILIBILI_SPACE_URL=http://127.0.0.1:18080
export MCPRO_DOUYIN_API_URL=http://127.0.0.1:18080
export MCPRO_DOUYIN_INDEX_URL=http://127.0.0.1:18080
export MCPRO_KUAISHOU_API_URL=http://127.0.0.1:18080/graphql
export MCPRO_KUAISHOU_REST_API_URL=http://127.0.0.1:18080
export MCPRO_WEIBO_API_URL=http://127.0.0.1:18080
export MCPRO_WEIBO_HOT_SEARCH_URL=http://127.0.0.1:18080/ajax/side/hotSearch
export MCPRO_WEIBO_HOT_BAND_URL=http://127.0.0.1:18080/ajax/statuses/hot_band
export MCPRO_TIEBA_URL=http://127.0.0.1:18080
export MCPRO_ZHIHU_URL=http://127.0.0.1:18080
export MCPRO_ZHIHU_ZHUANLAN_URL=http://127.0.0.1:18080
```

自动化测试防外网开关：

```bash
export MCPRO_TEST_BLOCK_EXTERNAL_HTTP=1
```

该开关会阻断非 loopback HTTP 连接，E2E 测试默认启用，用来证明测试只访问本地 mock server。

## 5. 依赖服务

### CookieBridge

真实抓取需要从 CookieBridge 获取账号 Cookie。默认地址是：

```bash
http://localhost:8274
```

运行前可以检查：

```bash
curl http://localhost:8274/ping
curl http://localhost:8274/api/status
curl -s http://localhost:8274/api/accounts | jq '.data.accounts[] | {client_id, platforms}'
```

业务运行不需要手动读取 Cookie 内容；Go 版本会通过账号池从 CookieBridge 的 `/api/accounts` 和 `/api/cookies/{platform}` 自动加载对应平台账号。

### SignSrv

XHS、Bilibili、Douyin、Zhihu 等请求会使用签名服务。配置项：

```yaml
signSrvConfig:
  signSrvHost: "127.0.0.1"
  signSrvPort: 8989
```

测试 mock 会内置 SignSrv 兼容接口；真实抓取时需要本地签名服务可用。

运行前可以检查：

```bash
curl http://127.0.0.1:8989/signsrv/pong
```

## 6. 存储后端

`--save_data_option` 或 `saveDataOption` 决定存储方式。

### JSON

```bash
go run . --platform xhs --type search --keywords "deepseek" --save_data_option json
```

输出目录默认在 `data/<platform>/json/`，可用 `MCPRO_DATA_DIR` 改变根目录。

### CSV

```bash
go run . --platform xhs --type search --keywords "deepseek" --save_data_option csv
```

输出目录默认在 `data/<platform>/`。CSV 会按 Python 兼容字段写入。

### SQLite

```bash
MCPRO_SQLITE_DB_PATH=/tmp/media_crawler.db \
go run . --platform zhihu --type hot_search --save_data_option sqlite --hot_search_max_items 2
```

SQLite 会自动建表和索引，适合本地验证和单机使用。

### MySQL/GORM

```bash
export RELATION_DB_HOST=127.0.0.1
export RELATION_DB_PORT=3306
export RELATION_DB_USER=root
export RELATION_DB_PWD=your_password
export RELATION_DB_NAME=media_crawler_pro

go run . --platform bili --type search --keywords "AI" --save_data_option db
```

MySQL 模式会走 GORM upsert。运行前确认数据库已存在且账号有建表/写入权限。

## 7. 常见运行命令

搜索内容：

```bash
go run . --platform xhs --type search --keywords "deepseek,AI" --save_data_option json
```

抓详情：

```bash
go run . \
  --platform xhs \
  --type detail \
  --urls "https://www.xiaohongshu.com/explore/NOTE_ID?xsec_token=...&xsec_source=pc_search" \
  --save_data_option sqlite
```

抓创作者：

```bash
go run . \
  --platform bili \
  --type creator \
  --urls "12345678" \
  --save_data_option csv
```

抓推荐流：

```bash
go run . --platform zhihu --type homefeed --save_data_option json --no-enable_comments
```

抓热搜/热榜：

```bash
go run . \
  --platform wb \
  --type hot_search \
  --hot_search_max_items 5 \
  --hot_search_per_item_count 2 \
  --save_data_option sqlite
```

开启评论和二级评论：

```bash
go run . \
  --platform dy \
  --type search \
  --keywords "AI" \
  --enable_comments \
  --enable_sub_comments \
  --save_data_option json
```

关闭评论：

```bash
go run . --platform ks --type search --keywords "AI" --no-enable_comments
```

## 8. 断点续爬

开启 checkpoint：

```bash
go run . \
  --platform xhs \
  --type search \
  --keywords "deepseek" \
  --enable_checkpoint \
  --checkpoint_id "xhs-deepseek-search" \
  --save_data_option json
```

下一次使用同一个平台、类型、关键词或同一个 `--checkpoint_id` 会恢复进度并跳过已写入内容。不同 crawler type 的恢复维度与 Python 版本保持一致：

| 类型 | 恢复维度 |
| --- | --- |
| `search` | keyword、page/offset/cursor、已抓内容 ID |
| `detail` | 指定 URL/ID 列表、已抓内容 ID |
| `creator` | creator ID、page/cursor/offset、已抓内容 ID |
| `homefeed` | cursor/index/page、已抓内容 ID |
| `hot_search` | 热榜快照、榜单项 index、每个热榜项已抓内容 ID、快照最大年龄 |

禁用 checkpoint：

```bash
go run . --platform xhs --type search --keywords "deepseek" --no-enable_checkpoint
```

## 9. Mock E2E

Go E2E 会在 `/tmp` 下创建工作目录、配置目录、数据目录和 mock HTTP server，然后运行编译后的真实 CLI，并检查 JSON、CSV、SQLite、MySQL 和 checkpoint 输出。

测试代码和运行时文件约定：

- 黑盒 CLI E2E 放在 `tests/e2e/`。
- 可复用 mock、临时 MySQL、CLI runner 和文件/DB 断言工具放在 `internal/e2e/` 与 `internal/testsupport/`。
- 测试运行产生的 config、data、sqlite、checkpoint、mock server 目录全部创建在 `/tmp`，不要放到仓库根目录。

常用命令：

```bash
go test ./internal/e2e -count=1
go test ./tests/e2e -run 'TestCLIXhsSearchJSONWithUnifiedRunner' -count=1 -timeout 120s
go test ./tests/e2e -run 'TestCLI.*WithUnifiedRunner' -count=1 -timeout 420s
go test ./tests/e2e -run 'TestCLI.*Checkpoint' -count=1 -timeout 420s
go test -p 1 -count=1 -timeout 420s ./...
```

MySQL E2E 会启动临时 MySQL 数据目录：

```bash
go test ./internal/e2e -run 'TestStartMySQLCreatesReachableDatabaseUnderTmp' -count=1
go test ./tests/e2e -run 'TestCLI.*DBWithUnifiedRunner' -count=1 -timeout 420s
```

测试默认启用：

```bash
MCPRO_TEST_BLOCK_EXTERNAL_HTTP=1
```

如果 E2E 误访问真实平台域名，会被 HTTP guard 拦截。

## 10. 真实平台 Live E2E

Live E2E 默认跳过，只有显式传入可用 CookieBridge URL 才会访问真实平台。运行前确认 CookieBridge 与 SignSrv 都可用：

```bash
curl http://localhost:8274/ping
curl http://127.0.0.1:8989/signsrv/pong
```

B站 live E2E 覆盖 `search/detail/creator/homefeed/hot_search`，并检查 JSON、CSV、SQLite、临时 MySQL 写入：

```bash
MCPRO_BILIBILI_LIVE_COOKIE_BRIDGE_URL=http://127.0.0.1:8274 \
MCPRO_BILIBILI_LIVE_KEYWORD=deepseek \
go test ./tests/e2e -run TestCLIBilibiliLiveBusinessWithRealCookie -count=1 -timeout 900s
```

其他平台 live E2E 当前覆盖 `xhs`、`dy`、`zhihu` 的真实 `search` 链路，检查 JSON、CSV、SQLite 写入，并用同一个 `/tmp` checkpoint 目录重复运行来证明不会追加重复内容：

```bash
MCPRO_OTHER_LIVE_COOKIE_BRIDGE_URL=http://127.0.0.1:8274 \
go test ./tests/e2e -run TestCLIOtherPlatformsLiveSearchWithRealCookie -count=1 -timeout 900s
```

可选覆盖关键词和 SignSrv 地址：

```bash
MCPRO_OTHER_LIVE_COOKIE_BRIDGE_URL=http://127.0.0.1:8274 \
MCPRO_OTHER_LIVE_SIGN_HOST=127.0.0.1 \
MCPRO_OTHER_LIVE_SIGN_PORT=8989 \
MCPRO_XHS_LIVE_KEYWORD=deepseek \
MCPRO_DOUYIN_LIVE_KEYWORD=deepseek \
MCPRO_ZHIHU_LIVE_KEYWORD=deepseek \
go test ./tests/e2e -run TestCLIOtherPlatformsLiveSearchWithRealCookie -count=1 -timeout 900s
```

截至 2026-06-22 本地实测，`xhs`、`dy`、`zhihu` 可用 Cookie 已通过 live E2E；`ks` 和 `wb` 在 Python 与 Go 侧都被当前账号状态阻塞；`tieba` 当前 CookieBridge 未提供可用 Cookie。账号状态恢复后，应先用同样 live E2E 方式补齐这些平台。

新增真实链路修复的聚焦回归：

```bash
go test ./internal/service/douyin ./internal/service/zhihu \
  -run 'TestSearchInfoByKeywordUsesPythonCompatibleSignedParams|TestGetNotesByKeywordSignsPathQueryAndUsesPythonCompatibleHeaders' \
  -count=1

go test ./internal/zhihu -run TestFilterSeenContentObjectsSkipsDuplicateContentIDs -count=1
```

## 11. 排障

- `Fatal error config file`: 确认当前目录或 `config/` 下存在 `config.yml`。
- CookieBridge 获取失败：先检查 `curl $COOKIE_BRIDGE_URL/ping` 和 `/api/status`。
- 签名失败：确认 `signSrvConfig.signSrvHost` / `signSrvConfig.signSrvPort` 指向可用 SignSrv。
- MySQL 连接失败：优先检查 `RELATION_DB_*` 环境变量，确认数据库存在、用户有权限。
- SQLite 文件未生成：确认 `saveDataOption: sqlite` 或 `--save_data_option sqlite`，并检查 `MCPRO_SQLITE_DB_PATH`。
- 没有输出文件：确认 `MCPRO_DATA_DIR`、`crawlerMaxNotesCount`、crawler type 和平台是否支持对应模式。
- checkpoint 没恢复：确认 `--enable_checkpoint` 没被关闭；需要稳定复用同一 `--checkpoint_id` 或同一平台/type/keyword/URL 输入。
- 自动化测试触发外网 guard：说明配置里的平台 endpoint 没有指向本地 mock server。
