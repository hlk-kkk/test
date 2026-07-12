# test

这是一个 MediaCrawlerPro 相关项目的聚合快照仓库，集中保存 7 个上游仓库的工作树内容，便于统一查看和研究。

## 项目来源

| 本仓库目录 | 上游仓库 | 快照提交 |
| --- | --- | --- |
| `Golang/` | [MediaCrawlerPro-Golang](https://github.com/MediaCrawlerPro/MediaCrawlerPro-Golang) | `b87bded` |
| `Python/` | [MediaCrawlerPro-Python](https://github.com/MediaCrawlerPro/MediaCrawlerPro-Python) | `6e5b419` |
| `ContentRemixAgent/` | [MediaCrawlerPro-ContentRemixAgent](https://github.com/MediaCrawlerPro/MediaCrawlerPro-ContentRemixAgent) | `3c96b72` |
| `PicTacticAgent/` | [PicTacticAgent](https://github.com/MediaCrawlerPro/PicTacticAgent) | `fdd9379` |
| `SignSrv/` | [MediaCrawlerPro-SignSrv](https://github.com/MediaCrawlerPro/MediaCrawlerPro-SignSrv) | `d8926b9` |
| `CookieBridge/` | [MediaCrawlerPro-CookieBridge](https://github.com/MediaCrawlerPro/MediaCrawlerPro-CookieBridge) | `bcecae9` |
| `Downloader/` | [MediaCrawlerPro-Downloader](https://github.com/MediaCrawlerPro/MediaCrawlerPro-Downloader) | `352ce78` |

## 使用方式

```bash
git clone https://github.com/zhuy3075-ui/test.git
cd test
```

进入所需的子目录，按照该项目自己的 `README` 安装依赖和运行。各项目相互独立，本聚合仓库不提供统一的构建或启动命令。

## 仓库结构

```text
test/
├── ContentRemixAgent/
├── CookieBridge/
├── Downloader/
├── Golang/
├── PicTacticAgent/
├── Python/
└── SignSrv/
```

## 验证说明

聚合前已核对 7 个上游仓库的远端地址、默认分支和工作树状态。本仓库只保存工作树文件，不包含各上游仓库的 `.git` 元数据，也未上传依赖缓存目录。

## 安全与许可证

- 发布前已检查常见私钥、GitHub 令牌、AWS Access Key 和 OpenAI 风格密钥特征，未发现匹配项。
- 请勿提交个人账号 Cookie、API Key、访问令牌或 `.env` 文件。
- 各子项目保留各自的 `LICENSE` 文件；版权、许可条件和使用限制以对应子目录及上游仓库为准。

