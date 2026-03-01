# Docker操作检查报告

## 概述

本报告检查了所有Docker Registry API操作，包括拉取、推送、删除等，确保权限检查和项目处理逻辑正确。

## 操作分类

### 1. 拉取操作（Pull）

#### 1.1 ListTags - 列出标签
**接口**: `GET /v2/<name>/tags/list`

**权限检查**:
- ✅ 检查 `pull` 权限
- ✅ PAT token需要 `read` scope
- ✅ JWT token继承用户所有权限

**功能**:
- 返回标签列表
- 包含标签大小、digest、推送时间、推送用户等信息
- 支持分页

**状态**: ✅ 正常

#### 1.2 GetManifest - 获取Manifest
**接口**: `GET/HEAD /v2/<name>/manifests/<reference>`

**权限检查**:
- ✅ 检查 `pull` 权限
- ✅ PAT token需要 `read` scope
- ✅ JWT token继承用户所有权限

**功能**:
- 返回manifest原始数据
- 设置正确的Content-Type和Digest响应头

**状态**: ✅ 正常

#### 1.3 GetBlob - 获取Blob
**接口**: `GET /v2/<name>/blobs/<digest>`

**权限检查**:
- ✅ 检查 `pull` 权限
- ✅ PAT token需要 `read` scope
- ✅ JWT token继承用户所有权限

**功能**:
- 返回blob数据流
- 支持Range请求（断点续传）

**状态**: ✅ 正常

#### 1.4 CheckBlob - 检查Blob
**接口**: `HEAD /v2/<name>/blobs/<digest>`

**权限检查**:
- ✅ 检查 `pull` 权限
- ✅ PAT token需要 `read` scope
- ✅ JWT token继承用户所有权限

**功能**:
- 返回blob大小和digest
- 不返回实际数据

**状态**: ✅ 正常

### 2. 推送/更新操作（Push）

#### 2.1 PutManifest - 推送/更新Manifest
**接口**: `PUT /v2/<name>/manifests/<reference>`

**权限检查**:
- ✅ 检查 `push` 权限
- ✅ PAT token需要 `write` scope
- ✅ JWT token继承用户所有权限

**项目处理**:
- ✅ 自动创建项目（如果不存在）
- ✅ 更新项目统计（镜像数量、存储使用量）
- ✅ 触发Webhook推送事件

**状态**: ✅ 正常

#### 2.2 InitiateBlobUpload - 初始化Blob上传
**接口**: `POST /v2/<name>/blobs/uploads/`

**权限检查**:
- ✅ 检查 `push` 权限
- ✅ PAT token需要 `write` scope
- ✅ JWT token继承用户所有权限

**项目处理**:
- ✅ 自动创建项目（如果不存在）

**功能**:
- 支持跨仓库挂载（mount参数）
- 支持单次上传（monolithic upload）
- 支持分片上传初始化

**状态**: ✅ 正常

#### 2.3 CompleteBlobUpload - 完成Blob上传
**接口**: `PUT /v2/<name>/blobs/uploads/<uuid>?digest=<digest>`

**权限检查**:
- ✅ 检查 `push` 权限
- ✅ PAT token需要 `write` scope
- ✅ JWT token继承用户所有权限

**功能**:
- 完成blob上传
- 支持在PUT请求中追加数据

**状态**: ✅ 正常

#### 2.4 UploadBlobChunk - 上传Blob分片
**接口**: `PATCH /v2/<name>/blobs/uploads/<uuid>`

**权限检查**:
- ✅ 检查 `push` 权限
- ✅ PAT token需要 `write` scope
- ✅ JWT token继承用户所有权限

**功能**:
- 上传blob分片数据
- 支持断点续传

**状态**: ✅ 正常

### 3. 删除操作（Delete）

#### 3.1 DeleteManifest - 删除Manifest
**接口**: `DELETE /v2/<name>/manifests/<reference>`

**权限检查**:
- ✅ 检查 `delete` 权限
- ✅ PAT token需要 `delete` scope
- ✅ JWT token继承用户所有权限

**项目处理**:
- ✅ 删除后更新项目统计（镜像数量、存储使用量）
- ✅ 触发Webhook删除事件

**功能**:
- 删除指定的manifest
- 记录删除日志（用户ID、用户名、IP等）

**状态**: ✅ 正常

