# CYP-Registry API 文档

本文档描述 CYP-Registry 的 RESTful API 接口规范。

## 基础信息

| 项目 | 值 |
|-----|-----|
| Base URL | `/api/v1` |
| 认证方式 | Bearer Token / PAT |
| 响应格式 | JSON |
| 字符编码 | UTF-8 |

## 统一响应格式

### 成功响应
```json
{
  "code": 20000,
  "message": "success",
  "data": {
    // 业务数据
  },
  "timestamp": 1704067200,
  "trace_id": "abc12345"
}
```

### 错误响应
```json
{
  "code": 10001,
  "message": "参数错误",
  "data": null,
  "timestamp": 1704067200,
  "trace_id": "abc12345"
}
```

## 错误码说明

| 错误码范围 | 说明 |
|-----------|------|
| 20000-20099 | 成功 |
| 10001-19999 | 参数错误 |
| 20001-29999 | 资源错误 |
| 30001-39999 | 认证/授权错误 |
| 50001-59999 | 服务器错误 |

## 认证

### 获取访问令牌
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "your_username",
  "password": "your_password"
}
```

### 响应
```json
{
  "code": 20000,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": 1704070800,
    "token_type": "Bearer"
  }
}
```

## API 接口列表

### 用户认证

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| POST | `/api/v1/auth/login` | 用户登录 | 否 |
| POST | `/api/v1/auth/refresh` | 刷新 Token | 是 |
| POST | `/api/v1/auth/logout` | 退出登录 | 是 |
| GET | `/api/v1/auth/default-admin-once` | 获取默认管理员提示（首次启动） | 否 |

### 用户管理

| 方法 | 路径 | 描述 | 认证 | 权限 |
|------|------|------|------|------|
| GET | `/api/v1/users/me` | 获取当前用户信息 | 是 | - |
| PUT | `/api/v1/users/me` | 更新当前用户信息 | 是 | - |
| PUT | `/api/v1/users/me/password` | 修改密码 | 是 | - |
| POST | `/api/v1/users/me/avatar` | 上传头像 | 是 | - |
| GET | `/api/v1/users/me/token-info` | 获取当前 Token 信息 | 是 | - |
| GET | `/api/v1/users/me/notification-settings` | 获取通知设置 | 是 | - |
| PUT | `/api/v1/users/me/notification-settings` | 更新通知设置 | 是 | - |
| POST | `/api/v1/users/me/pat` | 创建 Personal Access Token | 是 | - |
| GET | `/api/v1/users/me/pat` | 列出所有 PAT | 是 | - |
| DELETE | `/api/v1/users/me/pat/:id` | 撤销 PAT | 是 | - |
| GET | `/api/v1/users` | 列出所有用户 | 是 | 管理员 |
| GET | `/api/v1/users/:id` | 获取用户详情 | 是 | 管理员 |
| PATCH | `/api/v1/users/:id` | 更新用户信息 | 是 | 管理员 |
| DELETE | `/api/v1/users/:id` | 删除用户 | 是 | 管理员 |

### 项目管理

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/v1/projects` | 列出项目 | 是 |
| POST | `/api/v1/projects` | 创建项目 | 是 |
| GET | `/api/v1/projects/statistics` | 获取项目统计信息 | 是 |
| GET | `/api/v1/projects/:id` | 项目详情 | 是 |
| PUT | `/api/v1/projects/:id` | 更新项目 | 是 |
| PATCH | `/api/v1/projects/:id` | 更新项目（兼容） | 是 |
| DELETE | `/api/v1/projects/:id` | 删除项目 | 是 |
| PUT | `/api/v1/projects/:id/quota` | 更新存储配额 | 是 |
| GET | `/api/v1/projects/:id/storage` | 获取存储使用情况 | 是 |
| POST | `/api/v1/projects/:id/images/import` | 从 URL 导入镜像 | 是 |
| GET | `/api/v1/projects/:id/images/import` | 获取导入任务列表 | 是 |
| GET | `/api/v1/projects/:id/images/import/:task_id` | 获取导入任务详情 | 是 |

**注意**：项目成员/团队功能已下线，相关接口（`/api/v1/projects/:id/members`）返回 410 状态码。

### Webhook

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/v1/webhooks` | 列出 Webhook | 是 |
| POST | `/api/v1/webhooks` | 创建 Webhook | 是 |
| GET | `/api/v1/webhooks/:id` | Webhook 详情 | 是 |
| PUT | `/api/v1/webhooks/:id` | 更新 Webhook | 是 |
| DELETE | `/api/v1/webhooks/:id` | 删除 Webhook | 是 |
| POST | `/api/v1/webhooks/:id/test` | 测试 Webhook | 是 |
| GET | `/api/v1/webhooks/:id/deliveries` | 发送记录 | 是 |

### Docker Registry API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/v2/` | API 版本检查 |
| GET | `/v2/:name/tags/list` | 列出标签 |
| GET | `/v2/:name/manifests/:ref` | 获取清单 |
| PUT | `/v2/:name/manifests/:ref` | 推送清单 |
| GET | `/v2/:name/blobs/:digest` | 拉取层 |
| POST | `/v2/:name/blobs/uploads/` | 开始上传 |
| PATCH | `/v2/:name/blobs/uploads/:uuid` | 上传块 |
| PUT | `/v2/:name/blobs/uploads/:uuid` | 完成上传 |
| DELETE | `/v2/:name/manifests/:ref` | 删除镜像 |

## 接口详细说明

### 创建项目
```http
POST /api/v1/projects
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "my-project",
  "description": "项目描述",
  "is_public": false,
  "storage_quota": 10737418240
}
```

### 响应
```json
{
  "code": 20000,
  "message": "success",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "my-project",
    "description": "项目描述",
    "is_public": false,
    "storage_quota": 10737418240,
    "storage_used": 0,
    "image_count": 0,
    "created_at": "2026-02-01T00:00:00Z",
    "updated_at": "2026-02-01T00:00:00Z"
  }
}
```

## 镜像导入接口详细说明

### 创建镜像导入任务
```http
POST /api/v1/projects/:id/images/import
Authorization: Bearer <token>
Content-Type: application/json

{
  "image": "nginx:latest",
  "username": "optional_username",
  "password": "optional_password"
}
```

### 响应
```json
{
  "code": 20000,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "pending",
    "image": "nginx:latest",
    "created_at": "2026-02-01T00:00:00Z"
  }
}
```

### 查询导入任务列表
```http
GET /api/v1/projects/:id/images/import
Authorization: Bearer <token>
```

### 查询导入任务详情
```http
GET /api/v1/projects/:id/images/import/:task_id
Authorization: Bearer <token>
```

### 响应
```json
{
  "code": 20000,
  "message": "success",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "image": "nginx:latest",
    "progress": 100,
    "message": "镜像导入成功",
    "created_at": "2026-02-01T00:00:00Z",
    "updated_at": "2026-02-01T00:05:00Z"
  }
}
```

**任务状态说明：**
- `pending`: 等待处理
- `running`: 正在导入
- `completed`: 导入成功
- `failed`: 导入失败

## 健康检查

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/health` | 健康检查 | 否 |
| GET | `/api/health` | 健康检查（兼容） | 否 |

### 响应
```json
{
  "status": "healthy",
  "service": "CYP-Registry",
  "version": "v1.1.0"
}
```

---
*文档版本: v1.1.0*  
*适用后端版本: v1.1.0*

