import { api } from './api'
import type { Webhook, WebhookDelivery, WebhookEvent } from '@/types'

export const webhookApi = {
  // 获取Webhook列表（api.ts 已解包统一响应，这里直接返回 Webhook[]）
  getWebhooks: (projectId: string) =>
    api.get<Webhook[]>('/v1/webhooks', { projectId }),

  // 获取单个Webhook
  getWebhook: (webhookId: string) =>
    api.get<Webhook>(`/v1/webhooks/${webhookId}`),

  // 创建Webhook
  createWebhook: (data: {
    projectId: string
    name: string
    description: string
    url: string
    secret?: string
    events: string[]
    headers?: Record<string, string>
    retryPolicy?: {
      maxRetries: number
      retryDelay: number
      backoffMultiplier: number
      maxDelay: number
    }
  }) =>
    api.post<Webhook>('/v1/webhooks', data),

  // 更新Webhook
  updateWebhook: (webhookId: string, data: Partial<Webhook>) =>
    api.patch<Webhook>(`/v1/webhooks/${webhookId}`, data),

  // 删除Webhook
  deleteWebhook: (webhookId: string) =>
    api.delete<void>(`/v1/webhooks/${webhookId}`),

  // 测试Webhook
  testWebhook: (webhookId: string, data?: { eventType?: string; payload?: Record<string, unknown> }) =>
    api.post<WebhookDelivery>(`/v1/webhooks/${webhookId}/test`, data),

  // 获取投递历史
  getDeliveries: (webhookId: string, params?: { eventId?: string }) =>
    api.get<WebhookDelivery[]>(`/v1/webhooks/${webhookId}/deliveries`, params),

  // 获取事件详情
  getEvent: (eventId: string) =>
    api.get<WebhookEvent>(`/v1/webhooks/events/${eventId}`),

  // 获取Webhook统计
  getStatistics: () =>
    api.get<WebhookStatistics>('/v1/webhooks/statistics'),
}

export interface WebhookStatistics {
  totalWebhooks: number
  activeWebhooks: number
  totalEvents: number
  deliveredEvents: number
  failedEvents: number
  successRate: number
}

