import { api } from './api'
import type { PaginatedResponse } from '@/types'

export interface ActivityItem {
  id: string
  type: 'push' | 'scan' | 'delete' | string
  user: string
  projectName: string
  reference: string
  time: string
}

export const activityApi = {
  getRecentActivities: (params?: { page?: number; pageSize?: number }) =>
    api.get<PaginatedResponse<ActivityItem>>(
      '/v1/activities/recent',
      params,
    ) as Promise<PaginatedResponse<ActivityItem>>,
}

