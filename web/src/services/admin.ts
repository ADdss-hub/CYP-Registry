import apiClient from "./api";

export interface AuditLog {
  id: string;
  user_id: string;
  action: string;
  resource: string;
  resource_id: string;
  ip: string;
  user_agent: string;
  details: string;
  created_at: string;
}

export interface AuditLogListResponse {
  logs: AuditLog[];
  total: number;
  page: number;
  page_size: number;
  total_page: number;
}

export interface AuditLogQueryParams {
  page?: number;
  page_size?: number;
  user_id?: string;
  action?: string;
  resource?: string;
  start_time?: string;
  end_time?: string;
  keyword?: string;
}

export interface SystemConfig {
  https: {
    enabled: boolean;
    ssl_certificate_path: string;
    ssl_certificate_key_path: string;
    ssl_protocols: string[];
    http_redirect: boolean;
  };
  cors: {
    allowed_origins: string[];
    allowed_methods: string[];
    allowed_headers: string[];
  };
  rate_limit: {
    enabled: boolean;
    requests_per_second: number;
    burst: number;
  };
}

export interface UpdateSystemConfigRequest {
  cors?: {
    allowed_origins: string[];
    allowed_methods?: string[];
    allowed_headers?: string[];
  };
  rate_limit?: {
    enabled: boolean;
    requests_per_second: number;
    burst: number;
  };
}

export const adminApi = {
  /**
   * 获取审计日志列表
   */
  async getAuditLogs(
    params: AuditLogQueryParams = {},
  ): Promise<AuditLogListResponse> {
    const response = await apiClient.get<{
      code: number;
      data: AuditLogListResponse;
    }>("/v1/admin/logs", { params });
    if (response.data.code !== 20000) {
      throw new Error(response.data.data?.toString() || "获取日志失败");
    }
    return response.data.data;
  },

  /**
   * 获取系统配置
   */
  async getSystemConfig(): Promise<SystemConfig> {
    const response = await apiClient.get<{
      code: number;
      data: SystemConfig;
    }>("/v1/admin/config");
    if (response.data.code !== 20000) {
      throw new Error("获取系统配置失败");
    }
    return response.data.data;
  },

  /**
   * 更新系统配置
   */
  async updateSystemConfig(config: UpdateSystemConfigRequest): Promise<void> {
    const response = await apiClient.put<{ code: number; data: null }>(
      "/v1/admin/config",
      config,
    );
    if (response.data.code !== 20000) {
      throw new Error("更新系统配置失败");
    }
  },
};
