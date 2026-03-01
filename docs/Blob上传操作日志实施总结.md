# Blob上传操作日志实施总结

## 概述

为所有Blob上传相关操作添加了完整的日志记录，包括成功日志和失败日志，确保所有Docker Registry操作都有完整的审计追踪。

## 实施内容

### 1. InitiateBlobUpload - 初始化Blob上传

**接口**: `POST /v2/<name>/blobs/uploads/`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、上传ID、挂载信息、用户信息、IP
- ✅ **失败日志**: 记录初始化上传失败的错误信息

**日志示例**:
```json
// 成功日志
{
  "action": "initiate_blob_upload",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "upload_id": "uuid",
    "mount": "sha256:abc123...",
    "from": "source-repo"
  }
}

// 失败日志
{
  "action": "initiate_blob_upload",
  "resource": "image",
  "status": "error",
  "details": {
    "repository": "test-project/test-image",
    "error": "failed to initiate upload: ..."
  }
}
```

### 2. UploadBlobChunk - 上传Blob分片

**接口**: `PATCH /v2/<name>/blobs/uploads/<uuid>`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、上传ID、偏移量、大小、用户信息、IP
- ✅ **失败日志**: 记录上传分片失败的错误信息

**注意**: 分片上传操作很频繁，会产生较多日志，但为了完整性和问题排查需要，仍然记录所有操作。

**日志示例**:
```json
// 成功日志
{
  "action": "upload_blob_chunk",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "upload_id": "uuid",
    "offset": 0,
    "new_offset": 1048576,
    "size": 1048576
  }
}
```

### 3. CompleteBlobUpload - 完成Blob上传

**接口**: `PUT /v2/<name>/blobs/uploads/<uuid>?digest=<digest>`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、上传ID、digest、大小、模式、用户信息、IP
- ✅ **失败日志**: 记录完成上传失败的错误信息

**支持模式**:
- Monolithic模式：一次性上传完整blob
- 分片上传模式：分多次上传blob

**日志示例**:
```json
// 成功日志（monolithic模式）
{
  "action": "complete_blob_upload",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "upload_id": "uuid",
    "digest": "sha256:abc123...",
    "size": 1048576,
    "mode": "monolithic"
  }
}
```

### 4. GetBlobUploadStatus - 获取上传状态

**接口**: `GET /v2/<name>/blobs/uploads/<uuid>`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、上传ID、大小、用户信息、IP
- ✅ **失败日志**: 记录获取上传状态失败的错误信息

**日志示例**:
```json
// 成功日志
{
  "action": "get_blob_upload_status",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "upload_id": "uuid",
    "size": 1048576
  }
}
```

### 5. CancelBlobUpload - 取消Blob上传

**接口**: `DELETE /v2/<name>/blobs/uploads/<uuid>`

**日志记录**:
- ✅ **权限拒绝日志**: 通过 `checkProjectPermission` 记录
- ✅ **成功日志**: 记录仓库、上传ID、用户信息、IP
- ✅ **失败日志**: 记录取消上传失败的错误信息

**日志示例**:
```json
// 成功日志
{
  "action": "cancel_blob_upload",
  "resource": "image",
  "status": "success",
  "details": {
    "repository": "test-project/test-image",
    "upload_id": "uuid"
  }
}
```

## 实现细节

### 日志记录函数

使用 `audit.Record()` 和 `audit.RecordError()` 函数记录日志：

