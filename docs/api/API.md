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
| POST | `/api/v1/auth/register` | 用户注册 | 否 |
| POST | `/api/v1/auth/refresh` | 刷新 Token | 是 |
| POST | `/api/v1/auth/logout` | 退出登录 | 是 |
| POST | `/api/v1/auth/mfa/verify` | MFA 验证 | 是 |

### 用户管理

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/v1/user/profile` | 获取用户信息 | 是 |
| PUT | `/api/v1/user/profile` | 更新用户信息 | 是 |
| PUT | `/api/v1/user/password` | 修改密码 | 是 |
| GET | `/api/v1/user/tokens` | 列出 PAT | 是 |
| POST | `/api/v1/user/tokens` | 创建 PAT | 是 |
| DELETE | `/api/v1/user/tokens/:id` | 删除 PAT | 是 |

### 项目管理

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/v1/projects` | 列出项目 | 是 |
| POST | `/api/v1/projects` | 创建项目 | 是 |
| GET | `/api/v1/projects/:id` | 项目详情 | 是 |
| PUT | `/api/v1/projects/:id` | 更新项目 | 是 |
| DELETE | `/api/v1/projects/:id` | 删除项目 | 是 |
| GET | `/api/v1/projects/:id/members` | 列出成员 | 是 |
| POST | `/api/v1/projects/:id/members` | 添加成员 | 是 |
| DELETE | `/api/v1/projects/:id/members/:userId` | 移除成员 | 是 |

### 镜像管理

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/v1/repositories` | 列出仓库 | 是 |
| GET | `/api/v1/repositories/:repo/tags` | 列出标签 | 是 |
| DELETE | `/api/v1/repositories/:repo/tags/:tag` | 删除标签 | 是 |
| GET | `/api/v1/repositories/:repo/manifests` | 列出清单 | 是 |
| GET | `/api/v1/repositories/:repo/vulnerabilities` | 漏洞信息 | 是 |

### 漏洞扫描

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/v1/scans` | 列出扫描任务 | 是 |
| GET | `/api/v1/scans/:id` | 扫描详情 | 是 |
| POST | `/api/v1/scans` | 触发扫描 | 是 |
| GET | `/api/v1/scans/:id/logs` | 扫描日志 | 是 |

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

---
*文档版本: v1.0.3*  
*更新日期: 2026-02-25*

