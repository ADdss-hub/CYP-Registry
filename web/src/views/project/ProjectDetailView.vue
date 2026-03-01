<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useProjectStore } from "@/stores/project";
import { useUserStore } from "@/stores/user";
import { useNotificationStore } from "@/stores/notification";
import type {} from "@/types";
import CypButton from "@/components/common/CypButton.vue";
import CypInput from "@/components/common/CypInput.vue";
import CypSelect from "@/components/common/CypSelect.vue";
import CypDialog from "@/components/common/CypDialog.vue";
import CypLoading from "@/components/common/CypLoading.vue";
import CypCheckbox from "@/components/common/CypCheckbox.vue";
import { copyToClipboard } from "@/utils/clipboard";
import { imageImportApi, type ImportTask } from "@/services/imageImport";
const route = useRoute();
const router = useRouter();
const projectStore = useProjectStore();
const userStore = useUserStore();
const notificationStore = useNotificationStore();

const project = ref<any>(null);
const activeTab = ref("images");

// 危险操作 & 结果反馈对话框（替代浏览器 confirm/alert，遵循界面规范3.3/3.4节）
const showDeleteImageDialog = ref(false);
const imageToDelete = ref<any>(null);
const showDeleteProjectDialog = ref(false);
const showMessageDialog = ref(false);
const messageDialogTitle = ref("");
const messageDialogContent = ref("");

function openMessageDialog(title: string, content: string) {
  messageDialogTitle.value = title;
  messageDialogContent.value = content;
  showMessageDialog.value = true;
}

interface ProjectImage {
  name: string;
  digest: string;
  size: number;
  // 若后端暂未提供推送时间/用户信息，则使用空值并在界面上展示“未知”
  pushTime: string | null;
  pushedBy: string | null;
}

interface ImportFormState {
  source_url: string;
  target_image: string;
  target_tag: string;
  auth: {
    username: string;
    password: string;
  };
}

// 镜像列表（从后端 Registry 实时加载）
const images = ref<ProjectImage[]>([]);
const isImagesLoading = ref(false);

// 镜像列表自动刷新开关与定时器
const autoRefreshEnabled = ref(false);
let autoRefreshTimer: number | null = null;
const AUTO_REFRESH_INTERVAL = 30000; // 30 秒

// Registry SSE 事件源，用于实时监听 push/delete 完成事件
let registryEventSource: EventSource | null = null;

const isOwner = computed(() => {
  const currentUser = userStore.user;
  return userStore.isAdmin || project.value?.ownerId === currentUser?.id;
});

