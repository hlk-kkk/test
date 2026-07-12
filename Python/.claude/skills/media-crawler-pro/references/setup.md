# 环境检查、安装与服务启动

## 定位项目根目录

项目根目录变量 `$MCR_ROOT` 的确定规则（按优先级）：
1. 如果当前工作目录包含 `MediaCrawlerPro-Python` 子目录，则当前目录即为根目录
2. 否则向上查找父目录，直到找到同时包含 `MediaCrawlerPro-Python` 和 `MediaCrawlerPro-SignSrv` 的目录
3. 如果都找不到，提示用户手动指定 `$MCR_ROOT`

后续所有路径均基于 `$MCR_ROOT`。

## 检查子项目是否存在

```bash
# 自动检测 MCR_ROOT（如果当前目录包含子项目）
if [ -d "./MediaCrawlerPro-Python" ]; then
  MCR_ROOT="$(pwd)"
elif [ -d "../MediaCrawlerPro-Python" ]; then
  MCR_ROOT="$(cd .. && pwd)"
else
  echo "ERROR: 无法自动检测 MCR_ROOT，请手动设置"
  exit 1
fi

# 检查必需子项目
ls "$MCR_ROOT/MediaCrawlerPro-SignSrv/app.py" 2>/dev/null && echo "SignSrv OK" || echo "SignSrv MISSING"
ls "$MCR_ROOT/MediaCrawlerPro-Python/main.py" 2>/dev/null && echo "Python OK" || echo "Python MISSING"
ls "$MCR_ROOT/MediaCrawlerPro-Downloader/DownloadServer/app.py" 2>/dev/null && echo "Downloader OK" || echo "Downloader MISSING"
ls "$MCR_ROOT/MediaCrawlerPro-CookieBridge/server/app.py" 2>/dev/null && echo "CookieBridge OK" || echo "CookieBridge MISSING"
```

## 如果缺少子项目，克隆安装

> **注意**：MediaCrawlerPro 所有仓库均为**私有仓库**，用户的 GitHub 账户必须已被邀请加入组织才能访问。
> 如果 `git clone` 遇到 404 错误，提示用户检查：
> 1. GitHub 账户是否已被邀请加入 MediaCrawlerPro 组织
> 2. 本地 Git/SSH 认证是否正常（`ssh -T git@github.com` 或 `gh auth status`）
>
> 进入对应子项目执行 `uv sync` 即可，uv 会按项目配置自动准备 Python 环境和依赖。

```bash
cd "$MCR_ROOT"

# 先验证仓库访问权限（避免 clone 时遇到 404）
gh repo view MediaCrawlerPro/MediaCrawlerPro-SignSrv --json name 2>&1 | grep -q '"name"' \
  && echo "GitHub 仓库访问权限正常" \
  || echo "ERROR: 无法访问私有仓库，请确认 GitHub 账户已加入 MediaCrawlerPro 组织"

# SignSrv 签名服务（所有爬虫功能的前置依赖）
git clone git@github.com:MediaCrawlerPro/MediaCrawlerPro-SignSrv.git
cd MediaCrawlerPro-SignSrv && uv sync && cd ..

# Python 爬虫（文本数据爬取）
git clone git@github.com:MediaCrawlerPro/MediaCrawlerPro-Python.git
cd MediaCrawlerPro-Python && uv sync && cd ..

# Downloader 下载服务（视频/图片下载）
git clone git@github.com:MediaCrawlerPro/MediaCrawlerPro-Downloader.git
cd MediaCrawlerPro-Downloader/DownloadServer && uv sync && cd ../..

# CookieBridge Cookie 自动同步（可选但推荐）
git clone git@github.com:MediaCrawlerPro/MediaCrawlerPro-CookieBridge.git
cd MediaCrawlerPro-CookieBridge/server && uv sync && cd ../..
```

## 启动依赖服务

所有爬虫功能都依赖 SignSrv 签名服务。CookieBridge 和 DownloadServer 按需启动。

```bash
# 检查并启动 SignSrv（必需）
curl -s http://localhost:8989/signsrv/pong | grep -q "pong" || \
  (cd "$MCR_ROOT/MediaCrawlerPro-SignSrv" && source .venv/bin/activate && python app.py &)

# 检查并启动 CookieBridge（推荐，自动获取 Cookie）
curl -s http://localhost:8274/ping | grep -q "pong" || \
  (cd "$MCR_ROOT/MediaCrawlerPro-CookieBridge" && ./start.sh &)

# 检查并启动 DownloadServer（仅下载场景需要）
curl -s http://localhost:8205/ping | grep -q "pong" || \
  (cd "$MCR_ROOT/MediaCrawlerPro-Downloader/DownloadServer" && source .venv/bin/activate && python app.py &)
```

## Cookie 准备检查

服务启动后、运行爬虫前，确认目标平台有可用的 Cookie：

```bash
# CookieBridge 模式：检查目标平台 Cookie 是否可用
curl -s http://localhost:8274/api/cookies/{platform} | python3 -c "
import sys,json; d=json.load(sys.stdin)
print('Cookie OK') if d.get('isok') and d.get('data',{}).get('cookies') else print('Cookie MISSING - 请在浏览器登录该平台')
"

# Excel 模式：确认 accounts_cookies.xlsx 存在
ls "$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx" && echo "Excel 文件存在" || echo "Excel 文件缺失"
```

> Cookie 不可用？参见 [cookie-preflight.md](cookie-preflight.md) 获取准备指南和故障排查。
