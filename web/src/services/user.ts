import { api } from './api'
import type { 
  LoginRequest, 
  LoginResponse, 
  User,
  PaginatedResponse,
  AccessToken,
  NotificationSettings,
} from '@/types'

interface CreatePATRequestDto {
  name: string
  scopes: string[]
  expire_in?: number
}

interface PATResponseDto {
  id: string
  name: string
  scopes: string[]
  expires_at: string
  created_at: string
  last_used_at?: string | null
}

interface CreatePATResponseDto {
  id: string
  name: string
  scopes: string[]
  expires_at: string
  created_at: string
  token: string
  token_type: string
}

function mapPATToAccessToken(dto: PATResponseDto | CreatePATResponseDto, tokenValue?: string): AccessToken {
  // 后端使用 9999-12-31T23:59:59Z 表示“永不过期”，前端在此处转为语义字段
  const isNeverExpire = dto.expires_at && dto.expires_at.startsWith('9999-12-31')
  const expiresAt = isNeverExpire ? undefined : dto.expires_at || undefined

  return {
    id: dto.id,
    name: dto.name,
    token: tokenValue ?? '********',
    createdAt: dto.created_at,
    lastUsedAt: 'last_used_at' in dto ? dto.last_used_at ?? undefined : undefined,
    expiresAt,
    scopes: dto.scopes,
  }
}

// 用户相关API
export const userApi = {
  // 登录
  // 注意：api.post 的响应拦截器会返回 payload.data，所以实际返回的是 LoginResponse
  login: (data: LoginRequest) =>
    api.post<LoginResponse>('/v1/auth/login', data) as Promise<LoginResponse>,
  
  // 获取当前用户信息
  // 注意：api.get 的响应拦截器会返回 payload.data，所以实际返回的是 User
  getCurrentUser: () =>
    api.get<User>('/v1/users/me') as Promise<User>,
  
  // 更新当前用户信息
  // 注意：此处必须使用 PUT /v1/users/me，否则后端会将 PATCH /users/me 误路由到 /users/{id} 并返回 invalid id
  updateCurrentUser: (data: Partial<User>) =>
    api.put<User>('/v1/users/me', data) as Promise<User>,

  // 上传头像
  // 注意：api.upload 的响应拦截器会返回 payload.data，所以实际返回的是 User
  uploadAvatar: (formData: FormData, onProgress?: (progress: number) => void) =>
    api.upload<User>('/v1/users/me/avatar', formData, onProgress) as Promise<User>,
  
  // 刷新令牌
  // 注意：api.post 的响应拦截器会返回 payload.data，所以实际返回的是 LoginResponse
  refreshToken: (refreshToken: string) =>
    api.post<LoginResponse>('/v1/auth/refresh', { refreshToken }) as Promise<LoginResponse>,
  
  // 登出
  logout: () =>
    api.post<void>('/v1/auth/logout') as Promise<void>,
  
  // 首次部署：一次性获取默认管理员账号信息（用户名 + 密码），用于登录页提示并复制保存
  getDefaultAdminOnce: () =>
    api.get<{ username: string; password: string; created_at?: string }>('/v1/auth/default-admin-once'),
  
  // 获取用户列表
  // 注意：api.get 的响应拦截器会返回 payload.data，所以实际返回的是 PaginatedResponse<User>
  getUsers: (params?: { page?: number; pageSize?: number; keyword?: string }) =>
    api.get<PaginatedResponse<User>>('/v1/users', params) as Promise<PaginatedResponse<User>>,
  
  // 获取单个用户
  // 注意：api.get 的响应拦截器会返回 payload.data，所以实际返回的是 User
  getUser: (id: string) =>
    api.get<User>(`/v1/users/${id}`) as Promise<User>,
  
  // 更新用户
  // 注意：api.patch 的响应拦截器会返回 payload.data，所以实际返回的是 User
  updateUser: (id: string, data: Partial<User>) =>
    api.patch<User>(`/v1/users/${id}`, data) as Promise<User>,
  
  // 删除用户
  deleteUser: (id: string) =>
    api.delete<void>(`/v1/users/${id}`) as Promise<void>,

  // 获取当前用户的通知设置
  getNotificationSettings: () =>
    api.get<NotificationSettings>('/v1/users/me/notification-settings') as Promise<NotificationSettings>,

  // 更新当前用户的通知设置
  updateNotificationSettings: (data: NotificationSettings) =>
    api.put<NotificationSettings>('/v1/users/me/notification-settings', data) as Promise<NotificationSettings>,

  // PAT：获取当前用户的访问令牌列表
  // 注意：api.get 的响应拦截器会返回 payload.data，所以实际返回的是 PATResponseDto[]
  listPAT: async (): Promise<AccessToken[]> => {
    const data = await api.get<PATResponseDto[]>('/v1/users/me/pat')
    return data.map((item) => mapPATToAccessToken(item))
  },

  // PAT：创建新的访问令牌（返回一次性完整 token 信息）
  // 注意：api.post 的响应拦截器会返回 payload.data，所以实际返回的是 CreatePATResponseDto
  createPAT: async (payload: { name: string; scopes: string[]; expireInDays?: number }): Promise<{ token: string; tokenType: string; accessToken: AccessToken }> => {
    const body: CreatePATRequestDto = {
      name: payload.name,
      scopes: payload.scopes,
      // 前端约定：未选择或选择“永不过期”时使用 -1 表示永不过期
      expire_in:
        payload.expireInDays != null
          ? payload.expireInDays * 24 * 60 * 60
          : -1,
    }
    const data = await api.post<CreatePATResponseDto>('/v1/users/me/pat', body)
    return {
      token: data.token,
      tokenType: data.token_type,
      accessToken: mapPATToAccessToken(data, data.token),
    }
  },

  // PAT：撤销指定访问令牌
  revokePAT: (id: string) =>
    api.delete<void>(`/v1/users/me/pat/${id}`) as Promise<void>,
}