// 从 Registry V2 API 加载当前项目下的镜像标签与信息
async function loadImages(showLoading = true) {
  if (!project.value || !project.value.name) {
    images.value = [];
    return;
  }

  if (showLoading) {
    isImagesLoading.value = true;
  }
  try {
    // 项目名即仓库名（例如: "pat-test/small"）
    const repoName: string = project.value.name;
    const segments = repoName.split("/");
    const encodedRepo = segments.map(encodeURIComponent).join("/");
    const basePath = `/v2/${encodedRepo}`;

    const headers: Record<string, string> = {};
    if (userStore.token) {
      headers["Authorization"] = `Bearer ${userStore.token}`;
    }

    // 1. 获取 tags 列表
    const tagsResp = await fetch(`${basePath}/tags/list`, { headers });
    if (!tagsResp.ok) {
      console.error("Failed to fetch tags:", await tagsResp.text());
      images.value = [];
      return;
    }
    const tagsJson = await tagsResp.json();
    const tagsData = (tagsJson && (tagsJson.data || tagsJson)) as any;
    const tags: string[] = Array.isArray(tagsData?.tags) ? tagsData.tags : [];

    // 从 /v2/.../tags/list 读取后端补充的 tag 级统计信息：
    // - tag_sizes:   { [tag]: size(bytes) }
    // - tag_digests: { [tag]: digest }
    const tagSizes: Record<string, number> =
      (tagsData?.tag_sizes as Record<string, number>) ||
      (tagsData?.tagSizes as Record<string, number>) ||
      {};
    const tagDigests: Record<string, string> =
      (tagsData?.tag_digests as Record<string, string>) ||
      (tagsData?.tagDigests as Record<string, string>) ||
      {};
    const tagPushTimes: Record<string, string> =
      (tagsData?.tag_push_times as Record<string, string>) ||
      (tagsData?.tagPushTimes as Record<string, string>) ||
      {};
    const tagPushedBy: Record<string, string> =
      (tagsData?.tag_pushed_by as Record<string, string>) ||
      (tagsData?.tagPushedBy as Record<string, string>) ||
      {};

    const result: ProjectImage[] = [];
    let totalSize = 0;

    // 2. 逐个 tag 组装镜像信息：
    //    - 优先使用后端预计算的 tag_sizes / tag_digests，避免多次请求 manifest 且修复 index manifest 下 size 始终为 0 的问题；
    //    - 若后端暂未提供详细信息，则回退到按 manifest.layers 计算。
    for (const tag of tags) {
      try {
        let digest = tagDigests[tag] || "";
        let size = typeof tagSizes[tag] === "number" ? tagSizes[tag] : 0;

        // 仅当 digest 或 size 缺失时才请求 manifest，减少不必要的 Registry 调用
        if (!digest || size <= 0) {
          const manifestResp = await fetch(
            `${basePath}/manifests/${encodeURIComponent(tag)}`,
            {
              headers: {
                ...headers,
                Accept: "application/vnd.docker.distribution.manifest.v2+json",
              },
            },
          );
          if (!manifestResp.ok) {
            console.warn(
              "Failed to fetch manifest for tag",
              tag,
              await manifestResp.text(),
            );
            continue;
          }

          if (!digest) {
            digest = manifestResp.headers.get("Docker-Content-Digest") || "";
          }
          if (size <= 0) {
            const manifestBody = await manifestResp.json();
            const manifest = (manifestBody &&
              (manifestBody.data || manifestBody)) as any;

            // 根据 manifest.layers 计算总大小（部分镜像为 index manifest，可能不包含 layers，此时代码会保持 size=0）
            const layers = Array.isArray(manifest?.layers)
              ? manifest.layers
              : [];
            size = layers.reduce(
              (sum: number, l: any) =>
                sum + (typeof l?.size === "number" ? l.size : 0),
              0,
            );
          }
        }

        result.push({
          name: tag,
          digest,
          size,
          // push 时间和用户优先使用后端 webhook 统计信息，若无则保持为 null 以便界面展示“未知”
          pushTime: tagPushTimes[tag] || null,
          pushedBy: tagPushedBy[tag] || null,
        });
        totalSize += size;
      } catch (e) {
        console.error("Failed to load manifest for tag", tag, e);
      }
    }

    images.value = result;
    // 前端兜底刷新项目统计信息，避免列表/仪表盘长期显示为 0
    if (project.value?.id) {
      projectStore.updateProjectStats(project.value.id, {
        imageCount: result.length,
        storageUsed: totalSize,
      });
    }
  } finally {
    if (showLoading) {
      isImagesLoading.value = false;
    }
  }
}

onMounted(async () => {
  const projectId = route.params.id as string;
  await projectStore.fetchProject(projectId);
  project.value = projectStore.currentProject;

  // 加载该项目下的实际镜像版本
  await loadImages();

  // 建立 SSE 连接，实时监听当前项目相关的 Registry 事件
  if (typeof window !== "undefined" && !registryEventSource) {
    const base = window.location.origin || "";
    // 使用统一的 /api 前缀，后端在 WebhookController 中注册了 /api/v1/stream/registry
    registryEventSource = new EventSource(`${base}/api/v1/stream/registry`);

    registryEventSource.addEventListener("registry", (evt: MessageEvent) => {
      try {
        const data = JSON.parse(evt.data) as {
          type: "push" | "delete";
          repository: string;
          projectId?: string;
        };

        // 仅当事件与当前项目仓库名称匹配时才触发刷新
        if (!project.value || !project.value.name) return;
        if (data.repository !== project.value.name) return;

        // 事件来自同一仓库：执行一次轻量刷新，保持界面与 Registry 同步
        void loadImages(false);
      } catch {
        // 忽略解析错误，避免终止事件流
      }
    });

    registryEventSource.onerror = () => {
      // 避免不断重试导致资源浪费，在出错时关闭当前 SSE；
      // 下一次进入详情页时会重新建立连接。
      registryEventSource?.close();
      registryEventSource = null;
    };
  }
});

