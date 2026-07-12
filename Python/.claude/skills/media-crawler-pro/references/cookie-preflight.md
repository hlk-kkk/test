# Cookie 准备与验证指南

所有爬虫操作都需要有效的平台登录 Cookie。没有 Cookie 会报错："账号池中没有可用的账号"。

## Cookie 方案选择

```
你的场景是？
├── 想自动化、不想手动复制粘贴 → CookieBridge（推荐）
│     前置：Chrome + Extension + CookieBridge Server
│     优势：登录即可，Cookie 自动刷新
│
├── 只是快速测试、不想装额外服务 → Excel
│     前置：Cookie-Editor 浏览器插件 + accounts_cookies.xlsx
│     注意：Cookie 过期需手动更新
│
└── 生产环境、需要状态持久化 → MySQL
      前置：MySQL 数据库 + 表 crawler_cookies_account
      注意：Cookie 过期需手动更新
```

## 各平台 Cookie 获取要点

| 平台 | 关键 Cookie | 获取方式 | 特殊注意 |
|------|------------|---------|---------|
| 小红书 | `web_session` | 在 xiaohongshu.com 登录 | 无特殊要求 |
| 抖音 | `sessionid` / `sessionid_ss` / `sid_tt` | 在 douyin.com 登录 | 任一存在即可 |
| 快手 | `passToken` | 在 kuaishou.com 登录 | 无特殊要求 |
| B站 | `SESSDATA` / `DedeUserID` | 在 bilibili.com 登录 | 任一存在即可 |
| **微博** | `SSOLoginState` / `SUB` | **必须从 H5 版 m.weibo.cn 获取** | PC 版 weibo.com 的 Cookie 不可用，需 F12 切手机模式刷新 |
| 贴吧 | `STOKEN` / `PTOKEN` | 在 tieba.baidu.com 登录 | 任一存在即可 |
| **知乎** | `z_c0` | 在 zhihu.com 登录 | **必须在搜索结果页获取**（如 zhihu.com/search?q=xxx），首页的 Cookie 搜索接口不可用 |

## 方案一：CookieBridge 快速准备

### 前置条件

- CookieBridge Server 已启动：`curl -s http://localhost:8274/ping | grep pong`
- Chrome Extension 已安装并连接（见 [cookie-bridge.md](cookie-bridge.md) 的安装步骤）

### 步骤

1. 在 Chrome 浏览器中打开目标平台并登录
2. 等待 Extension 自动同步（通常 5-10 秒）
3. 验证：

```bash
# 检查 Extension 连接状态
curl -s http://localhost:8274/api/accounts | python3 -c "
import sys,json
d=json.load(sys.stdin)
accounts=d.get('data',{}).get('accounts',[])
if not accounts: print('FAIL: 无 Extension 连接，请检查 Chrome Extension 安装')
else:
    for a in accounts:
        connected='在线' if a.get('connected') else '离线'
        platforms=', '.join(f\"{k}({'有' if v.get('has_cookies') else '无'})\" for k,v in a.get('platforms',{}).items())
        print(f'  [{connected}] {a.get(\"client_id\",\"\")[:8]}... -> {platforms}')
"

# 检查目标平台 Cookie 是否可用
curl -s http://localhost:8274/api/cookies/{platform} | python3 -c "
import sys,json
d=json.load(sys.stdin)
if d.get('isok') and d.get('data',{}).get('cookies'):
    src=d['data'].get('source','unknown')
    print(f'OK: Cookie 可用 (来源: {src})')
else:
    print(f'FAIL: Cookie 不可用，请在浏览器登录该平台')
"
```

## 方案二：Excel 手动配置

### 步骤

