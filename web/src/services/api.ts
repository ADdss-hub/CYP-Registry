import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig } from 'axios'
import { useUserStore } from '@/stores/user'
import router from '@/router'
import { useErrorCollector } from '@/composables/useErrorCollector'

// 创建全局错误收集器实例
const errorCollector = useErrorCollector()

// 创建Axios实例
const apiClient: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    try {
      // 添加认证令牌
      const userStore = useUserStore()
      if (userStore.token) {
        config.headers.Authorization = `Bearer ${userStore.token}`
      }

      // 添加请求ID
      config.headers['X-Request-ID'] = generateRequestId()

      return config
    } catch (error) {
      // 记录请求拦截器错误
      console.error('[API Request Interceptor Error]', {
        error,
        url: config.url,
        method: config.method,
        timestamp: new Date().toISOString(),
      })
      return Promise.reject(error)
    }
  },
  (error: AxiosError) => {
    // 记录请求拦截器错误
    console.error('[API Request Interceptor Error]', {
      error: error.message,
      stack: error.stack,
      timestamp: new Date().toISOString(),
    })
    return Promise.reject(error)
  }
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response: any) => {
    const payload: any = response.data
    const config = response.config

    // 兼容后端统一响应：{ code, message, data, ... }
    // 后端错误场景可能仍返回 HTTP 200，此处统一转为 Promise.reject，交由业务层处理
    if (payload && typeof payload === 'object' && typeof payload.code === 'number') {
      if (payload.code !== 20000) {
        const err: any = new Error(payload.message || '请求失败')
        err.code = payload.code
        err.data = payload.data
        err.payload = payload
        
        // 记录错误
        errorCollector.recordError({
          url: config?.url || 'unknown',
          method: config?.method?.toUpperCase() || 'unknown',
          status: response.status,
          message: payload.message || '请求失败',
          error: err,
        })
        
        return Promise.reject(err)
      }
      
      // 记录成功
      errorCollector.recordSuccess({
        url: config?.url || 'unknown',
        method: config?.method?.toUpperCase() || 'unknown',
        status: response.status,
        message: payload.message || 'Success',
      })
      
      // 成功时返回data字段，而不是整个payload
      return payload.data !== undefined ? payload.data : payload
    }

    // 记录成功（非标准响应格式）
    errorCollector.recordSuccess({
      url: config?.url || 'unknown',
      method: config?.method?.toUpperCase() || 'unknown',
      status: response.status,
    })

    return payload
  },
  (error: AxiosError) => {
    const { response, request, config } = error

    // 记录所有API错误（包括网络错误和HTTP错误）
    console.error('[API Response Error]', {
      url: config?.url || 'unknown',
      method: config?.method || 'unknown',
      status: response?.status || 'network_error',
      statusText: response?.statusText || 'Network Error',
      data: response?.data,
      message: error.message,
      timestamp: new Date().toISOString(),
    })

    // 记录到错误收集器
    errorCollector.recordError({
      url: config?.url || 'unknown',
      method: config?.method?.toUpperCase() || 'unknown',
      status: response?.status,
      message: error.message || '请求失败',
      error: error,
      stack: error.stack,
    })

    if (response) {
      // 尝试从响应数据中提取错误消息（后端统一响应格式）
      const responseData = response.data as any
      if (responseData && typeof responseData === 'object') {
        // 优先使用后端返回的message
        const message = responseData.message || `请求失败 (${response.status})`
        const err: any = new Error(message)
        err.code = responseData.code || response.status
        err.data = responseData.data
        err.payload = responseData
        return Promise.reject(err)
      }

      // 如果没有响应数据，根据状态码生成错误消息
      let message = '请求失败'
      switch (response.status) {
        case 401:
          message = '未授权，请重新登录'
          const userStore = useUserStore()
          userStore.logout()
          router.push('/login')
          break
        case 403:
          message = '禁止访问'
          break
        case 404:
          message = '资源不存在'
          break
        case 422:
          message = '参数验证错误'
          break
        case 429:
          message = '请求过于频繁，请稍后再试'
          break
        case 500:
          message = '服务器内部错误'
          break
        default:
          message = `请求失败 (${response.status})`
      }
      const err: any = new Error(message)
      err.code = response.status
      return Promise.reject(err)
    } else if (request) {
      // 网络错误（请求已发送但未收到响应）
      const err: any = new Error('网络错误，请检查网络连接')
      err.code = 'NETWORK_ERROR'
      return Promise.reject(err)
    } else {
      // 请求配置错误或其他错误
      const err: any = new Error(error.message || '请求配置错误')
      err.code = 'REQUEST_ERROR'
      return Promise.reject(err)
    }
  }
)

// 生成请求ID
function generateRequestId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`
}

// API方法封装
export const api = {
  get: <T>(url: string, params?: object) =>
    apiClient.get<T, T>(url, { params }),
  
  post: <T>(url: string, data?: object) =>
    apiClient.post<T, T>(url, data),
  
  put: <T>(url: string, data?: object) =>
    apiClient.put<T, T>(url, data),
  
  patch: <T>(url: string, data?: object) =>
    apiClient.patch<T, T>(url, data),
  
  delete: <T>(url: string) =>
    apiClient.delete<T, T>(url),
  
  upload: <T>(url: string, formData: FormData, onProgress?: (progress: number) => void) =>
    apiClient.post<T, T>(url, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (progressEvent: any) => {
        if (onProgress && progressEvent.total) {
          const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total)
          onProgress(progress)
        }
      },
    }),
}

export default apiClient