onUnmounted(() => {
  stopAutoRefresh();
  if (registryEventSource) {
    registryEventSource.close();
    registryEventSource = null;
  }
});

function startAutoRefresh() {
  if (autoRefreshTimer != null) return;
  autoRefreshTimer = window.setInterval(() => {
    // 避免并发重复加载
    if (!isImagesLoading.value && project.value?.name) {
      void loadImages(false);
    }
  }, AUTO_REFRESH_INTERVAL);
}

function stopAutoRefresh() {
  if (autoRefreshTimer != null) {
    clearInterval(autoRefreshTimer);
    autoRefreshTimer = null;
  }
}

watch(autoRefreshEnabled, (enabled) => {
  if (enabled) {
    // 开启时先刷新一次，再进入轮询
    void loadImages(false);
    startAutoRefresh();
  } else {
    stopAutoRefresh();
  }
});

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString("zh-CN");
}

// 镜像管理 & Docker 使用辅助
function handlePullImage(image: any) {
  // 项目名即仓库名（如 "pat-test/small"），image.name 为 tag
  const repo = project.value?.name || "project";
  const pullCommand = `docker pull ${repo}:${image.name}`;
  copyToClipboard(pullCommand)
    .then(() => {
      openMessageDialog("复制成功", `拉取命令已复制到剪贴板:\n${pullCommand}`);
    })
    .catch((err: any) => {
      console.error("复制拉取命令失败", err);
      openMessageDialog(
        "复制失败",
        err?.message || "复制拉取命令到剪贴板失败，请尝试手动复制",
      );
    });

  // 写入通知中心：拉取命令复制
  notificationStore.addNotification({
    source: "registry",
    title: "拉取镜像命令已生成",
    message: `已为镜像 ${repo}:${image.name} 生成并复制 docker pull 命令`,
    status: "success",
  });
}

function handlePushImageHelp() {
  if (!project.value?.name) {
    openMessageDialog("操作失败", "项目名称未加载完成，请稍后重试");
    return;
  }

  const host = window.location.host || "localhost:8080";
  const repo = project.value.name;
  const sample = [
    "# 1. 登录 Registry（如已登录可跳过）",
    `docker login ${host}`,
    "",
    "# 2. 为本地镜像打标签并推送到当前项目",
    `docker tag your-image:tag ${host}/${repo}:your-tag`,
    `docker push ${host}/${repo}:your-tag`,
  ].join("\n");

  copyToClipboard(sample)
    .then(() => {
      openMessageDialog(
        "复制成功",
        `推送镜像参考命令已复制到剪贴板：\n\n${sample}`,
      );
    })
    .catch((err: any) => {
      console.error("复制推送命令失败", err);
      openMessageDialog(
        "复制失败",
        err?.message || "复制推送镜像命令失败，请尝试手动复制",
      );
    });

  notificationStore.addNotification({
    source: "registry",
    title: "推送镜像命令已生成",
    message: `已为项目 ${repo} 生成并复制 docker push 参考命令`,
    status: "success",
  });
}

const showImportDialog = ref(false);
const importForm = ref<ImportFormState>({
  source_url: "",
  target_image: "",
  target_tag: "",
  auth: {
    username: "",
    password: "",
  },
});
const isImporting = ref(false);
const importTask = ref<ImportTask | null>(null);
const importTaskPolling = ref<number | null>(null);

function openImportFromUrl() {
  importForm.value = {
    source_url: "",
    target_image: "",
    target_tag: "",
    auth: {
      username: "",
      password: "",
    },
  };
  importTask.value = null;
  showImportDialog.value = true;
}

