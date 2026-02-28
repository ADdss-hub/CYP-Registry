import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { Project } from "@/types";
import { projectApi } from "@/services/project";

export const useProjectStore = defineStore("project", () => {
  // 将后端 snake_case 字段映射为前端 Project（camelCase），避免列表/仪表盘统计显示为 0
  function mapProjectResponse(raw: any): Project {
    return {
      id: raw?.id,
      name: raw?.name ?? "",
      description: raw?.description ?? "",
      ownerId: raw?.ownerId ?? raw?.owner_id ?? "",
      isPublic: raw?.isPublic ?? raw?.is_public ?? false,
      storageUsed: raw?.storageUsed ?? raw?.storage_used ?? 0,
      storageQuota: raw?.storageQuota ?? raw?.storage_quota ?? 0,
      imageCount: raw?.imageCount ?? raw?.image_count ?? 0,
      createdAt: raw?.createdAt ?? raw?.created_at ?? "",
      updatedAt: raw?.updatedAt ?? raw?.updated_at ?? "",
    };
  }

  // 状态
  const projects = ref<Project[]>([]);
  const currentProject = ref<Project | null>(null);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  const pagination = ref({
    page: 1,
    pageSize: 10,
    total: 0,
  });

  // 计算属性
  const publicProjects = computed(() =>
    projects.value.filter((p) => p.isPublic),
  );
  const myProjects = computed(() =>
    projects.value.filter((p) => p.isPublic || true),
  ); // 根据实际权限过滤

  // 方法
  async function fetchProjects(params?: {
    page?: number;
    pageSize?: number;
    keyword?: string;
    isPublic?: boolean;
  }) {
    isLoading.value = true;
    error.value = null;

    try {
      // api.ts 的响应拦截器已经返回了 payload.data，所以 response 直接是 data 字段的内容
      // 后端返回结构：{ list: [...], total: ..., page: ..., page_size: ... }
      const data: any = await projectApi.getProjects({
        page: params?.page || pagination.value.page,
        pageSize: params?.pageSize || pagination.value.pageSize,
        keyword: params?.keyword,
        isPublic: params?.isPublic,
      });

      // 兼容后端实际返回结构：{ list, total, page, page_size } 或 { projects, total, page, page_size }
      const listRaw = (data?.list ?? data?.projects ?? []) as any[];
      const page = data?.page ?? 1;
      const pageSize =
        data?.pageSize ?? data?.page_size ?? pagination.value.pageSize;

      projects.value = Array.isArray(listRaw)
        ? listRaw.map(mapProjectResponse)
        : [];
      pagination.value.total = data?.total ?? 0;
      pagination.value.page = page;
      pagination.value.pageSize = pageSize;

      return data;
    } catch (err: any) {
      error.value = err.message || "获取项目列表失败";
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  async function fetchProject(id: string) {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await projectApi.getProject(id);
      // api.ts 的响应拦截器已经处理过，response 本身通常就是 data 字段
      // 兼容两种结构：{ project: {...} } 或直接 Project
      const data: any = response as any;
      const projectRaw = data.project ?? data;
      const project = mapProjectResponse(projectRaw);
      currentProject.value = project;
      return project;
    } catch (err: any) {
      error.value = err.message || "获取项目详情失败";
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  async function createProject(data: {
    name: string;
    description: string;
    isPublic: boolean;
    storageQuota?: number;
  }) {
    isLoading.value = true;
    error.value = null;

    try {
      // api.ts 的响应拦截器已经返回了 payload.data，所以 response 直接是 data 字段的内容
      // 后端返回结构：{ project: {...} }
      const response: any = await projectApi.createProject(data);
      const project = mapProjectResponse(response?.project ?? response);

      if (project) {
        // 检查是否已存在（避免重复添加）
        const exists = projects.value.find(
          (p) => p.id === project.id || p.name === project.name,
        );
        if (!exists) {
          projects.value.unshift(project);
        }
        // 更新分页总数
        pagination.value.total = (pagination.value.total || 0) + 1;
      }
      return project;
    } catch (err: any) {
      error.value = err.message || "创建项目失败";
      // 如果是项目已存在的错误，确保错误码和消息正确传递
      if (
        err.code === 20002 ||
        err.message?.includes("已存在") ||
        err.message?.includes("already exists")
      ) {
        const conflictError: any = new Error(
          err.message || "项目已存在，请使用其他名称",
        );
        conflictError.code = 20002;
        throw conflictError;
      }
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  async function updateProject(
    id: string,
    data: {
      name?: string;
      description?: string;
      isPublic?: boolean;
      storageQuota?: number;
    },
  ) {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await projectApi.updateProject(id, data);
      // api.ts 的响应拦截器已经返回 payload.data，这里统一兼容 { project } 或直接 Project
      const updated: any = mapProjectResponse(
        (response as any)?.project ?? (response as any),
      );

      // 更新列表中的项目
      const index = projects.value.findIndex((p) => p.id === id);
      if (index !== -1 && updated) {
        projects.value[index] = updated as Project;
      }

      // 更新当前项目
      if (currentProject.value?.id === id && updated) {
        currentProject.value = updated as Project;
      }

      return updated as Project;
    } catch (err: any) {
      error.value = err.message || "更新项目失败";
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  async function deleteProject(id: string) {
    isLoading.value = true;
    error.value = null;

    try {
      await projectApi.deleteProject(id);

      // 从列表中移除
      projects.value = projects.value.filter((p) => p.id !== id);

      // 清空当前项目
      if (currentProject.value?.id === id) {
        currentProject.value = null;
      }

      // 同步更新分页总数，避免仪表盘/列表统计不同步
      if (
        typeof pagination.value.total === "number" &&
        pagination.value.total > 0
      ) {
        pagination.value.total = pagination.value.total - 1;
      }
    } catch (err: any) {
      error.value = err.message || "删除项目失败";
      throw err;
    } finally {
      isLoading.value = false;
    }
  }

  function clearCurrentProject() {
    currentProject.value = null;
  }

  // 前端兜底：在项目详情页根据 Registry 实际镜像列表动态刷新单个项目的统计信息，
  // 避免后端统计尚未更新时仪表盘/项目列表长期显示为 0。
  function updateProjectStats(
    id: string,
    stats: { imageCount?: number; storageUsed?: number },
  ) {
    const idx = projects.value.findIndex((p) => p.id === id);
    if (idx !== -1) {
      projects.value[idx] = {
        ...projects.value[idx],
        imageCount: stats.imageCount ?? projects.value[idx].imageCount,
        storageUsed: stats.storageUsed ?? projects.value[idx].storageUsed,
      };
    }

    if (currentProject.value?.id === id) {
      currentProject.value = {
        ...currentProject.value,
        imageCount: stats.imageCount ?? currentProject.value.imageCount,
        storageUsed: stats.storageUsed ?? currentProject.value.storageUsed,
      };
    }
  }

  return {
    // 状态
    projects,
    currentProject,
    isLoading,
    error,
    pagination,
    // 计算属性
    publicProjects,
    myProjects,
    // 方法
    fetchProjects,
    fetchProject,
    createProject,
    updateProject,
    deleteProject,
    clearCurrentProject,
    updateProjectStats,
  };
});
