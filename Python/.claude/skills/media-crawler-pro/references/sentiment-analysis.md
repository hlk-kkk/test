# 评论舆情分析流程

当用户要求分析评论或做舆情分析时，按以下步骤执行：

## 步骤 1: 爬取评论数据

确保启用评论爬取，执行 detail 模式：
```bash
ENABLE_GET_COMMENTS=true DB_TYPE=sqlite ACCOUNT_POOL_SAVE_TYPE=cookie_bridge \
  uv run --project "$MCR_ROOT/MediaCrawlerPro-Python" \
  "$MCR_ROOT/MediaCrawlerPro-Python/main.py" \
  --platform {platform} --type detail --urls "{content_url}"
```

## 步骤 2: 查询评论数据

```bash
# 获取评论（按点赞数排序，取高赞评论）
sqlite3 -header -column "$DB_PATH" "SELECT content, nickname, like_count, ip_location, create_time FROM {comment_table} WHERE {id_field}='{content_id}' ORDER BY CAST(like_count AS INTEGER) DESC LIMIT 100;"
```

> **列名差异注意**：tieba/zhihu 评论表的昵称字段为 `user_nickname`（非 `nickname`），时间字段为 `publish_time`（非 `create_time`）；bilibili 评论表无 `ip_location` 字段。

## 步骤 3: 分析维度

拿到评论数据后，从以下维度分析：
1. **情感分布**: 正面/中性/负面评论比例
2. **核心话题**: 评论中讨论最多的话题/关键词
3. **高赞观点**: 点赞数最高的评论代表的主流态度
4. **地域分布**: 基于 `ip_location` 的评论者地域分布
5. **争议性**: 是否存在明显对立的观点
6. **意见领袖**: 高互动评论者的影响力

## 步骤 4: 输出报告

```json
{
  "总评论数": 150,
  "情感分布": {"正面": "62%", "中性": "25%", "负面": "13%"},
  "核心话题": ["话题1", "话题2", "话题3"],
  "高赞观点摘要": ["观点1 (520赞)", "观点2 (380赞)"],
  "争议性评估": "低/中/高",
  "综合结论": "..."
}
```

## 数据量控制策略

- 评论 < 100 条：全量分析
- 100-500 条：按点赞数取 TOP 100 + 随机采样 50 条
- \> 500 条：TOP 50 高赞 + 随机 50 条 + 最新 20 条
