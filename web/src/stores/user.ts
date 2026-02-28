import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, LoginRequest, AccessToken, NotificationSettings } from '@/types'
import { userApi } from '@/services/user'
import router from '@/router'
import { LEGAL_STATEMENT_STORAGE_KEY } from '@/constants/legal'

export const useUserStore = defineStore('user', () => {
  // 状态
  const user = ref<User | null>(null)
  const token = ref<string | null>(localStorage.getItem('token'))
  const refreshToken = ref<string | null>(localStorage.getItem('refreshToken'))
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // 计算属性
  const isLoggedIn = computed(() => !!token.value && !!user.value)
  const userRole = computed(() => (user.value?.is_admin ? 'admin' : 'viewer'))
  const isAdmin = computed(() => !!user.value?.is_admin)

  // 方法
  async function login(credentials: LoginRequest) {
    isLoading.value = true
    error.value = null

    try {
      // api.ts 的响应拦截器已经处理了响应，直接返回 data 字段
      // 所以 response 就是 LoginResponse 对象
      const response = await userApi.login(credentials)
      
      // 检查响应结构
      if (!response || !response.access_token) {
        throw new Error('登录响应格式错误')
      }

      token.value = response.access_token
      refreshToken.value = response.refresh_token
      user.value = response.user

      // 保存到本地存储
      localStorage.setItem('token', response.access_token)
      localStorage.setItem('refreshToken', response.refresh_token)

      // 跳转逻辑：
      // - 若当前版本声明与数据处理规范尚未确认，则先进入独立声明界面
      // - 否则直接跳转到首页
      const hasAcknowledgedLegalStatement =
        localStorage.getItem(LEGAL_STATEMENT_STORAGE_KEY) === '1'

      if (hasAcknowledgedLegalStatement) {
        router.push('/')
      } else {
        router.push('/legal/statement')
      }
      
      return response
    } catch (err: any) {
      // 确保错误消息被正确设置
      const errorMessage = err?.message || err?.payload?.message || '登录失败，请检查用户名和密码'
      error.value = errorMessage
      // 重新抛出错误，让组件可以捕获并显示
      const loginError: any = new Error(errorMessage)
      loginError.code = err?.code
      loginError.payload = err?.payload
      throw loginError
    } finally {
      isLoading.value = false
    }
  }

  async function fetchCurrentUser() {
    if (!token.value) return

    isLoading.value = true
    try {
      // api.ts 的响应拦截器已经处理了响应，直接返回 data 字段
      // 所以 response 就是 User 对象
      const response = await userApi.getCurrentUser()
      user.value = response
    } catch (err) {
      // 如果获取失败，可能是令牌过期
      logout()
    } finally {
      isLoading.value = false
    }
  }

  function logout() {
    user.value = null
    token.value = null
    refreshToken.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    router.push('/login')
  }

  async function refreshAccessToken() {
    if (!refreshToken.value) {
      logout()
      return
    }

    try {
      // api.ts 的响应拦截器已经处理了响应，直接返回 data 字段
      // 所以 response 就是 LoginResponse 对象
      const response = await userApi.refreshToken(refreshToken.value)
      token.value = response.access_token
      refreshToken.value = response.refresh_token
      
      localStorage.setItem('token', response.access_token)
      localStorage.setItem('refreshToken', response.refresh_token)
      
      return response.access_token
    } catch {
      logout()
      throw new Error('令牌刷新失败')
    }
  }

  // 更新当前登录用户信息
  async function updateCurrentUser(data: { nickname?: string; avatar?: string; bio?: string }) {
    // api.ts 的响应拦截器已经处理了响应，直接返回 data 字段
    // 所以 res 就是 User 对象
    const res = await userApi.updateCurrentUser(data)
    user.value = res
    return res
  }

  // 上传头像（文件方式）
  async function uploadAvatar(file: File) {
    const formData = new FormData()
    formData.append('file', file)
    // api.ts 的响应拦截器已经处理了响应，直接返回 data 字段
    // 所以 res 就是 User 对象
    const res = await userApi.uploadAvatar(formData)
    user.value = res
    return res
  }

  // PAT：加载当前用户的访问令牌列表
  async function listPAT(): Promise<AccessToken[]> {
    return userApi.listPAT()
  }

  // PAT：创建访问令牌
  async function createPAT(payload: { name: string; scopes: string[]; expireInDays?: number }) {
    return userApi.createPAT(payload)
  }

  // PAT：撤销访问令牌
  async function revokePAT(id: string) {
    return userApi.revokePAT(id)
  }

  // 获取当前用户的通知设置
  async function getNotificationSettings(): Promise<NotificationSettings> {
    return userApi.getNotificationSettings()
  }

  // 更新当前用户的通知设置
  async function updateNotificationSettings(settings: NotificationSettings): Promise<NotificationSettings> {
    return userApi.updateNotificationSettings(settings)
  }

  // 初始化时获取用户信息
  if (token.value) {
    fetchCurrentUser()
  }

  return {
    // 状态
    user,
    token,
    refreshToken,
    isLoading,
    error,
    // 计算属性
    isLoggedIn,
    userRole,
    isAdmin,
    // 方法
    login,
    fetchCurrentUser,
    logout,
    refreshAccessToken,
    listPAT,
    createPAT,
    updateCurrentUser,
    uploadAvatar,
    revokePAT,
    getNotificationSettings,
    updateNotificationSettings,
  }
})