async function handleImportFromUrl() {
  if (!importForm.value.source_url || !project.value?.id) {
    openMessageDialog("校验失败", "请输入有效的镜像URL");
    return;
  }

  isImporting.value = true;
  try {
    const task = await imageImportApi.importImage(project.value.id, {
      source_url: importForm.value.source_url,
      target_image: importForm.value.target_image || undefined,
      target_tag: importForm.value.target_tag || undefined,
      auth: importForm.value.auth?.username
        ? {
            username: importForm.value.auth.username,
            password: importForm.value.auth.password,
          }
        : undefined,
    });

    importTask.value = task;
    notificationStore.addNotification({
      source: "registry",
      title: "镜像导入任务已创建",
      message: `已开始导入镜像 ${importForm.value.source_url}`,
      status: "success",
    });

    // 开始轮询任务状态
    startTaskPolling(task.task_id);
  } catch (error: any) {
    console.error("导入镜像失败", error);
    openMessageDialog(
      "导入失败",
      error.response?.data?.message || error.message || "导入镜像失败，请稍后重试"
    );
    notificationStore.addNotification({
      source: "registry",
      title: "镜像导入失败",
      message: error.response?.data?.message || error.message || "导入镜像失败",
      status: "failed",
    });
  } finally {
    isImporting.value = false;
  }
}

function startTaskPolling(taskId: string) {
  if (importTaskPolling.value) {
    clearInterval(importTaskPolling.value);
  }

  importTaskPolling.value = window.setInterval(async () => {
    if (!project.value?.id || !taskId) return;

    try {
      const task = await imageImportApi.getTask(project.value.id, taskId);
      importTask.value = task;

      if (task.status === "success" || task.status === "failed") {
        stopTaskPolling();
        if (task.status === "success") {
          notificationStore.addNotification({
            source: "registry",
            title: "镜像导入完成",
            message: `镜像 ${importForm.value.source_url} 已成功导入`,
            status: "success",
          });
          // 刷新镜像列表
          await loadImages();
        } else {
          notificationStore.addNotification({
            source: "registry",
            title: "镜像导入失败",
            message: task.error || "镜像导入失败",
            status: "failed",
          });
        }
      }
    } catch (error) {
      console.error("获取任务状态失败", error);
      stopTaskPolling();
    }
  }, 2000); // 每2秒轮询一次
}

function stopTaskPolling() {
  if (importTaskPolling.value) {
    clearInterval(importTaskPolling.value);
    importTaskPolling.value = null;
  }
}

function closeImportDialog() {
  stopTaskPolling();
  showImportDialog.value = false;
  importTask.value = null;
}

function handleDeleteImage(image: any) {
  imageToDelete.value = image;
  showDeleteImageDialog.value = true;
}

async function confirmDeleteImage() {
  if (!project.value?.name || !imageToDelete.value) return;

  if (!isOwner.value && !userStore.isAdmin) {
    openMessageDialog("权限不足", "仅项目所有者或管理员可以删除镜像");
    return;
  }

  // 与 loadImages 复用相同的仓库路径编码逻辑
  const repoName: string = project.value.name;
  const segments = repoName.split("/");
  const encodedRepo = segments.map(encodeURIComponent).join("/");
  const basePath = `/v2/${encodedRepo}`;

  const headers: Record<string, string> = {};
  if (userStore.token) {
    headers["Authorization"] = `Bearer ${userStore.token}`;
  }

  try {
    // 前端按“镜像版本(tag)”维度删除：使用 tag 名称作为 reference，
    // 后端在 DeleteManifest 中会仅删除该 tag 映射，并在无其他引用时再清理底层 manifest。
    const tag: string = imageToDelete.value.name;
    const resp = await fetch(
      `${basePath}/manifests/${encodeURIComponent(tag)}`,
      {
        method: "DELETE",
        headers,
      },
    );

    if (!resp.ok) {
      const text = await resp.text();
      console.error("Failed to delete image", tag, resp.status, text);
      openMessageDialog(
        "删除失败",
        `删除镜像失败（HTTP ${resp.status}），请稍后重试`,
      );
      return;
    }

    showDeleteImageDialog.value = false;
    imageToDelete.value = null;
    await loadImages();

    notificationStore.addNotification({
      source: "registry",
      title: "镜像版本已删除",
      message: `项目 ${project.value.name} 中的镜像版本 ${tag} 已被删除`,
      status: "success",
    });
  } catch (err: any) {
    console.error("Failed to delete image", err);
    openMessageDialog("删除失败", err?.message || "删除镜像失败，请稍后重试");
  }
}