```go
// 成功日志
audit.Record(ctx.Request.Context(), "initiate_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
    "repository": project,
    "upload_id":  info.UUID,
    "mount":       mount,
    "from":        from,
})

// 失败日志
audit.RecordError(ctx.Request.Context(), "initiate_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
    "repository": project,
    "error":      err.Error(),
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

### 上传模式区分

对于CompleteBlobUpload，在日志中标记上传模式：

```go
audit.Record(ctx.Request.Context(), "complete_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
    "repository": project,
    "upload_id":   uploadID,
    "digest":      digest,
    "size":        size,
    "mode":        "monolithic", // 或 "chunked"
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
- ✅ **详细错误**: 记录完整的错误信息，便于问题排查

### 权限拒绝日志

- ✅ **所有操作**: 通过 `checkProjectPermission` 统一记录
- ✅ **包含信息**: 错误码、错误消息、权限类型、项目名称

## 最终状态

### 完整日志覆盖

| 操作 | 权限拒绝 | 成功 | 失败 | 状态 |
|------|---------|------|------|------|
| InitiateBlobUpload | ✅ | ✅ | ✅ | ✅ 完整 |
| UploadBlobChunk | ✅ | ✅ | ✅ | ✅ 完整 |
| CompleteBlobUpload | ✅ | ✅ | ✅ | ✅ 完整 |
| GetBlobUploadStatus | ✅ | ✅ | ✅ | ✅ 完整 |
| CancelBlobUpload | ✅ | ✅ | ✅ | ✅ 完整 |

### 日志记录特点

1. **完整性**: 所有操作都有成功和失败日志
2. **详细信息**: 包含用户、IP、操作详情等关键信息
3. **模式区分**: 区分不同的上传模式（monolithic vs chunked）
4. **性能优化**: 异步记录，不阻塞主流程
5. **自动清理**: 支持自动清理过期日志

## 使用场景

### 场景1: 问题排查

**需求**: 排查某个Blob上传失败的原因

**查询**: 在日志中搜索 `"action":"complete_blob_upload"` 和 `"status":"error"`，查看错误详情

### 场景2: 审计追踪

**需求**: 查看某个用户的所有上传操作

**查询**: 在日志中搜索 `"user_id":"xxx"` 和 `"action":"upload_blob_chunk"`，查看该用户的所有上传操作

### 场景3: 性能分析

**需求**: 分析上传操作的频率和成功率

**分析**: 解析日志中的 `action`、`status` 字段进行统计

### 场景4: 安全监控

**需求**: 监控异常上传行为（如大量失败、异常时间上传等）

**监控**: 基于日志中的 `status`、`timestamp`、`user_id` 等字段进行监控

## 注意事项

### 日志量考虑

1. **UploadBlobChunk**: 分片上传操作很频繁，会产生较多日志
   - 建议：如果日志量过大，可以考虑只记录失败日志，或使用采样记录
   - 当前实现：为了完整性和问题排查需要，记录所有操作

2. **其他操作**: 频率适中，日志量可控

### 性能影响

- ✅ **异步记录**: 所有日志都是异步记录，不阻塞主流程
- ✅ **轻量级**: 日志记录是轻量级操作，对性能影响很小

## 总结

### 实施效果

- ✅ **完整实现**: 所有Blob上传操作都有完整的日志记录（成功+失败）
- ✅ **详细信息**: 包含用户、IP、操作详情等关键信息
- ✅ **模式区分**: 区分不同的上传模式
- ✅ **性能优化**: 异步记录，不影响主服务性能
- ✅ **自动清理**: 支持自动清理过期日志

### 改进前后对比

**改进前**:
- ❌ 只有权限拒绝日志
- ❌ 无法追踪上传操作
- ❌ 无法进行上传统计分析

**改进后**:
- ✅ 成功和失败都有日志
- ✅ 可以完整追踪上传历史
- ✅ 支持上传统计分析
- ✅ 满足审计需求

### 最终状态

系统现在对所有Docker Registry操作都有完整的日志记录：
- ✅ 拉取操作：ListTags、GetManifest、GetBlob、CheckBlob、GetReferrers
- ✅ 推送操作：PutManifest、InitiateBlobUpload、UploadBlobChunk、CompleteBlobUpload
- ✅ 删除操作：DeleteManifest、DeleteBlob
- ✅ 管理操作：GetBlobUploadStatus、CancelBlobUpload
- ✅ 所有操作：权限拒绝日志

系统日志记录已完善，满足生产环境的审计、问题排查和安全监控需求。
