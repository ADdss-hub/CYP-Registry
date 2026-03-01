import apiClient from "./api";

export interface ImportImageRequest {
  source_url: string;
  target_image?: string;
  target_tag?: string;
  auth?: {
    username: string;
    password: string;
  };
}

export interface ImportTask {
  task_id: string;
  status: "pending" | "running" | "success" | "failed";
  progress: number;
  message: string;
  source_url: string;
  target_image: string;
  target_tag: string;
  error?: string;
  created_at: string;
  completed_at?: string;
}

export interface ImportTaskListResponse {
  tasks: ImportTask[];
  total: number;
  page: number;
  page_size: number;
  total_page: number;
}

export const imageImportApi = {
  /**
   * 导入镜像
   */
  async importImage(
    projectId: string,
    data: ImportImageRequest
  ): Promise<ImportTask> {
    const response = await apiClient.post<{
      code: number;
      data: ImportTask;
    }>(`/v1/projects/${projectId}/images/import`, data);
    if (response.data.code !== 20000) {
      throw new Error(response.data.data?.toString() || "导入镜像失败");
    }
    return response.data.data;
  },

  /**
   * 获取任务信息
   */
  async getTask(projectId: string, taskId: string): Promise<ImportTask> {
    const response = await apiClient.get<{
      code: number;
      data: ImportTask;
    }>(`/v1/projects/${projectId}/images/import/${taskId}`);
    if (response.data.code !== 20000) {
      throw new Error(response.data.data?.toString() || "获取任务信息失败");
    }
    return response.data.data;
  },

  /**
   * 列出项目的导入任务
   */
  async listTasks(
    projectId: string,
    page: number = 1,
    pageSize: number = 20
  ): Promise<ImportTaskListResponse> {
    const response = await apiClient.get<{
      code: number;
      data: ImportTaskListResponse;
    }>(`/v1/projects/${projectId}/images/import`, {
      params: { page, page_size: pageSize },
    });
    if (response.data.code !== 20000) {
      throw new Error(response.data.data?.toString() || "获取任务列表失败");
    }
    return response.data.data;
  },
};
