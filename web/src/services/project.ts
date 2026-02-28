import { api } from "./api";
import type {
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  PaginatedResponse,
} from "@/types";

// 将前端的 CreateProjectRequest 映射为后端期望的字段命名（snake_case）
function mapCreateProjectPayload(data: CreateProjectRequest) {
  return {
    name: data.name,
    description: data.description,
    is_public: data.isPublic,
    // 后端 CreateProjectRequest 当前未显式接收 storage_quota，此字段为预留
    storage_quota: data.storageQuota,
  };
}

// 将前端的 UpdateProjectRequest 映射为后端期望的字段命名
function mapUpdateProjectPayload(data: UpdateProjectRequest) {
  return {
    description: data.description,
    is_public: data.isPublic,
    storage_quota: data.storageQuota,
  };
}

export const projectApi = {
  // 注意：api 层已将统一响应格式解包为 data 字段
  getProjects: (params?: {
    page?: number;
    pageSize?: number;
    keyword?: string;
    isPublic?: boolean;
  }) =>
    api.get<PaginatedResponse<Project>>("/v1/projects", params) as Promise<
      PaginatedResponse<Project>
    >,

  getProject: (id: string) =>
    api.get<Project>(`/v1/projects/${id}`) as Promise<Project>,

  createProject: (data: CreateProjectRequest) =>
    api.post<{ project: Project }>(
      "/v1/projects",
      mapCreateProjectPayload(data),
    ) as Promise<{ project: Project }>,

  updateProject: (id: string, data: UpdateProjectRequest) =>
    api.put<Project>(
      `/v1/projects/${id}`,
      mapUpdateProjectPayload(data),
    ) as Promise<Project>,

  deleteProject: (id: string) =>
    api.delete<void>(`/v1/projects/${id}`) as Promise<void>,
};
