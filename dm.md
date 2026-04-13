# 需要决策的设计问题

> 在前端/后端类型对齐过程中发现的需要讨论的问题。
> 审计时间: 2026-04-12

---

## 1. Galgame 详情页响应结构

**现状**: Go 后端返回 `{ wiki: <wiki 原始 JSON>, stats: GalgameStats, isLiked, isFavorited }`

**前端期望**: 扁平结构 `{ id, vndbId, name, banner, ... , likeCount, isLiked, favoriteCount, isFavorited, ... }`

**方案 A**: 后端合并 — Go 后端解析 wiki JSON 并与 stats 合并为扁平对象返回。前端无需改动。

**方案 B**: 前端合并 — 前端收到嵌套结构后在 composable 中 merge。后端无需改动。

**建议**: 方案 A（后端合并），因为前端消费更简洁，且 wiki JSON 结构由 galgame service 控制，后端做一层适配更合理。

---

## 2. User Profile 返回字段数量

**现状**: Go 后端 `UserProfileDetail` 只返回 4 个统计字段（topic_count, reply_count, galgame_count, like_count）

**前端期望**: `UserInfo` 类型有 ~25 个统计字段（各种 count、daily limits 等）

**问题**:
- 很多前端期望的字段需要跨多张表 COUNT 查询（galgameComment, galgamePr, galgameLink, topicPoll 等）
- 部分字段如 `dailyTopicCount`, `dailyGalgameCount` 是当日实时计算值

**方案 A**: 后端扩展 — 在 Go 后端添加所有 25 个统计查询（性能影响大，但前端无需改动）

**方案 B**: 精简前端 — UserInfo 页面只展示后端已有的 4 个计数，其他统计移到独立的管理面板

**方案 C**: 分步加载 — 基础信息先返回，统计数据通过独立的 `/api/user/:uid/stats` 端点异步加载

**建议**: 方案 C，保持首屏加载快，统计数据懒加载。

---

## 3. snake_case vs camelCase 字段命名

**现状**: Go 后端部分 DTO 使用 camelCase JSON tag（如 TopicDetail 的 `likeCount`），部分直接暴露 GORM model 使用 snake_case（如 DocArticle 的 `content_markdown`）

**影响模块**: Doc、Website、Galgame Comment、Search

**方案 A**: 统一所有 Go JSON tag 为 camelCase（前端友好，但需改大量 GORM json tag）

**方案 B**: 统一所有前端类型为 snake_case（与数据库一致，但不符合 JS 惯例）

**方案 C**: 对于直接暴露 GORM model 的端点，添加 response DTO 层做字段映射

**建议**: 方案 C — 只在暴露给前端的端点加 DTO 转换层。内部用 snake_case，对外 camelCase。优先修复 Doc 和 Website 模块（影响面最大）。

---

## 4. Website 模块响应完整度

**现状**: Website 列表返回原始 GORM model（缺少关联数据），详情返回 `{ website, isLiked, isFavorited }` 缺少 category 对象、tags 数组、comments 数组

**前端期望**: WebsiteDetail 需要完整的 category 对象、tags 数组、comments 数组

**建议**: 后端 Website 详情端点需要补充 JOIN 查询：
- Join galgame_website_category 获取 category 对象
- Join galgame_website_tag_relation + galgame_website_tag 获取 tags
- Join galgame_website_comment 获取评论列表
- 合并为扁平响应

---

## 5. Search 端点 JSON tag 缺失

**现状**: `search.go` 中 `searchTopic` 的 row struct 没有 JSON tag，Go 默认序列化为 PascalCase（`ID`, `Title`, `View`...），前端期望 camelCase

**需要**: 给所有 search row struct 加 json tag，并补充缺失字段（section, tag, user 嵌套对象等）

---

## 6. Avatar 上传响应

**现状**: 后端返回 S3 key 字符串

**前端期望**: `{ avatar: string, avatarMin: string }`（缩略图版本）

**问题**: 目前没有图片缩放/缩略图生成。是否需要？

**建议**: 暂时后端返回 `{ avatar: url }` 格式（单字段对象），前端不使用 `avatarMin`。等图片处理库集成后再添加缩略图支持。

---

## 7. 仍使用旧 API 模式的 18 个前端文件

以下文件仍使用 `kungalgameResponseHandler`，需要决定是否迁移：

**Toolset 相关 (7 个)** — 随 toolset 后端完成后迁移
**Register/Forgot (2 个)** — 这些端点走 OAuth，不走 Go 后端，保持现状
**Admin UserCard (1 个)** — 需要迁移
**Unmoe (1 个)** — unmoe 端点未实现，暂不迁移
**Verification Code (1 个)** — 邮件验证码端点走 OAuth 或未实现
**Edit Toolset (2 个)** — 随 toolset 迁移

**建议**: Register/Forgot/Verification Code 保持旧模式（它们调 OAuth 不走 Go 后端）。其余随对应模块迁移。