1. 安装 Chrome 插件 [Cookie-Editor](https://chromewebstore.google.com/detail/cookie-editor/hlkenndednhfkekhgcdicdfddnkalmdm)
2. 在浏览器登录目标平台（注意微博和知乎的特殊要求）
3. 点击 Cookie-Editor → Export → **Header String** 格式复制
4. 打开 `$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx`
5. 找到对应平台的 sheet（sheet 名 = 平台 ID：xhs / dy / ks / bili / wb / tieba / zhihu）
6. 在 cookies 列（C 列）粘贴完整的 Cookie 字符串

```python
# 或通过代码写入
import openpyxl
wb = openpyxl.load_workbook('$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx')
ws = wb['{platform}']  # sheet 名 = 平台 ID
ws['A2'] = 1                    # id
ws['B2'] = 'account_1'         # account_name
ws['C2'] = 'cookies_string'    # cookies（Cookie-Editor 复制的完整字符串）
ws['D2'] = '{platform}'        # platform_name
ws['E2'] = 'normal'            # status
wb.save('$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx')
```

### 验证

```bash
python3 -c "
import openpyxl
wb=openpyxl.load_workbook('$MCR_ROOT/MediaCrawlerPro-Python/config/accounts_cookies.xlsx')
p='{platform}'
if p in wb.sheetnames:
    ws=wb[p]; has=any(ws.cell(r,3).value for r in range(2,ws.max_row+1))
    print(f'OK: Excel 中 {p} 有 Cookie') if has else print(f'FAIL: Excel 中 {p} 的 cookies 列为空')
else: print(f'FAIL: Excel 中无 {p} sheet')
"
```

## Cookie 有效性一键检查

```bash
# CookieBridge 模式
curl -s http://localhost:8274/api/cookies/{platform} | python3 -c "import sys,json; d=json.load(sys.stdin); print('OK: Cookie 可用') if d.get('isok') and d.get('data',{}).get('cookies') else print('FAIL: Cookie 不可用')"

# 或使用增强版 preflight 脚本
bash "$MCR_ROOT/MediaCrawlerPro-Python/scripts/preflight.sh" {platform}
```

## 常见 Cookie 问题排查

### 1. "账号池中没有可用的账号"

**原因**: 账号池加载时未找到任何有效 Cookie。

**排查**:
- CookieBridge 模式：`curl -s http://localhost:8274/api/accounts` → 检查目标平台 `has_cookies` 是否为 true
- Excel 模式：打开 `accounts_cookies.xlsx` → 检查对应 sheet 的 cookies 列是否有值
- 环境变量：`echo $ACCOUNT_POOL_SAVE_TYPE` → 确认与实际 Cookie 来源一致

### 2. CookieBridge 显示有 Cookie 但爬虫仍失败

**可能原因**: Cookie 已过期但仍在缓存中。

**排查**:
- 检查 API 返回的 `source` 字段：`stale_cache` 表示过期缓存
- 在浏览器中刷新目标平台页面，让 Extension 重新同步
- 确认浏览器中仍处于登录状态

### 3. 微博爬虫 Cookie 无效

**原因**: 使用了 PC 版 weibo.com 的 Cookie。

**解决**: 必须使用 H5 版本 m.weibo.cn 的 Cookie。方法：PC 登录后 → F12 → 切换手机模式 → 访问 m.weibo.cn → 此时获取的 Cookie 才能用于爬虫。

### 4. 知乎搜索失败但详情正常

**原因**: Cookie 来源页面不对。

**解决**: 知乎的 Cookie 必须在搜索结果页（如 `zhihu.com/search?q=xxx`）获取。首页获取的 Cookie 缺少搜索所需的参数。

### 5. 小红书 "xsec_token 无效"

**说明**: 这不是 Cookie 问题，而是 URL 参数过期。

**解决**: xsec_token 有时效性，需要在浏览器打开目标页面，从地址栏复制包含最新 xsec_token 的完整 URL。

### 6. CookieBridge 无法连接

**排查顺序**:
1. Server 是否运行？`curl -s http://localhost:8274/ping`
2. Extension 是否安装？检查 `chrome://extensions/` 中 CookieBridge 是否启用
3. Extension 是否连接？查看 Extension Popup 面板中 Server 状态
4. 查看 Server 日志是否有 WebSocket 连接相关错误
