# PAT 令牌使用示例

## 概述

本文档提供 PAT (Personal Access Token) 令牌的使用示例，说明如何创建和使用 PAT 令牌。

## 创建 PAT 令牌

### 1. 创建只读 PAT

**前端选择**：只勾选"读取"

**请求示例**：
```bash
POST /api/v1/users/me/pat
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "name": "只读令牌",
  "scopes": ["read"],
  "expire_in": 86400
}
```

**可用功能**：
- ✅ 拉取镜像、查看项目信息
- ❌ 推送镜像、删除镜像、管理员功能

### 2. 创建读写 PAT

**前端选择**：勾选"读取"和"写入"

**请求示例**：
```bash
POST /api/v1/users/me/pat
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "name": "读写令牌",
  "scopes": ["read", "write"],
  "expire_in": 86400
}
```

**可用功能**：
- ✅ 拉取镜像、推送镜像、查看/创建项目
- ❌ 删除镜像、管理员功能

### 3. 创建完整权限 PAT

**前端选择**：勾选"读取"、"写入"、"删除"

**请求示例**：
```bash
POST /api/v1/users/me/pat
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "name": "完整权限令牌",
  "scopes": ["read", "write", "delete"],
  "expire_in": 86400
}
```

**可用功能**：
- ✅ 所有项目操作（拉取、推送、删除）
- ❌ 管理员功能

### 4. 创建管理员 PAT

**前端选择**：勾选"管理"

**请求示例**：
```bash
POST /api/v1/users/me/pat
Authorization: Bearer <your_jwt_token>
Content-Type: application/json

{
  "name": "管理员令牌",
  "scopes": ["admin"],
  "expire_in": 0
}
```

**可用功能**：
- ✅ 所有功能（包括管理员功能、查看日志等）

**说明**：
- `expire_in` 为 0 表示使用默认过期时间，-1 表示永不过期


## 使用 PAT 令牌

### 1. 使用 Bearer Token 方式

**请求示例**：
```bash
GET /api/v1/admin/logs
Authorization: Bearer pat_v1_abc123...
```

### 2. 使用 Basic Auth 方式（Docker 客户端）

**请求示例**：
```bash
docker login registry.example.com
Username: <any_username>
Password: pat_v1_abc123...
```

### 3. 直接使用 PAT（不推荐）

**请求示例**：
```bash
GET /api/v1/admin/logs
Authorization: pat_v1_abc123...
```

## 权限验证规则

### 权限检查原则

**选择什么权限就是什么权限**：前端界面选择的权限选项直接对应后端的权限检查。

### 管理员功能访问

当使用 PAT 令牌访问需要管理员权限的接口时：

- ✅ 必须包含 `admin`、`admin:*` 或 `*` scope
- ❌ 否则返回错误码 30017

### 示例场景

#### 场景 1：只读 PAT 访问管理员功能

```bash
# 创建 PAT（只有读取权限）
POST /api/v1/users/me/pat
{
  "name": "只读令牌",
  "scopes": ["read"]
}

# 使用 PAT 访问管理员功能
GET /api/v1/admin/logs
Authorization: Bearer pat_v1_xyz789...
# ❌ 拒绝访问，返回错误码 30017
```

#### 场景 2：管理员 PAT 访问日志

```bash
# 创建 PAT（包含管理员权限）
POST /api/v1/users/me/pat
{
  "name": "管理员令牌",
  "scopes": ["admin"]
}

# 使用 PAT 访问管理员功能
GET /api/v1/admin/logs
Authorization: Bearer pat_v1_def456...
# ✅ 允许访问（包含管理员权限）
```

## 查看 PAT 列表

**请求示例**：
```bash
GET /api/v1/users/me/pat
Authorization: Bearer <your_jwt_token>
```

**响应示例**：
```json
{
  "code": 20000,
  "message": "success",
  "data": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "name": "管理员令牌",
      "scopes": [],
      "expires_at": "9999-12-31T23:59:59Z",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

## 撤销 PAT 令牌

**请求示例**：
```bash
DELETE /api/v1/users/me/pat/123e4567-e89b-12d3-a456-426614174000
Authorization: Bearer <your_jwt_token>
```

**响应示例**：
```json
{
  "code": 20000,
  "message": "PAT已删除",
  "data": null
}
```

## 最佳实践

1. **使用默认权限**：对于管理员，推荐创建 scopes 为空的 PAT，默认拥有全部权限
2. **明确权限范围**：如果需要限制权限，明确指定 scopes
3. **定期审查**：定期查看 PAT 列表，撤销不再使用的令牌
4. **安全存储**：PAT 令牌应安全存储，不要提交到代码仓库
5. **设置过期时间**：为 PAT 设置合理的过期时间，避免长期有效的令牌

## 常见问题

### Q: 如何限制 PAT 的权限？

A: 在创建 PAT 时，明确指定 scopes。例如，只授予读取权限：`["read"]`

### Q: 权限是如何继承的？

A: 系统支持权限继承：
- `delete` scope 包含 `write` 和 `read` 权限
- `write` scope 包含 `read` 权限
- `admin` scope 包含所有权限

因此，如果PAT有`write`权限，可以执行需要`read`权限的操作。

### Q: PAT 令牌可以用于哪些场景？

A: PAT 令牌可以用于：
- API 调用（替代 JWT token）
- Docker 客户端登录（使用 Basic Auth）
- CI/CD 流水线自动化
- 第三方系统集成

### Q: PAT 令牌和 JWT token 有什么区别？

A: 
- **JWT token**：短期有效，需要定期刷新，继承用户所有权限
- **PAT 令牌**：可以设置长期有效，适合自动化场景，通过scopes精确控制权限
- **权限控制**：PAT 按照选择的scopes进行权限检查，JWT token 继承用户的所有权限

### Q: 权限错误码有什么作用？

A: 系统为不同的权限错误定义了专门的状态码：
- `30014` - 缺少读取权限
- `30015` - 缺少写入权限
- `30016` - 缺少删除权限
- `30017` - 缺少管理员权限

前端可以根据错误码提供针对性的提示，帮助用户快速解决问题。
