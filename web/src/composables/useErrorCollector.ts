import { ref, computed } from 'vue'

interface ErrorRecord {
  id: string
  timestamp: string
  type: 'error' | 'success'
  url: string
  method: string
  status?: number
  message: string
  error?: any
  stack?: string
  component?: string
  info?: string
}

// 全局错误和成功记录存储
const errorRecords = ref<ErrorRecord[]>([])
const maxRecords = 1000 // 最多保存1000条记录

// 统计信息
const stats = computed(() => {
  const total = errorRecords.value.length
  const errors = errorRecords.value.filter(r => r.type === 'error').length
  const successes = errorRecords.value.filter(r => r.type === 'success').length
  
  return {
    total,
    errors,
    successes,
    errorRate: total > 0 ? (errors / total) * 100 : 0,
  }
})

/**
 * 记录错误
 */
function recordError(error: {
  url?: string
  method?: string
  status?: number
  message: string
  error?: any
  stack?: string
  component?: string
  info?: string
}) {
  const record: ErrorRecord = {
    id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
    timestamp: new Date().toISOString(),
    type: 'error',
    url: error.url || 'unknown',
    method: error.method || 'unknown',
    status: error.status,
    message: error.message || 'Unknown error',
    error: error.error,
    stack: error.stack,
    component: error.component,
    info: error.info,
  }
  
  errorRecords.value.unshift(record)
  
  // 限制记录数量
  if (errorRecords.value.length > maxRecords) {
    errorRecords.value = errorRecords.value.slice(0, maxRecords)
  }
  
  // 输出到控制台（开发环境）
  if (import.meta.env.DEV) {
    console.error('[Error Collector]', record)
  }
  
  return record.id
}

/**
 * 记录成功
 */
function recordSuccess(success: {
  url?: string
  method?: string
  status?: number
  message?: string
}) {
  const record: ErrorRecord = {
    id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
    timestamp: new Date().toISOString(),
    type: 'success',
    url: success.url || 'unknown',
    method: success.method || 'unknown',
    status: success.status || 200,
    message: success.message || 'Success',
  }
  
  errorRecords.value.unshift(record)
  
  // 限制记录数量
  if (errorRecords.value.length > maxRecords) {
    errorRecords.value = errorRecords.value.slice(0, maxRecords)
  }
  
  // 输出到控制台（开发环境）
  if (import.meta.env.DEV) {
    console.log('[Success Collector]', record)
  }
  
  return record.id
}

/**
 * 清空所有记录
 */
function clearRecords() {
  errorRecords.value = []
}

/**
 * 获取最近的错误记录
 */
function getRecentErrors(limit: number = 10): ErrorRecord[] {
  return errorRecords.value
    .filter(r => r.type === 'error')
    .slice(0, limit)
}

/**
 * 获取最近的成功记录
 */
function getRecentSuccesses(limit: number = 10): ErrorRecord[] {
  return errorRecords.value
    .filter(r => r.type === 'success')
    .slice(0, limit)
}

/**
 * 获取所有记录
 */
function getAllRecords(): ErrorRecord[] {
  return [...errorRecords.value]
}

/**
 * 导出记录为JSON
 */
function exportRecords(): string {
  return JSON.stringify({
    stats: stats.value,
    records: errorRecords.value,
    exportedAt: new Date().toISOString(),
  }, null, 2)
}

export function useErrorCollector() {
  return {
    // 状态
    records: computed(() => errorRecords.value),
    stats,
    
    // 方法
    recordError,
    recordSuccess,
    clearRecords,
    getRecentErrors,
    getRecentSuccesses,
    getAllRecords,
    exportRecords,
  }
}