async function handleRefreshImages() {
  await loadImages();
  if (project.value?.name) {
    notificationStore.addNotification({
      source: "registry",
      title: "镜像列表已刷新",
      message: `项目 ${project.value.name} 的镜像列表已从 Registry 重新加载`,
      status: "info",
    });
  }
}

function handleDeleteProject() {
  showDeleteProjectDialog.value = true;
}

// 项目设置
const editForm = ref({
  name: "",
  description: "",
  isPublic: false,
  storageQuota: 0,
});

function openSettings() {
  if (project.value) {
    editForm.value = {
      name: project.value.name,
      description: project.value.description || "",
      isPublic: project.value.isPublic,
      storageQuota: project.value.storageQuota,
    };
  }
  activeTab.value = "settings";
}

async function handleSaveSettings() {
  if (!project.value) {
    openMessageDialog("保存失败", "项目信息未加载完成，请稍后重试");
    return;
  }

  const payload = {
    name: editForm.value.name?.trim?.() || editForm.value.name,
    description:
      editForm.value.description?.trim?.() || editForm.value.description,
    isPublic: editForm.value.isPublic,
    storageQuota: editForm.value.storageQuota,
  };

  try {
    const before = project.value;
    const updated = await projectStore.updateProject(project.value.id, payload);
    if (updated) {
      project.value = updated as any;
    }
    // 生成更精确的变更描述
    const changes: string[] = [];
    if (before.name !== payload.name) {
      changes.push("名称");
    }
    if ((before.description || "") !== (payload.description || "")) {
      changes.push("描述");
    }
    if (before.isPublic !== payload.isPublic) {
      changes.push("可见性");
    }
    if ((before.storageQuota || 0) !== (payload.storageQuota || 0)) {
      changes.push("存储配额");
    }
    const changeText = changes.length > 0 ? changes.join("、") : "配置";

    openMessageDialog("保存成功", `项目设置已保存（更新项：${changeText}）`);
    notificationStore.addNotification({
      source: "project",
      title: "项目设置已更新",
      message: `项目「${project.value.name}」的${changeText}已更新`,
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog(
      "保存失败",
      err?.message || "项目设置保存失败，请稍后重试",
    );
  }
}
</script>

<template>
  <div class="project-detail-page">
    <div v-if="project" class="project-content">
      <!-- 项目头部 -->
      <div class="project-header">
        <div class="header-left">
          <button class="back-btn" @click="$router.back()">
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"
              />
            </svg>
          </button>
          <div class="project-info">
            <h1 class="project-name">
              {{ project.name }}
            </h1>
            <p class="project-description">
              {{ project.description || "暂无描述" }}
            </p>
          </div>
        </div>
        <div class="header-actions">
          <span class="visibility-badge" :class="{ public: project.isPublic }">
            {{ project.isPublic ? "公开" : "私有" }}
          </span>
        </div>
      </div>

      <!-- 项目统计 -->
      <div class="stats-row">
        <div class="stat-item">
          <div class="stat-value">
            {{ images.length }}
          </div>
          <div class="stat-label">镜像版本</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">
            {{ formatBytes(project.storageUsed || 0) }}
          </div>
          <div class="stat-label">已用存储</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">
            {{ formatBytes(project.storageQuota || 0) }}
          </div>
          <div class="stat-label">配额</div>
        </div>
      </div>

      <!-- 标签页 -->
      <div class="tabs">
        <button
          class="tab"
          :class="{ active: activeTab === 'images' }"
          @click="activeTab = 'images'"
        >
          镜像版本
        </button>
        <button
          class="tab"
          :class="{ active: activeTab === 'settings' }"
          @click="openSettings"
        >
          项目设置
        </button>
      </div>

      <!-- 镜像列表 -->
      <div v-if="activeTab === 'images'" class="tab-content">
        <div class="images-header">
          <h2>镜像版本</h2>
          <div class="images-actions">
            <CypButton type="primary" size="small" @click="handlePushImageHelp">
              推送镜像
            </CypButton>
            <CypButton
              size="small"
              style="margin-left: 8px"
              @click="openImportFromUrl"
            >
              从 URL 添加
            </CypButton>
            <CypButton
              size="small"
              type="default"
              style="margin-left: 8px"
              :loading="isImagesLoading"
              @click="handleRefreshImages"
            >
              刷新列表
            </CypButton>
            <label class="auto-refresh-toggle">
              <input v-model="autoRefreshEnabled" type="checkbox" />
              自动刷新(30s)
            </label>
          </div>
        </div>
        <div class="image-list">
          <div v-for="image in images" :key="image.name" class="image-item">
            <div class="image-info">
              <div class="image-name">
                {{ image.name }}
              </div>
              <div class="image-digest">
                {{ image.digest }}
              </div>
            </div>
            <div class="image-meta">
              <span>{{ formatBytes(image.size) }}</span>
              <span class="separator">•</span>
              <span>
                {{
                  image.pushTime
                    ? formatDate(image.pushTime)
                    : project?.updatedAt
                      ? formatDate(project.updatedAt)
                      : "未知时间"
                }}
              </span>
              <span class="separator">•</span>
              <span>{{ image.pushedBy || "未知用户" }}</span>
            </div>
            <div class="image-actions">
              <CypButton size="small" @click="handlePullImage(image)">
                拉取
              </CypButton>
              <CypButton
                size="small"
                type="danger"
                @click="handleDeleteImage(image)"
              >
                删除
              </CypButton>
            </div>
          </div>
        </div>
      </div>

      <!-- 项目设置 -->
      <div v-if="activeTab === 'settings'" class="tab-content">
        <div class="settings-section">
          <h3>基本设置</h3>
          <div class="form-group">
            <label>项目名称</label>
            <CypInput v-model="editForm.name" placeholder="输入项目名称" />
          </div>
          <div class="form-group">
            <label>描述</label>
            <textarea
              v-model="editForm.description"
              class="textarea"
              placeholder="输入项目描述"
              rows="3"
            />
          </div>
          <div class="form-group">
            <label>可见性</label>
            <div class="visibility-toggle">
              <CypCheckbox v-model="editForm.isPublic">
                公开项目（允许匿名拉取）
              </CypCheckbox>
            </div>
          </div>
          <div class="form-group">
            <label>存储配额</label>
            <CypSelect
              v-model="editForm.storageQuota"
              :options="[
                { value: 5 * 1024 * 1024 * 1024, label: '5 GB' },
                { value: 10 * 1024 * 1024 * 1024, label: '10 GB' },
                { value: 50 * 1024 * 1024 * 1024, label: '50 GB' },
                { value: 100 * 1024 * 1024 * 1024, label: '100 GB' },
              ]"
            />
          </div>
          <CypButton type="primary" @click="handleSaveSettings">
            保存更改
          </CypButton>
        </div>

        <div v-if="isOwner" class="settings-section danger-zone">
          <h3>危险操作</h3>
          <p>删除项目将永久移除所有镜像和配置，此操作无法撤销。</p>
          <CypButton type="danger" @click="handleDeleteProject">
            删除项目
          </CypButton>
        </div>
      </div>
    </div>

    <!-- 删除镜像确认对话框 -->
    <CypDialog
      v-model="showDeleteImageDialog"
      title="删除镜像"
      width="420px"
      @close="showDeleteImageDialog = false"
    >
      <div v-if="imageToDelete" class="dialog-form">
        <p>
          确定要删除镜像 "<strong>{{ imageToDelete.name }}</strong
          >" 吗？
        </p>
        <p class="danger-text">此操作不可撤销，相关版本将无法再被拉取。</p>
      </div>
      <template #footer>
        <CypButton @click="showDeleteImageDialog = false"> 取消 </CypButton>
        <CypButton type="danger" @click="confirmDeleteImage">
          确认删除
        </CypButton>
      </template>
    </CypDialog>

    <!-- 删除项目确认对话框 -->
    <CypDialog
      v-model="showDeleteProjectDialog"
      title="删除项目"
      width="480px"
      @close="showDeleteProjectDialog = false"
    >
      <div v-if="project" class="dialog-form">
        <p>
          确定要删除项目 "<strong>{{ project.name }}</strong
          >" 吗？
        </p>
        <p class="danger-text">此操作将永久删除所有镜像和配置，且无法恢复。</p>
      </div>
      <template #footer>
        <CypButton @click="showDeleteProjectDialog = false"> 取消 </CypButton>
        <CypButton
          type="danger"
          @click="
            async () => {
              if (!project) return;
              await projectStore.deleteProject(project.id);
              showDeleteProjectDialog = false;
              notificationStore.addNotification({
                source: 'project',
                title: '项目已删除',
                message: `项目「${project.name}」及其下的镜像已被删除`,
                status: 'success',
              });
              router.push('/projects');
            }
          "
        >
          确认删除
        </CypButton>
      </template>
    </CypDialog>

    <!-- 从 URL 添加镜像对话框 -->
    <CypDialog
      v-model="showImportDialog"
      title="从 URL 添加镜像"
      width="600px"
      @close="closeImportDialog"
    >
      <div class="dialog-content">
        <div class="form-group" style="margin-bottom: 16px">
          <label>镜像URL *</label>
          <CypInput
            v-model="importForm.source_url"
            placeholder="例如: docker.io/library/nginx:latest 或 nginx:latest"
            :disabled="isImporting || importTask !== null"
          />
          <small style="color: #64748b; margin-top: 4px; display: block">
            支持 Docker Hub、GHCR 等公共仓库的镜像URL
          </small>
        </div>

        <div class="form-group" style="margin-bottom: 16px">
          <label>目标镜像名称（可选）</label>
          <CypInput
            v-model="importForm.target_image"
            placeholder="留空则使用源镜像名称"
            :disabled="isImporting || importTask !== null"
          />
        </div>

        <div class="form-group" style="margin-bottom: 16px">
          <label>目标标签（可选）</label>
          <CypInput
            v-model="importForm.target_tag"
            placeholder="留空则使用源镜像标签"
            :disabled="isImporting || importTask !== null"
          />
        </div>

        <div class="form-group" style="margin-bottom: 16px">
          <label>认证信息（可选，私有镜像需要）</label>
          <CypInput
            v-model="importForm.auth!.username"
            placeholder="用户名"
            style="margin-bottom: 8px"
            :disabled="isImporting || importTask !== null"
          />
          <CypInput
            v-model="importForm.auth!.password"
            type="password"
            placeholder="密码/Token"
            :disabled="isImporting || importTask !== null"
          />
        </div>

        <!-- 任务进度显示 -->
        <div v-if="importTask" class="task-progress" style="margin-top: 20px; padding: 16px; background: #f8fafc; border-radius: 8px">
          <div style="display: flex; justify-content: space-between; margin-bottom: 8px">
            <span style="font-weight: 500">任务状态: {{ importTask.status }}</span>
            <span style="color: #64748b">{{ importTask.progress }}%</span>
          </div>
          <div style="width: 100%; height: 8px; background: #e2e8f0; border-radius: 4px; overflow: hidden; margin-bottom: 8px">
            <div
              :style="{
                width: importTask.progress + '%',
                height: '100%',
                background: importTask.status === 'failed' ? '#ef4444' : '#3b82f6',
                transition: 'width 0.3s',
              }"
            ></div>
          </div>
          <div style="font-size: 13px; color: #64748b">{{ importTask.message }}</div>
          <div v-if="importTask.error" style="margin-top: 8px; padding: 8px; background: #fee2e2; border-radius: 4px; color: #dc2626; font-size: 12px">
            {{ importTask.error }}
          </div>
        </div>

        <div class="dialog-actions" style="margin-top: 24px">
          <CypButton @click="closeImportDialog" :disabled="isImporting && importTask === null">
            取消
          </CypButton>
          <CypButton
            type="primary"
            @click="handleImportFromUrl"
            :loading="isImporting"
            :disabled="importTask !== null && (importTask.status === 'running' || importTask.status === 'pending')"
          >
            {{ importTask ? (importTask.status === 'success' ? '完成' : importTask.status === 'failed' ? '重试' : '导入中...') : '开始导入' }}
          </CypButton>
        </div>
      </div>
    </CypDialog>

    <!-- 通用提示框（信息/错误提示） -->
    <CypDialog
      v-model="showMessageDialog"
      :title="messageDialogTitle"
      width="360px"
      @close="showMessageDialog = false"
    >
      <p>{{ messageDialogContent }}</p>
      <template #footer>
        <CypButton type="primary" @click="showMessageDialog = false">
          知道了
        </CypButton>
      </template>
    </CypDialog>

    <div v-if="!project" class="loading">
      <CypLoading text="加载中..." />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.project-detail-page {
  max-width: 1200px;
  margin: 0 auto;
}

