# Docker操作完整日志实施总结

## 概述

为所有Docker Registry操作（ListTags、GetManifest、GetBlob、CheckBlob、GetReferrers）添加了完整的日志记录，包括成功日志和失败日志。

## 实施内容

### 1. ListTags - 获取标签列表

**接口**: `GET /v2/<name>/tags/list`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、标签数量、用户信息、IP
- ✅ **失败日志**: 记录获取标签列表失败的错误信息

**日志示例**:
```json
// 成功日志
{
  "action": "list_tags",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "tag_count": 5
  }
}

// 失败日志
{
  "action": "list_tags",
  "resource": "image",
  "status": "error",
  "details": {
    "repository": "test-project/test-image",
    "error": "failed to list tags: ..."
  }
}
```

### 2. GetManifest - 获取Manifest

**接口**: `GET /v2/<name>/manifests/<reference>`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、引用、digest、用户信息、IP
- ✅ **失败日志**: 记录获取manifest失败的错误信息（包括404和500错误）

**日志示例**:
```json
// 成功日志
{
  "action": "get_manifest",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "reference": "latest",
    "digest": "sha256:abc123..."
  }
}

// 失败日志（404）
{
  "action": "get_manifest",
  "resource": "image",
  "status": "error",
  "details": {
    "repository": "test-project/test-image",
    "reference": "latest",
    "not_found": true,
    "error": "manifest not found"
  }
}
```

### 3. GetBlob - 获取Blob

**接口**: `GET /v2/<name>/blobs/<digest>`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、digest、大小、用户信息、IP
- ✅ **失败日志**: 记录获取blob失败的错误信息（包括404和500错误）

**日志示例**:
```json
// 成功日志
{
  "action": "get_blob",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "digest": "sha256:abc123...",
    "size": 1048576
  }
}

// 失败日志（404）
{
  "action": "get_blob",
  "resource": "image",
  "status": "error",
  "details": {
    "repository": "test-project/test-image",
    "digest": "sha256:abc123...",
    "not_found": true,
    "error": "blob not found"
  }
}
```

### 4. CheckBlob - 检查Blob是否存在

**接口**: `HEAD /v2/<name>/blobs/<digest>`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、digest、大小、用户信息、IP
- ✅ **失败日志**: 记录检查blob失败的错误信息（500错误）

**日志示例**:
```json
// 成功日志
{
  "action": "check_blob",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "digest": "sha256:abc123...",
    "size": 1048576
  }
}

// 失败日志
{
  "action": "check_blob",
  "resource": "image",
  "status": "error",
  "details": {
    "repository": "test-project/test-image",
    "digest": "sha256:abc123...",
    "error": "failed to check blob: ..."
  }
}
```

### 5. GetReferrers - 获取引用列表

**接口**: `GET /v2/<name>/manifests/<reference>/referrers`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、digest、引用数量、用户信息、IP
- ✅ **失败日志**: 记录获取referrers失败的错误信息

**日志示例**:
```json
// 成功日志
{
  "action": "get_referrers",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "digest": "sha256:abc123...",
    "count": 3
  }
}

// 失败日志
{
  "action": "get_referrers",
  "resource": "image",
  "status": "error",
  "details": {
    "repository": "test-project/test-image",
    "digest": "sha256:abc123...",
    "error": "failed to get referrers: ..."
  }
}
```

## 实现细节

### 日志记录函数

使用 `audit.Record()` 和 `audit.RecordError()` 函数记录日志：

```go
// 成功日志
audit.Record(ctx.Request.Context(), "list_tags", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
    "repository": project,
    "tag_count":  len(paginatedTags),
})

// 失败日志
audit.RecordError(ctx.Request.Context(), "list_tags", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
    "repository": project,
})
```

### 用户信息提取

所有日志都包含用户信息（如果可识别）：

```go
var userID *uuid.UUID
if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
    if userUUID, ok := userIDVal.(uuid.UUID); ok {
        userID = &userUUID
    }
}
```

### 错误类型区分

对于可能返回404的操作（GetManifest、GetBlob），在失败日志中标记 `not_found`：

```go
audit.RecordError(ctx.Request.Context(), "get_manifest", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
    "repository": project,
    "reference":  reference,
    "not_found":  err == registry.ErrManifestNotFound,
})
```

## 日志记录策略

### 成功日志

- ✅ **所有操作**: 记录成功执行的详细信息
- ✅ **包含信息**: 用户ID、IP、操作详情、时间戳
- ✅ **异步记录**: 不阻塞主流程

### 失败日志

- ✅ **所有操作**: 记录失败的错误信息
- ✅ **包含信息**: 用户ID、IP、错误详情、错误类型
- ✅ **错误区分**: 区分404（资源不存在）和500（服务器错误）

### 权限拒绝日志

- ✅ **所有操作**: 通过 `checkProjectPermission` 统一记录
- ✅ **包含信息**: 错误码、错误消息、权限类型、项目名称

## 最终状态

### 完整日志覆盖

| 操作 | 权限拒绝 | 成功 | 失败 | 状态 |
|------|---------|------|------|------|
| ListTags | ✅ | ✅ | ✅ | ✅ 完整 |
| GetManifest | ✅ | ✅ | ✅ | ✅ 完整 |
| GetBlob | ✅ | ✅ | ✅ | ✅ 完整 |
| CheckBlob | ✅ | ✅ | ✅ | ✅ 完整 |
| GetReferrers | ✅ | ✅ | ✅ | ✅ 完整 |

### 日志记录特点

1. **完整性**: 所有操作都有成功和失败日志
2. **详细信息**: 包含用户、IP、操作详情等关键信息
3. **错误区分**: 区分不同类型的错误（404 vs 500）
4. **性能优化**: 异步记录，不阻塞主流程
5. **自动清理**: 支持自动清理过期日志

## 使用场景

### 场景1: 问题排查

**需求**: 排查某个镜像拉取失败的原因

**查询**: 在日志中搜索 `"action":"get_manifest"` 和 `"status":"error"`，查看错误详情

### 场景2: 审计追踪

**需求**: 查看某个用户的所有操作

**查询**: 在日志中搜索 `"user_id":"xxx"`，查看该用户的所有操作记录

### 场景3: 统计分析

**需求**: 统计拉取操作的频率和成功率

**分析**: 解析日志中的 `action`、`status` 字段进行统计

### 场景4: 安全监控

**需求**: 监控异常操作（如大量失败、异常时间操作等）

**监控**: 基于日志中的 `status`、`timestamp`、`user_id` 等字段进行监控

## 总结

- ✅ **完整实现**: 所有操作都有完整的日志记录（成功+失败）
- ✅ **详细信息**: 包含用户、IP、操作详情等关键信息
- ✅ **错误区分**: 区分不同类型的错误
- ✅ **性能优化**: 异步记录，不影响主服务性能
- ✅ **自动清理**: 支持自动清理过期日志

系统日志记录已完整实现，满足生产环境的审计、问题排查和安全监控需求。