#### 3.2 DeleteBlob - 删除Blob
**接口**: `DELETE /v2/<name>/blobs/<digest>`

**权限检查**:
- ✅ 检查 `delete` 权限
- ✅ PAT token需要 `delete` scope
- ✅ JWT token继承用户所有权限

**项目处理**:
- ✅ 删除后更新项目统计（存储使用量）

**功能**:
- 删除指定的blob
- 重新计算项目的存储使用量

**状态**: ✅ 已修复

### 4. 其他操作

#### 4.1 CancelBlobUpload - 取消Blob上传
**接口**: `DELETE /v2/<name>/blobs/uploads/<uuid>`

**权限检查**:
- ✅ 检查 `push` 权限
- ✅ PAT token需要 `write` scope
- ✅ JWT token继承用户所有权限

**状态**: ✅ 正常

#### 4.2 GetBlobUploadStatus - 获取上传状态
**接口**: `GET /v2/<name>/blobs/uploads/<uuid>`

**权限检查**:
- ✅ 检查 `push` 权限
- ✅ PAT token需要 `write` scope
- ✅ JWT token继承用户所有权限

**状态**: ✅ 已修复

#### 4.3 GetReferrers - 获取引用列表
**接口**: `GET /v2/<name>/manifests/<reference>/referrers`

**权限检查**:
- ✅ 检查 `pull` 权限
- ✅ PAT token需要 `read` scope
- ✅ JWT token继承用户所有权限

**状态**: ✅ 正常

## 发现的问题（已修复）

### 问题1: DeleteBlob后未更新项目统计 ✅ 已修复

**位置**: `src/modules/registry/controller/registry_controller.go:1267-1289`

**问题**: 删除blob后没有更新项目的存储使用量统计。

**影响**: 项目统计中的存储使用量可能不准确。

**修复**: 在删除blob后，重新计算项目的存储使用量并更新。

### 问题2: GetBlobUploadStatus缺少权限检查 ✅ 已修复

**位置**: `src/modules/registry/controller/registry_controller.go:1293-1305`

**问题**: 获取上传状态时没有检查权限。

**影响**: 未授权用户可能查看上传状态。

**修复**: 添加 `push` 权限检查。

## 权限检查总结

### PAT Token权限映射

| 操作类型 | 所需权限 | PAT Scope | 检查函数 |
|---------|---------|-----------|---------|
| 拉取 | `pull` | `read` | `HasScope(ctx, "read")` |
| 推送 | `push` | `write` | `HasScope(ctx, "write")` |
| 删除 | `delete` | `delete` | `HasScope(ctx, "delete")` |

### JWT Token权限

- JWT token继承用户所有权限
- 不需要检查scopes
- 通过 `checkProjectPermission` 中的项目所有权检查

## 项目处理总结

### 自动创建项目

- ✅ `PutManifest`: 自动创建项目
- ✅ `InitiateBlobUpload`: 自动创建项目

### 统计更新

- ✅ `PutManifest`: 更新镜像数量和存储使用量
- ✅ `DeleteManifest`: 更新镜像数量和存储使用量
- ❌ `DeleteBlob`: 未更新统计（需要修复）

### Webhook事件

- ✅ `PutManifest`: 触发Push事件
- ✅ `DeleteManifest`: 触发Delete事件

## 已实施的修复

### 修复1: DeleteBlob后更新统计 ✅

已在 `DeleteBlob` 函数中添加统计更新逻辑：
- 删除blob后重新计算项目的存储使用量
- 更新项目统计信息

### 修复2: GetBlobUploadStatus添加权限检查 ✅

已在 `GetBlobUploadStatus` 函数开头添加权限检查：
- 检查 `push` 权限
- PAT token需要 `write` scope
- 未授权用户无法查看上传状态

## 总结

### 正常功能

- ✅ 所有拉取操作都有正确的权限检查
- ✅ 所有推送操作都有正确的权限检查和项目创建
- ✅ 删除manifest操作有正确的权限检查和统计更新
- ✅ 权限检查逻辑正确（PAT scope检查在项目存在性检查之前）

### 已修复

- ✅ `DeleteBlob` 后更新项目统计
- ✅ `GetBlobUploadStatus` 添加权限检查

### 整体评估

系统整体设计良好，权限检查严格，项目处理逻辑完善。所有问题已修复。