.project-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  color: #64748b;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: #f8fafc;
    color: #1e293b;
  }
}

.project-name {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 8px;
}

.project-description {
  font-size: 14px;
  color: #64748b;
  margin: 0;
}

.visibility-badge {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  background: #fee2e2;
  color: #ef4444;

  &.public {
    background: #dcfce7;
    color: #22c55e;
  }
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-item {
  background: white;
  padding: 20px;
  border-radius: 12px;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
  margin-top: 4px;
}

.tabs {
  display: flex;
  gap: 4px;
  margin-bottom: 24px;
  background: white;
  padding: 4px;
  border-radius: 10px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.tab {
  flex: 1;
  padding: 12px 20px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    color: #1e293b;
  }

  &.active {
    background: #6366f1;
    color: white;
  }
}

.tab-content {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.images-header,
.members-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h2 {
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
  }
}

.image-list,
.member-list {
  display: flex;
  flex-direction: column;
}

.image-item {
  display: flex;
  align-items: center;
  padding: 16px 0;
  border-bottom: 1px solid #f1f5f9;

  &:last-child {
    border-bottom: none;
  }
}

.image-info {
  flex: 1;
}

.image-name {
  font-size: 15px;
  font-weight: 500;
  color: #1e293b;
}

.image-digest {
  font-size: 12px;
  color: #64748b;
  font-family: monospace;
  margin-top: 4px;
}

.image-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #64748b;
  margin-right: 24px;
  .separator {
    color: #e2e8f0;
  }
}

