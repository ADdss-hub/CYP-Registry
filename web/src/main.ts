import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useErrorCollector } from '@/composables/useErrorCollector'

// 导入全局样式
import '@/assets/styles/global.scss'

// 创建Vue应用
const app = createApp(App)

// 创建全局错误收集器实例
const errorCollector = useErrorCollector()

// 全局错误处理器配置
// 1. Vue组件错误处理
app.config.errorHandler = (err: unknown, instance, info) => {
  const errorMessage = err instanceof Error ? err.message : String(err)
  const errorStack = err instanceof Error ? err.stack : ''
  const componentName = instance?.$?.type?.name || 'Unknown'
  
  console.error('[Vue Error Handler]', {
    error: errorMessage,
    stack: errorStack,
    component: componentName,
    info,
    timestamp: new Date().toISOString(),
    url: window.location.href,
  })
  
  // 记录到错误收集器
  errorCollector.recordError({
    url: window.location.href,
    method: 'VUE_COMPONENT',
    message: errorMessage,
    error: err,
    stack: errorStack,
    component: componentName,
    info: typeof info === 'string' ? info : JSON.stringify(info),
  })
  
  // 输出详细错误信息到控制台
  console.error('错误详情:', err)
  console.error('组件信息:', info)
  if (errorStack) {
    console.error('错误堆栈:', errorStack)
  }
}

// 2. 未处理的Promise拒绝处理
window.addEventListener('unhandledrejection', (event: PromiseRejectionEvent) => {
  const error = event.reason
  const errorMessage = error instanceof Error ? error.message : String(error)
  const errorStack = error instanceof Error ? error.stack : ''
  
  console.error('[Unhandled Promise Rejection]', {
    error: errorMessage,
    stack: errorStack,
    timestamp: new Date().toISOString(),
    url: window.location.href,
  })
  
  // 记录到错误收集器
  errorCollector.recordError({
    url: window.location.href,
    method: 'PROMISE_REJECTION',
    message: errorMessage,
    error: error,
    stack: errorStack,
  })
  
  // 输出详细错误信息
  console.error('Promise拒绝原因:', error)
  if (errorStack) {
    console.error('错误堆栈:', errorStack)
  }
  
  // 阻止默认行为（在控制台显示错误）
  event.preventDefault()
})

// 3. 全局JavaScript错误处理
window.addEventListener('error', (event: ErrorEvent) => {
  console.error('[Global Error Handler]', {
    message: event.message,
    filename: event.filename,
    lineno: event.lineno,
    colno: event.colno,
    error: event.error,
    stack: event.error?.stack,
    timestamp: new Date().toISOString(),
    url: window.location.href,
  })
  
  // 记录到错误收集器
  errorCollector.recordError({
    url: event.filename || window.location.href,
    method: 'JAVASCRIPT_ERROR',
    message: event.message,
    error: event.error,
    stack: event.error?.stack,
    info: `文件: ${event.filename}, 行: ${event.lineno}, 列: ${event.colno}`,
  })
  
  // 输出详细错误信息
  console.error('错误消息:', event.message)
  console.error('错误文件:', event.filename, `行: ${event.lineno}, 列: ${event.colno}`)
  if (event.error) {
    console.error('错误对象:', event.error)
    if (event.error.stack) {
      console.error('错误堆栈:', event.error.stack)
    }
  }
  
  // 返回false以允许默认错误处理继续
  return false
})

// 4. 资源加载错误处理
window.addEventListener('error', (event: ErrorEvent) => {
  // 检查是否是资源加载错误
  if (event.target && event.target !== window) {
    const target = event.target as HTMLElement
    const src = (target as HTMLImageElement).src || (target as HTMLLinkElement).href || 'N/A'
    
    console.error('[Resource Load Error]', {
      tag: target.tagName,
      src,
      timestamp: new Date().toISOString(),
      url: window.location.href,
    })
    
    // 记录到错误收集器
    errorCollector.recordError({
      url: src,
      method: 'RESOURCE_LOAD',
      message: `资源加载失败: ${target.tagName}`,
      info: `标签: ${target.tagName}, 源: ${src}`,
    })
  }
}, true) // 使用捕获阶段

// 使用Pinia状态管理
const pinia = createPinia()
app.use(pinia)

// 使用Vue Router
app.use(router)

// 挂载应用
app.mount('#app')

