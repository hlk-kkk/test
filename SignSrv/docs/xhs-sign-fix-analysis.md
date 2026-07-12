# XHS 签名 GET 请求 406 问题排查与修复全过程

> 本文档记录了 XHS（小红书）签名服务中 GET 带查询参数请求返回 406 的完整排查过程。
> 适合想深入理解签名机制、逆向分析方法论、以及如何系统性定位 bug 的同学阅读。
> 所有过程仅个人学习JS签名使用，请不要用于商业用途。

## 目录

- [XHS 签名 GET 请求 406 问题排查与修复全过程](#xhs-签名-get-请求-406-问题排查与修复全过程)
  - [目录](#目录)
  - [1. 问题背景](#1-问题背景)
  - [2. 现象确认](#2-现象确认)
  - [3. 排查思路总览](#3-排查思路总览)
  - [4. 第一轮：签名格式对比](#4-第一轮签名格式对比)
    - [思路](#思路)
    - [实验](#实验)
    - [结论](#结论)
  - [5. 第二轮：XYW\_ 格式假说验证](#5-第二轮xyw_-格式假说验证)
    - [背景](#背景)
    - [实验](#实验-1)
    - [结论](#结论-1)
  - [6. 第三轮：x3 payload 交叉实验](#6-第三轮x3-payload-交叉实验)
    - [关键思路](#关键思路)
    - [实验设计与结果](#实验设计与结果)
    - [关键发现](#关键发现)
  - [7. 第四轮：逐字节对比定位 a3\_hash](#7-第四轮逐字节对比定位-a3_hash)
    - [mns0301 payload 结构（144 字节）](#mns0301-payload-结构144-字节)
    - [对比结果](#对比结果)
    - [验证 a3\_hash 是否被服务端校验](#验证-a3_hash-是否被服务端校验)
  - [8. 第五轮：hash 输入穷举——找到根因](#8-第五轮hash-输入穷举找到根因)
    - [a3\_hash 的计算逻辑](#a3_hash-的计算逻辑)
    - [穷举不同输入](#穷举不同输入)
    - [结果](#结果)
    - [根因](#根因)
  - [9. 修复方案](#9-修复方案)
    - [Monkey-Patch 修复 a3\_hash](#monkey-patch-修复-a3_hash)
  - [10. 实战踩坑：URL 编码不一致](#10-实战踩坑url-编码不一致)
    - [排查](#排查)
    - [解决](#解决)
  - [11. 最终修复代码](#11-最终修复代码)
    - [修复后验证结果](#修复后验证结果)
  - [12. 经验总结](#12-经验总结)
    - [方法论](#方法论)
    - [踩过的坑](#踩过的坑)
    - [相关链接](#相关链接)

---

## 1. 问题背景

我们的签名服务使用 [xhshow](https://github.com/Cloxl/xhshow) 开源库（纯 Python 算法还原小红书签名），为 MediaCrawlerPro 爬虫提供 `X-s`、`X-t`、`X-s-common`、`X-b3-traceid` 等请求头签名。

签名服务对 **POST 请求**（如 feed、search、homefeed）和 **GET 无参数请求**（如 selfinfo、user/me）都能正常工作，但对 **GET 带查询参数的请求**（如 `user_posted`、`querytrending`）始终返回 HTTP **406**（Not Acceptable）。

```
GET /api/sns/web/v1/user_posted?num=30&cursor=&user_id=xxx&image_formats=jpg,webp,avif
    &xsec_token=xxx&xsec_source=pc_feed

→ HTTP 406, {"code": -1, "success": false}
```

## 2. 现象确认

先用浏览器抓包获取一组有效的请求头（包含 `x-s`、`x-t` 等），然后做对照实验：

| 测试场景 | 结果 |
|---------|------|
| 浏览器原始签名 → user_posted | **200** ✅ |
| xhshow 签名 → user_posted | **406** ❌ |
| xhshow 签名 → selfinfo（GET 无参数） | **200** ✅ |
| xhshow 签名 → search（POST） | **200** ✅ |
| xhshow 签名 → homefeed（POST） | **200** ✅ |

**结论：问题仅出现在 GET 带查询参数的请求上。**

---

## 3. 排查思路总览

整个排查过程可以分为五轮，每一轮都排除一类可能性，逐步缩小范围：

```
第1轮: 签名外壳 (x0/x2/x-s-common) 有问题？ → 排除
第2轮: 需要 XYW_ 新格式？                   → 排除
第3轮: x3 payload 有问题？                   → 确认！
第4轮: payload 哪个字段有问题？               → a3_hash（bytes 128-143）
第5轮: hash 函数输入有什么不同？              → 找到根因！
```

---

## 4. 第一轮：签名格式对比

### 思路

XYS\_ 签名结构是 `XYS_` + 自定义 Base64 编码的 JSON：

```json
{
  "x0": "4.2.6",       // SDK 版本
  "x1": "xhs-pc-web",  // app 标识
  "x2": "Windows",     // 平台
  "x3": "mns0301_...", // 核心签名 payload
  "x4": ""
}
```

浏览器的 x0 = `"4.3.3"`、x2 = `"Mac OS"`，而 xhshow 默认是 `"4.2.6"` / `"Windows"`。会不会是这些字段被服务端校验了？

### 实验

```python
# 实验 A: 修改 x0=4.3.3, x2 保持 Windows → 406
# 实验 B: 修改 x2=Mac OS, x0 保持 4.2.6 → 406
# 实验 C: 同时修改 x0=4.3.3, x2=Mac OS → 406
# 实验 D: 浏览器 x-s + xhshow x-s-common → 200 ✅
```

### 结论

- **x0/x2 不是问题**：四种组合都返回 406
- **x-s-common 不是问题**：实验 D 证明 xhshow 的 x-s-common 搭配浏览器 x-s 能通过
- **x-t 不是问题**：用不同时间戳的 x-t 搭配浏览器 x-s 也能通过

**问题锁定在 x-s 内部的 x3 payload。**

---

## 5. 第二轮：XYW\_ 格式假说验证

### 背景

[xhshow#104](https://github.com/Cloxl/xhshow/issues/104) 提到浏览器现在使用 `XYW_` 格式签名，数据接口会拒绝 `XYS_` 格式。对应的 [PR#105](https://github.com/Cloxl/xhshow/pull/105) 提供了格式转换方案。

`XYW_` 格式是将 x3 payload 包装在新的 JSON 信封中：

```json
{
  "signSvn": "56",
  "signType": "x2",
  "appId": "xhs-pc-web",
  "signVersion": "1",
  "payload": "<x3 值>"
}
```

然后用标准 Base64 编码，加 `XYW_` 前缀。

### 实验

```python
# 将 xhshow 的 XYS_ 签名转换为 XYW_ 格式后测试
# user_posted (XYW_) → 406 ❌ 更糟了
# selfinfo   (XYW_) → 406 ❌ 原来能用的也坏了
# feed POST  (XYW_) → 406 ❌
```

### 结论

**XYW\_ 格式不是解决方案**（至少在当前 xhshow 的 payload 基础上不行）。问题在 x3 payload 本身。

---

## 6. 第三轮：x3 payload 交叉实验

### 关键思路

既然浏览器的 x-s 能通过，xhshow 的不行，那我把浏览器的 x3 payload 取出来，放进 xhshow 的外壳里，会怎样？

### 实验设计与结果

| # | x3 来源 | 外壳 (x0/x2) | x-t | x-s-common | 结果 |
|---|--------|-------------|-----|-----------|------|
| 0 | 浏览器 | 浏览器 | 浏览器 | 浏览器 | 200 ✅ |
| 1 | xhshow | xhshow | xhshow | xhshow | 406 ❌ |
| **2** | **浏览器** | **xhshow** | **xhshow** | **xhshow** | **200 ✅** |
| 3 | 浏览器 | 浏览器 | xhshow | xhshow | 200 ✅ |
| 6 | 浏览器 | 浏览器 | xhshow | 浏览器 | 200 ✅ |
| 7 | xhshow | xhshow | 浏览器 | xhshow | 406 ❌ |

### 关键发现

**实验 2** 是决定性的：浏览器的 x3 放进 xhshow 的外壳（x0=4.2.6, x2=Windows），搭配 xhshow 的 x-t 和 x-s-common，照样返回 **200**！

这证明：
- 外壳字段（x0/x2）**不影响**
- x-t 时间戳**不影响**
- x-s-common **不影响**
- **只有 x3 payload 的内容决定成败**

---

## 7. 第四轮：逐字节对比定位 a3\_hash

### mns0301 payload 结构（144 字节）

```
偏移        字段              字节数   说明
[0-3]      version_bytes      4      版本标识 [121,104,96,41]
[4-7]      seed               4      随机种子 (LE int32)
[8-15]     timestamp_ms       8      时间戳毫秒 (LE int64)
[16-23]    page_load_ts       8      页面加载时间 (LE int64)
[24-27]    sequence_value     4      序列值 (LE int32)
[28-31]    window_props_len   4      window 属性数 (LE int32)
[32-35]    uri_length         4      content_string 字节长度
[36-43]    md5_xor            8      MD5 前 8 字节 XOR seed_byte
[44]       a1_length          1      a1 cookie 长度
[45-96]    a1_value          52      a1 cookie 值
[97]       app_id_length      1      app 标识长度
[98-107]   app_id            10      "xhs-pc-web"
[108-123]  env_part11        16      环境检测指纹
[124-127]  a3_prefix          4      [2, 97, 51, 16]
[128-143]  a3_hash           16      custom_hash_v2 的输出 XOR seed_byte
```

> **注意**：这里有一个容易遗漏的 `uri_length` 字段（bytes 32-35）。最初的解析器漏掉了它，导致后续所有字段偏移错误，走了弯路。

### 对比结果

用浏览器的参数手动重建 payload，与浏览器原始 payload 逐字节对比：

```
bytes [0-127]:   ✅ 完全匹配（version, seed, ts, a1, app_id, env 全部一致）
bytes [128-143]: ❌ 全部 16 字节不同！
```

**唯一的差异在 a3\_hash 字段！**

### 验证 a3\_hash 是否被服务端校验

```python
# 修改浏览器 payload 的 a3_hash 为全零 → 406
# 修改为随机值 → 406
# 只翻转第一个字节 → 406
# 用 xhshow 算出的错误值替换 → 406
```

**服务端严格校验 a3\_hash，任何偏差都会被拒绝。**

---

## 8. 第五轮：hash 输入穷举——找到根因

### a3\_hash 的计算逻辑

```python
# xhshow 的 build_payload_array 中：
api_path = extract_api_path(string_param)      # 去掉 "?" 和 "{" 后的内容
api_path_bytes = api_path.encode("utf-8")
hex_md5 = hashlib.md5(api_path_bytes).hexdigest()
md5_path_bytes = [int(hex_md5[i:i+2], 16) for i in range(0, 32, 2)]

a3_hash = custom_hash_v2(ts_bytes + md5_path_bytes)
```

其中 `extract_api_path` 的行为：

```python
# POST: "/api/sns/web/v1/feed{\"num\":47}" → "/api/sns/web/v1/feed"  (去掉 "{" 后的)
# GET:  "/api/sns/web/v1/user_posted?num=30&..." → "/api/sns/web/v1/user_posted"  (去掉 "?" 后的)
```

### 穷举不同输入

已知浏览器的正确 a3\_hash 输出，用不同的输入组合喂给 `custom_hash_v2`：

```python
inputs_to_try = [
    ("ts + MD5(api_path)",            ts_bytes + api_md5_bytes),     # xhshow 当前方式
    ("ts + MD5(full_content_string)", ts_bytes + full_md5_bytes),    # 完整 URL 的 MD5
    ("ts + MD5(page_load_ts)",        plt_bytes + api_md5_bytes),
    # ... 更多组合
]
```

### 结果

```
ts + MD5(api_path)            → first4=['0xA5', '0x67', '0x36', '0x8C']  ❌
*** MATCH *** ts + MD5(full_content_string)  → 完全匹配浏览器！             ✅
```

### 根因

xhshow 的 `build_payload_array` 对 **所有请求** 使用 `MD5(api_path)` 计算 a3\_hash，但浏览器的行为是：

| 请求类型 | xhshow 的 a3 输入 | 浏览器的 a3 输入 | 是否一致 |
|---------|------------------|----------------|---------|
| GET 无参数 | MD5("/api/.../selfinfo") | MD5("/api/.../selfinfo") | ✅ 一致 |
| GET 有参数 | MD5("/api/.../user_posted") | MD5("/api/.../user_posted?num=30&...") | ❌ **不一致** |
| POST | MD5("/api/.../feed") | MD5("/api/.../feed") | ✅ 一致 |

对于 GET 无参数，`api_path == content_string`，所以 MD5 相同，碰巧正确。
对于 POST，`extract_api_path` 去掉 `{...}` 后的 JSON body，与浏览器行为一致。
**只有 GET 带参数时**，`extract_api_path` 把 `?` 后的查询参数也去掉了，导致 MD5 不匹配。

---

## 9. 修复方案

### Monkey-Patch 修复 a3\_hash

在签名服务启动时，对 xhshow 的 `CryptoProcessor.build_payload_array` 进行 monkey-patch：

```python
def _patch_xhshow_a3_hash():
    import hashlib
    from xhshow.core.crypto import CryptoProcessor

    _original_build = CryptoProcessor.build_payload_array

    def _patched_build(self, hex_parameter, a1_value, app_identifier="xhs-pc-web",
                       string_param="", timestamp=None, sign_state=None):
        payload = _original_build(self, hex_parameter, a1_value, app_identifier,
                                  string_param, timestamp, sign_state)

        # 仅修复 GET 请求 (content_string 不含 "{")
        # POST 请求的原实现已正确
        if "{" not in string_param:
            correct_md5_hex = hashlib.md5(string_param.encode("utf-8")).hexdigest()
            correct_md5_bytes = [int(correct_md5_hex[i:i+2], 16) for i in range(0, 32, 2)]
            seed_byte = payload[4]
            ts_bytes = payload[8:16]
            correct_a3_hash = self._custom_hash_v2(list(ts_bytes) + correct_md5_bytes)
            for i in range(16):
                payload[128 + i] = correct_a3_hash[i] ^ seed_byte

        return payload

    CryptoProcessor.build_payload_array = _patched_build
```

**判断逻辑**：通过 `"{" not in string_param` 区分 GET 和 POST：
- POST 的 content\_string 形如 `/api/path{"key":"value"}`，包含 `{`
- GET 的 content\_string 形如 `/api/path?key=value`，不包含 `{`

---

## 10. 实战踩坑：URL 编码不一致

修复 a3\_hash 后，本地测试全部通过。但 MediaCrawlerPro-Python 爬虫调用签名服务时仍然失败！

### 排查

爬虫发来的 URI：
```
/api/sns/web/v1/user_posted?...&xsec_token=AB-p6URsgUZZP6OFtBNLhWz6UYAWOtFPIriEP0bbiWnbw=&...
```

注意 `xsec_token` 末尾的 `=` 是 **raw 未编码** 的。

但 xhshow 内部的 `_build_content_string` 会用 `urllib.parse.quote(value, safe=",")` 重编码，把 `=` 变成 `%3D`：

```python
# 爬虫实际请求 URL:  ...xsec_token=AB...w=&xsec_source=pc_feed
# xhshow 签名的 URL: ...xsec_token=AB...w%3D&xsec_source=pc_feed
#                                       ^^^ 不匹配！
```

两个 URL 的 MD5 不同，签名自然无效。

### 解决

对 GET 请求，绕过 xhshow 的高层 API（`sign_headers_get`），直接使用原始 URI 作为 content\_string：

```python
# 不再 parse_qs + rebuild，直接用原始 URI
content_string = req.uri  # 保持原始编码
d_value = hashlib.md5(content_string.encode("utf-8")).hexdigest()
# 调用底层签名方法 ...
```

这样签名的 content\_string 与爬虫实际请求的 URL 完全一致。

---

## 11. 最终修复代码

修改文件：`logic/xhs/xhs_logic.py`

核心改动两处：

1. **模块顶层**：添加 `_patch_xhshow_a3_hash()` 函数并在模块加载时调用，修复 GET 请求的 a3\_hash
2. **`XhsJavascriptSign.sign()` 方法**：GET 请求不再使用 xhshow 的 `sign_headers_get`，而是直接用原始 URI 构建签名

### 修复后验证结果

| 端点 | 修复前 | 修复后 |
|------|-------|-------|
| GET user\_posted（带参数，%3D 编码） | 406 | **200** ✅ |
| GET user\_posted（带参数，raw = 编码） | 406 | **200** ✅ |
| GET selfinfo（无参数） | 200 | **200** ✅ |
| GET querytrending（带参数） | 406 | **200** ✅ |
| POST homefeed | 200 | **200** ✅ |
| POST search | 200 | **200** ✅ |

---

## 12. 经验总结

### 方法论

1. **控制变量实验**：通过交换浏览器/xhshow 的各个签名组件（x-s、x-t、x-s-common），精确定位问题在 x3 payload
2. **逐字节对比**：解码 payload 后逐字段对比，发现唯一差异在 a3\_hash
3. **穷举输入**：对 hash 函数尝试不同输入组合，快速找到正确输入
4. **端到端验证**：不仅本地测试，还要通过实际爬虫调用验证

### 踩过的坑

| 坑 | 教训 |
|----|------|
| 解析 payload 时漏掉 `uri_length` 字段 | 反序列化二进制结构时，每个字段都要对照源码确认偏移 |
| 修改代码后忘记重启签名服务 | 修改了服务端代码后必须重启进程！ |
| `parse_qs` + `quote` 改变了 URL 编码 | 签名的 content\_string 必须与实际请求 URL 完全一致，不能有任何编码差异 |
| 以为 XYW\_ 格式是解决方案 | 不要盲目相信未经验证的 PR，要自己测试 |

### 相关链接

- xhshow 库：https://github.com/Cloxl/xhshow
- 相关 Issue：https://github.com/Cloxl/xhshow/issues/104
- 相关 PR：https://github.com/Cloxl/xhshow/pull/105