.image-actions {
  display: flex;
  gap: 8px;
}

.auto-refresh-toggle {
  display: inline-flex;
  align-items: center;
  margin-left: 12px;
  font-size: 12px;
  color: #64748b;

  input {
    margin-right: 4px;
  }
}

.member-item {
  display: flex;
  align-items: center;
  padding: 16px 0;
  border-bottom: 1px solid #f1f5f9;

  &:last-child {
    border-bottom: none;
  }
}

.member-avatar {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  margin-right: 16px;
}

.member-info {
  flex: 1;
}

.member-name {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
}

.member-email {
  font-size: 13px;
  color: #64748b;
}

.role-badge {
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
  text-transform: capitalize;

  &.owner {
    background: #e0e7ff;
    color: #6366f1;
  }
  &.maintainer {
    background: #fef3c7;
    color: #d97706;
  }
  &.developer {
    background: #dcfce7;
    color: #22c55e;
  }
  &.guest {
    background: #f1f5f9;
    color: #64748b;
  }
}

.member-actions {
  display: flex;
  gap: 8px;
}

.settings-section {
  margin-bottom: 32px;

  h3 {
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
    margin: 0 0 16px;
  }
}

.form-group {
  margin-bottom: 16px;

  label {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: #374151;
    margin-bottom: 8px;
  }
}

.visibility-toggle {
  padding: 12px 0;
}

.input,
.textarea {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #1e293b;
  background: #f8fafc;

  &:focus {
    outline: none;
    border-color: #6366f1;
  }
}

.danger-zone {
  padding: 20px;
  background: #fef2f2;
  border-radius: 8px;
  border: 1px solid #fecaca;

  h3 {
    color: #dc2626;
  }

  p {
    font-size: 14px;
    color: #991b1b;
    margin-bottom: 16px;
  }
}

.dialog-form {
  padding: 0;
}

.danger-text {
  margin-top: 4px;
  font-size: 13px;
  color: #b91c1c;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 100px;
  color: #64748b;
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }

  .image-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .image-meta {
    margin-right: 0;
  }

  .member-item {
    flex-wrap: wrap;
    gap: 12px;
  }

  .member-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
