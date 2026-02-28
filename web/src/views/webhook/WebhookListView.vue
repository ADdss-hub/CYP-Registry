<script setup lang="ts">
import { ref, onMounted, computed } from "vue";
import { useProjectStore } from "@/stores/project";
import { useNotificationStore } from "@/stores/notification";
import { webhookApi } from "@/services/webhook";
import CypButton from "@/components/common/CypButton.vue";
import CypInput from "@/components/common/CypInput.vue";
import CypSelect from "@/components/common/CypSelect.vue";
import CypDialog from "@/components/common/CypDialog.vue";
import CypCard from "@/components/common/CypCard.vue";
import CypSwitch from "@/components/common/CypSwitch.vue";
import CypTag from "@/components/common/CypTag.vue";
import type { Webhook } from "@/types";

const projectStore = useProjectStore();
const notificationStore = useNotificationStore();

// 对话框状态
const showCreateDialog = ref(false);
const showEditDialog = ref(false);
const showTestDialog = ref(false);
const showDetailDialog = ref(false);
const selectedWebhook = ref<Webhook | null>(null);
const testResult = ref<any>(null);
const isTesting = ref(false);

// 删除确认 & 通用提示框（替代浏览器 confirm/alert，遵循界面规范3.3/3.4节）
const showDeleteConfirmDialog = ref(false);
const webhookToDelete = ref<Webhook | null>(null);
const showMessageDialog = ref(false);
const messageDialogTitle = ref("");
const messageDialogContent = ref("");

function openMessageDialog(title: string, content: string) {
  messageDialogTitle.value = title;
  messageDialogContent.value = content;
  showMessageDialog.value = true;
}

// 新建Webhook表单
const newWebhook = ref({
  projectId: "",
  name: "",
  description: "",
  url: "",
  secret: "",
  events: [] as string[],
  headers: {} as Record<string, string>,
});

// 测试负载
const testPayload = ref({
  eventType: "push",
  payload: {},
});

// Webhook列表（从后端实时加载）
const webhooks = ref<Webhook[]>([]);

// 全局统计数据（来自后端 /v1/webhooks/statistics）
const rawStatistics = ref<
  import("@/services/webhook").WebhookStatistics | null
>(null);

// 事件类型
const eventTypes = [
  {
    value: "push",
    label: "镜像推送",
    description: "当镜像被推送到仓库时触发",
    icon: "📤",
  },
  {
    value: "pull",
    label: "镜像拉取",
    description: "当镜像被拉取时触发",
    icon: "📥",
  },
  {
    value: "delete",
    label: "镜像删除",
    description: "当镜像被删除时触发",
    icon: "🗑️",
  },
  {
    value: "scan",
    label: "扫描完成",
    description: "当安全扫描任务完成时触发",
    icon: "🔍",
  },
  {
    value: "scan_fail",
    label: "扫描失败",
    description: "当安全扫描任务失败时触发",
    icon: "❌",
  },
];

// 项目选项
const projectOptions = computed(() =>
  projectStore.projects.map((p) => ({
    value: p.id,
    label: p.name,
  })),
);

// 统计信息（优先使用后端统计结果，失败时回退为前端计算）
const statistics = computed(() => ({
  total: rawStatistics.value?.totalWebhooks ?? webhooks.value.length,
  active:
    rawStatistics.value?.activeWebhooks ??
    webhooks.value.filter((w) => w.isActive).length,
  totalTriggers:
    rawStatistics.value?.totalEvents ??
    webhooks.value.reduce(
      (sum, w: any) => sum + (w.successCount || 0) + (w.failedCount || 0),
      0,
    ),
}));

onMounted(async () => {
  await projectStore.fetchProjects();
  if (projectStore.projects.length > 0 && !newWebhook.value.projectId) {
    newWebhook.value.projectId = projectStore.projects[0].id;
  }
  // 初次加载当前项目下的 Webhook 列表
  if (newWebhook.value.projectId) {
    await loadWebhooks(newWebhook.value.projectId);
  }
  // 加载全局统计数据，确保顶部卡片展示真实统计
  try {
    const statsResult = await webhookApi.getStatistics();
    rawStatistics.value = statsResult;
  } catch (err) {
    // 统计获取失败时不打断页面，仅使用本地计算的兜底数据
    console.error("Failed to fetch webhook statistics:", err);
  }
});

// 辅助函数
function formatDate(dateStr?: string): string {
  if (!dateStr) return "-";
  const date = new Date(dateStr);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60000) return "刚刚";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
  return date.toLocaleDateString("zh-CN");
}

// 切换事件选择
function toggleEvent(event: string) {
  const index = newWebhook.value.events.indexOf(event);
  if (index === -1) {
    newWebhook.value.events.push(event);
  } else {
    newWebhook.value.events.splice(index, 1);
  }
}

function isEventSelected(event: string): boolean {
  return newWebhook.value.events.includes(event);
}

// 加载指定项目下的 Webhook 列表
async function loadWebhooks(projectId: string) {
  if (!projectId) return;
  try {
    const result = await webhookApi.getWebhooks(projectId);
    webhooks.value = result || [];
  } catch (err: any) {
    openMessageDialog("加载失败", err?.message || "加载 Webhook 列表失败");
  }
}

// 创建Webhook
async function handleCreateWebhook() {
  if (
    !newWebhook.value.name ||
    !newWebhook.value.url ||
    newWebhook.value.events.length === 0
  ) {
    openMessageDialog("校验失败", "请填写完整的Webhook配置");
    return;
  }

  try {
    const result = await webhookApi.createWebhook(newWebhook.value);
    if (result) {
      webhooks.value.unshift(result);
    }
    showCreateDialog.value = false;
    resetNewWebhook();
    notificationStore.addNotification({
      source: "webhook",
      title: "Webhook 已创建",
      message: `Webhook「${result.name}」已创建，用于项目事件通知`,
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("创建失败", err.message || "创建失败");
  }
}

function resetNewWebhook() {
  newWebhook.value = {
    projectId: projectStore.projects[0]?.id || "",
    name: "",
    description: "",
    url: "",
    secret: "",
    events: [],
    headers: {},
  };
}

// 编辑Webhook
function openEditDialog(webhook: Webhook) {
  selectedWebhook.value = { ...webhook };
  showEditDialog.value = true;
}

// 保存编辑
async function handleSaveWebhook() {
  if (!selectedWebhook.value) return;

  try {
    const updated = await webhookApi.updateWebhook(
      selectedWebhook.value.webhookId,
      selectedWebhook.value,
    );
    const index = webhooks.value.findIndex(
      (w) => w.webhookId === selectedWebhook.value!.webhookId,
    );
    if (index !== -1) {
      webhooks.value[index] = {
        ...(updated || selectedWebhook.value),
        updatedAt: new Date().toISOString(),
      };
    }
    showEditDialog.value = false;
    notificationStore.addNotification({
      source: "webhook",
      title: "Webhook 已更新",
      message: `Webhook「${selectedWebhook.value.name}」配置已保存`,
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("保存失败", err.message || "保存失败");
  }
}

// 切换启用状态
async function toggleWebhookStatus(webhook: Webhook) {
  try {
    await webhookApi.updateWebhook(webhook.webhookId, {
      isActive: !webhook.isActive,
    });
    webhook.isActive = !webhook.isActive;
    webhook.updatedAt = new Date().toISOString();
    notificationStore.addNotification({
      source: "webhook",
      title: webhook.isActive ? "Webhook 已启用" : "Webhook 已禁用",
      message: `Webhook「${webhook.name}」已${webhook.isActive ? "启用" : "禁用"}`,
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("操作失败", err.message || "状态切换失败");
  }
}

// 测试Webhook
function openTestDialog(webhook: Webhook) {
  selectedWebhook.value = webhook;
  testResult.value = null;
  testPayload.value = {
    eventType: webhook.events[0] || "push",
    payload: {},
  };
  showTestDialog.value = true;
}

async function submitTest() {
  if (!selectedWebhook.value) return;

  isTesting.value = true;
  testResult.value = null;

  try {
    const result = await webhookApi.testWebhook(
      selectedWebhook.value.webhookId,
      testPayload.value,
    );
    testResult.value = result;
    notificationStore.addNotification({
      source: "webhook",
      title: "Webhook 测试成功",
      message: `Webhook「${selectedWebhook.value.name}」测试已发送，状态码 ${result.responseStatus}`,
      status: "success",
    });
  } catch (err: any) {
    // 统一处理错误信息并做简单本地化
    const raw =
      err?.payload?.message ||
      err?.response?.data?.message ||
      err?.message ||
      "测试失败";

    let localized = raw;
    if (typeof raw === "string") {
      if (raw.includes("Webhook not found")) {
        localized = "Webhook 不存在或已被删除";
      } else if (raw.startsWith("Failed to test webhook")) {
        localized = "测试 Webhook 失败，请检查回调地址和网络连接";
      }
    }

    testResult.value = { error: localized };
    notificationStore.addNotification({
      source: "webhook",
      title: "Webhook 测试失败",
      message: localized,
      status: "failed",
    });
  } finally {
    isTesting.value = false;
  }
}

// 查看详情：在打开弹窗前先加载最新统计数据，确保触发次数/最近触发时间为实时值
async function openDetailDialog(webhook: Webhook) {
  selectedWebhook.value = webhook;
  showDetailDialog.value = true;

  try {
    const latest = await webhookApi.getWebhook(webhook.webhookId);
    selectedWebhook.value = latest;
  } catch (err) {
    console.error("Failed to fetch latest webhook detail", err);
    // 加载失败时保留列表中的兜底数据，避免打断用户查看
  }
}

// 删除Webhook
async function handleDeleteWebhook(webhookId: string) {
  const found = webhooks.value.find((w) => w.webhookId === webhookId) || null;
  webhookToDelete.value = found;
  showDeleteConfirmDialog.value = true;
}

// 确认删除（在确认弹窗中调用）
async function confirmDeleteWebhook() {
  if (!webhookToDelete.value) return;
  try {
    await webhookApi.deleteWebhook(webhookToDelete.value.webhookId);
    webhooks.value = webhooks.value.filter(
      (w) => w.webhookId !== webhookToDelete.value!.webhookId,
    );
    showDeleteConfirmDialog.value = false;
    webhookToDelete.value = null;
    notificationStore.addNotification({
      source: "webhook",
      title: "Webhook 已删除",
      message: "选中的 Webhook 已被删除",
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("删除失败", err?.message || "删除失败");
  }
}

// 获取事件标签
function getEventLabel(event: string): string {
  return eventTypes.find((e) => e.value === event)?.label || event;
}
</script>

<template>
  <div class="webhook-page">
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">Webhook管理</h2>
        <p class="page-subtitle">
          配置外部系统的事件通知，实现与CI/CD、监控等系统的集成
        </p>
      </div>
      <CypButton type="primary" @click="showCreateDialog = true">
        <svg
          viewBox="0 0 24 24"
          width="16"
          height="16"
          style="margin-right: 6px"
        >
          <path fill="currentColor" d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z" />
        </svg>
        创建Webhook
      </CypButton>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon total">
          <svg viewBox="0 0 24 24" width="24" height="24">
            <path
              fill="currentColor"
              d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 14H4V8l8 5 8-5v10zm-8-7L4 6h16l-8 5z"
            />
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">
            {{ statistics.total }}
          </div>
          <div class="stat-label">总Webhook</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon active">
          <svg viewBox="0 0 24 24" width="24" height="24">
            <path
              fill="currentColor"
              d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"
            />
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">
            {{ statistics.active }}
          </div>
          <div class="stat-label">已启用</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon triggers">
          <svg viewBox="0 0 24 24" width="24" height="24">
            <path
              fill="currentColor"
              d="M13 2.05v2.02c3.95.49 7 3.85 7 7.93 0 3.21-1.92 6-4.72 7.28L13 17v5h5l-1.22-1.22C19.91 19.07 22 15.76 22 12c0-5.18-3.95-9.45-9-9.95zM11 2.05C5.95 2.55 2 6.82 2 12c0 3.76 2.09 7.07 5.22 8.78L6 22h5v-5l-2.28 2.28C6.92 18 5 15.21 5 12c0-4.08 3.05-7.44 7-7.93V2.05z"
            />
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">
            {{ statistics.totalTriggers }}
          </div>
          <div class="stat-label">总触发次数</div>
        </div>
      </div>
    </div>

    <!-- Webhook列表 -->
    <div class="webhook-list">
      <CypCard
        v-for="webhook in webhooks"
        :key="webhook.webhookId"
        class="webhook-card"
      >
        <template #header>
          <div class="webhook-header">
            <div class="webhook-info">
              <h3 class="webhook-name">
                {{ webhook.name }}
              </h3>
              <p class="webhook-description">
                {{ webhook.description }}
              </p>
            </div>
            <CypSwitch
              :model-value="webhook.isActive"
              @update:model-value="toggleWebhookStatus(webhook)"
            />
          </div>
        </template>

        <div class="webhook-content">
          <div class="content-row">
            <span class="label">URL:</span>
            <code>{{ webhook.url }}</code>
          </div>

          <div class="content-row">
            <span class="label">事件:</span>
            <div class="event-tags">
              <CypTag
                v-for="event in webhook.events"
                :key="event"
                type="primary"
                size="small"
              >
                {{ getEventLabel(event) }}
              </CypTag>
            </div>
          </div>

          <div class="content-row stats">
            <div class="mini-stat">
              <span class="mini-value success">{{
                webhook.successCount || 0
              }}</span>
              <span class="mini-label">成功</span>
            </div>
            <div class="mini-stat">
              <span class="mini-value danger">{{
                webhook.failedCount || 0
              }}</span>
              <span class="mini-label">失败</span>
            </div>
            <div v-if="webhook.lastTriggeredAt" class="mini-stat">
              <span class="mini-value">{{
                formatDate(webhook.lastTriggeredAt)
              }}</span>
              <span class="mini-label">最近触发</span>
            </div>
          </div>
        </div>

        <template #footer>
          <div class="webhook-actions">
            <CypButton size="small" @click="openDetailDialog(webhook)">
              详情
            </CypButton>
            <CypButton size="small" @click="openTestDialog(webhook)">
              测试
            </CypButton>
            <CypButton size="small" @click="openEditDialog(webhook)">
              编辑
            </CypButton>
            <CypButton
              size="small"
              type="danger"
              @click="handleDeleteWebhook(webhook.webhookId)"
            >
              删除
            </CypButton>
          </div>
        </template>
      </CypCard>

      <div v-if="webhooks.length === 0" class="empty-state">
        <svg viewBox="0 0 24 24" width="64" height="64">
          <path
            fill="currentColor"
            d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 14H4V8l8 5 8-5v10zm-8-7L4 6h16l-8 5z"
          />
        </svg>
        <h3>暂无Webhook</h3>
        <p>创建Webhook以接收镜像仓库事件通知</p>
        <CypButton type="primary" @click="showCreateDialog = true">
          创建Webhook
        </CypButton>
      </div>
    </div>

    <!-- 创建Webhook对话框 -->
    <CypDialog
      v-model="showCreateDialog"
      title="创建Webhook"
      width="600px"
      @close="showCreateDialog = false"
    >
      <div class="dialog-form">
        <div class="form-group">
          <label>关联项目</label>
          <CypSelect
            v-model="newWebhook.projectId"
            :options="projectOptions"
            placeholder="选择项目"
          />
        </div>

        <div class="form-group">
          <label>名称 *</label>
          <CypInput v-model="newWebhook.name" placeholder="Webhook名称" />
        </div>

        <div class="form-group">
          <label>描述</label>
          <textarea
            v-model="newWebhook.description"
            class="textarea"
            placeholder="描述此Webhook的用途"
            rows="2"
          />
        </div>

        <div class="form-group">
          <label>回调URL *</label>
          <CypInput
            v-model="newWebhook.url"
            placeholder="https://example.com/webhook"
          />
        </div>

        <div class="form-group">
          <label>密钥</label>
          <CypInput
            v-model="newWebhook.secret"
            placeholder="用于生成签名的密钥（可选）"
            type="password"
          />
        </div>

        <div class="form-group">
          <label>事件类型 *</label>
          <div class="event-grid">
            <label
              v-for="event in eventTypes"
              :key="event.value"
              class="event-card"
              :class="{ selected: isEventSelected(event.value) }"
            >
              <input
                type="checkbox"
                :checked="isEventSelected(event.value)"
                @change="toggleEvent(event.value)"
              />
              <span class="event-icon">{{ event.icon }}</span>
              <span class="event-label">{{ event.label }}</span>
              <span class="event-desc">{{ event.description }}</span>
            </label>
          </div>
        </div>
      </div>
      <template #footer>
        <CypButton @click="showCreateDialog = false"> 取消 </CypButton>
        <CypButton type="primary" @click="handleCreateWebhook">
          创建
        </CypButton>
      </template>
    </CypDialog>

    <!-- 编辑Webhook对话框 -->
    <CypDialog
      v-model="showEditDialog"
      title="编辑Webhook"
      width="600px"
      @close="showEditDialog = false"
    >
      <div v-if="selectedWebhook" class="dialog-form">
        <div class="form-group">
          <label>名称 *</label>
          <CypInput v-model="selectedWebhook.name" placeholder="Webhook名称" />
        </div>

        <div class="form-group">
          <label>描述</label>
          <textarea
            v-model="selectedWebhook.description"
            class="textarea"
            placeholder="描述此Webhook的用途"
            rows="2"
          />
        </div>

        <div class="form-group">
          <label>回调URL *</label>
          <CypInput
            v-model="selectedWebhook.url"
            placeholder="https://example.com/webhook"
          />
        </div>

        <div class="form-group">
          <label>密钥</label>
          <CypInput
            v-model="selectedWebhook.secret"
            placeholder="留空保持原密钥不变"
            type="password"
          />
        </div>
      </div>
      <template #footer>
        <CypButton @click="showEditDialog = false"> 取消 </CypButton>
        <CypButton type="primary" @click="handleSaveWebhook"> 保存 </CypButton>
      </template>
    </CypDialog>

    <!-- 测试Webhook对话框 -->
    <CypDialog
      v-model="showTestDialog"
      title="测试Webhook"
      width="500px"
      @close="showTestDialog = false"
    >
      <div class="dialog-form">
        <div v-if="selectedWebhook" class="test-info">
          <p><strong>Webhook:</strong> {{ selectedWebhook.name }}</p>
          <p><strong>URL:</strong> {{ selectedWebhook.url }}</p>
        </div>

        <div class="form-group">
          <label>测试事件类型</label>
          <CypSelect
            v-model="testPayload.eventType"
            :options="
              eventTypes.map((e) => ({ value: e.value, label: e.label }))
            "
          />
        </div>

        <div
          v-if="testResult"
          class="test-result"
          :class="{ error: testResult.error }"
        >
          <template v-if="testResult.error">
            <span class="result-icon">❌</span>
            <span>{{ testResult.error }}</span>
          </template>
          <template v-else>
            <span class="result-icon">✅</span>
            <div>
              <p>状态码: {{ testResult.responseStatus }}</p>
              <p>耗时: {{ testResult.duration }}ms</p>
            </div>
          </template>
        </div>
      </div>
      <template #footer>
        <CypButton @click="showTestDialog = false"> 关闭 </CypButton>
        <CypButton type="primary" :loading="isTesting" @click="submitTest">
          发送测试
        </CypButton>
      </template>
    </CypDialog>

    <!-- 查看详情对话框 -->
    <CypDialog
      v-model="showDetailDialog"
      title="Webhook详情"
      width="600px"
      @close="showDetailDialog = false"
    >
      <div v-if="selectedWebhook" class="detail-content">
        <div class="detail-section">
          <h4>基本信息</h4>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-label">名称</span>
              <span class="detail-value">{{ selectedWebhook.name }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">状态</span>
              <CypTag :type="selectedWebhook.isActive ? 'success' : 'default'">
                {{ selectedWebhook.isActive ? "启用" : "禁用" }}
              </CypTag>
            </div>
            <div class="detail-item">
              <span class="detail-label">创建时间</span>
              <span class="detail-value">{{ selectedWebhook.createdAt }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">更新时间</span>
              <span class="detail-value">{{ selectedWebhook.updatedAt }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h4>回调配置</h4>
          <div class="detail-grid">
            <div class="detail-item full">
              <span class="detail-label">URL</span>
              <code>{{ selectedWebhook.url }}</code>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h4>触发统计</h4>
          <div class="stats-cards">
            <div class="stats-card success">
              <span class="stats-number">{{
                selectedWebhook.successCount || 0
              }}</span>
              <span class="stats-label">成功</span>
            </div>
            <div class="stats-card danger">
              <span class="stats-number">{{
                selectedWebhook.failedCount || 0
              }}</span>
              <span class="stats-label">失败</span>
            </div>
            <div v-if="selectedWebhook.lastTriggeredAt" class="stats-card">
              <span class="stats-number">{{
                formatDate(selectedWebhook.lastTriggeredAt)
              }}</span>
              <span class="stats-label">最近触发</span>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <CypButton @click="showDetailDialog = false"> 关闭 </CypButton>
      </template>
    </CypDialog>

    <!-- 删除Webhook确认对话框（系统框 + 确认提示框） -->
    <CypDialog
      v-model="showDeleteConfirmDialog"
      title="删除Webhook"
      width="480px"
      @close="showDeleteConfirmDialog = false"
    >
      <div v-if="webhookToDelete" class="confirm-content">
        <p>
          确定要删除 Webhook "<strong>{{ webhookToDelete.name }}</strong
          >" 吗？
        </p>
        <p class="warning">此操作不可撤销，相关事件通知将立即停止。</p>
      </div>
      <template #footer>
        <CypButton @click="showDeleteConfirmDialog = false"> 取消 </CypButton>
        <CypButton type="danger" @click="confirmDeleteWebhook">
          确认删除
        </CypButton>
      </template>
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
  </div>
</template>

<style lang="scss" scoped>
.webhook-page {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--text-primary, #1e293b);
  margin: 0 0 4px;
}

.page-subtitle {
  font-size: 14px;
  color: #64748b;
  margin: 0;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.stat-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;

  &.total {
    background: #e0e7ff;
    color: #6366f1;
  }
  &.active {
    background: #dcfce7;
    color: #22c55e;
  }
  &.triggers {
    background: #fef3c7;
    color: #f59e0b;
  }
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
}

.webhook-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 20px;
}

.webhook-card {
  :deep(.cyp-card__header) {
    padding-bottom: 0;
  }

  :deep(.cyp-card__body) {
    padding-top: 0;
  }
}

.webhook-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  width: 100%;
}

.webhook-info {
  flex: 1;
}

.webhook-name {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 4px;
}

.webhook-description {
  font-size: 13px;
  color: #64748b;
  margin: 0;
}

.webhook-content {
  .content-row {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 12px;

    &:last-child {
      margin-bottom: 0;
    }

    .label {
      color: #64748b;
      font-size: 13px;
      min-width: 60px;
    }

    code {
      background: #f1f5f9;
      padding: 2px 8px;
      border-radius: 4px;
      color: #1e293b;
      font-size: 13px;
    }
  }

  .event-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .stats {
    background: #f8fafc;
    padding: 12px;
    border-radius: 8px;
    display: flex;
    gap: 24px;
  }

  .mini-stat {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .mini-value {
    font-size: 14px;
    font-weight: 600;
    color: #1e293b;

    &.success {
      color: #22c55e;
    }
    &.danger {
      color: #ef4444;
    }
  }

  .mini-label {
    font-size: 12px;
    color: #64748b;
  }
}

.webhook-actions {
  display: flex;
  gap: 8px;
}

.confirm-content {
  p {
    margin: 0 0 8px;
    font-size: 14px;
    color: #374151;
  }

  .warning {
    color: #b91c1c;
  }
}

.empty-state {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 64px;
  text-align: center;
  background: white;
  border-radius: 12px;
  color: #64748b;

  svg {
    opacity: 0.5;
    margin-bottom: 16px;
  }

  h3 {
    font-size: 18px;
    color: #1e293b;
    margin: 0 0 8px;
  }

  p {
    margin: 0 0 16px;
  }
}

.dialog-form {
  .form-group {
    margin-bottom: 20px;

    &:last-child {
      margin-bottom: 0;
    }

    label {
      display: block;
      font-size: 14px;
      font-weight: 500;
      color: #374151;
      margin-bottom: 8px;
    }
  }
}

// 创建/编辑 Webhook 中的描述输入框背景强制为白色，避免在深色或灰色背景下可读性差
.dialog-form .textarea {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #1e293b;
  font-family: inherit;
  resize: vertical;
  background-color: #ffffff !important;

  &:focus {
    outline: none;
    border-color: #6366f1;
  }
}

.event-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}

.event-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
  background: #f8fafc;
  border: 2px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: #f1f5f9;
  }

  &.selected {
    background: #eef2ff;
    border-color: #6366f1;
  }

  input {
    display: none;
  }

  .event-icon {
    font-size: 24px;
    margin-bottom: 8px;
  }

  .event-label {
    font-size: 14px;
    font-weight: 500;
    color: #1e293b;
    margin-bottom: 4px;
  }

  .event-desc {
    font-size: 12px;
    color: #64748b;
    text-align: center;
  }
}

.test-info {
  background: #f8fafc;
  padding: 12px;
  border-radius: 8px;
  margin-bottom: 20px;

  p {
    margin: 4px 0;
    font-size: 13px;
    color: #374151;
  }
}

.test-result {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #dcfce7;
  border-radius: 8px;
  color: #22c55e;

  &.error {
    background: #fee2e2;
    color: #ef4444;
  }

  .result-icon {
    font-size: 24px;
  }
}

.detail-content {
  .detail-section {
    margin-bottom: 24px;

    &:last-child {
      margin-bottom: 0;
    }

    h4 {
      font-size: 14px;
      font-weight: 600;
      color: #1e293b;
      margin: 0 0 12px;
    }
  }

  .detail-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
  }

  .detail-item {
    &.full {
      grid-column: 1 / -1;
    }

    .detail-label {
      display: block;
      font-size: 12px;
      color: #64748b;
      margin-bottom: 4px;
    }

    .detail-value {
      font-size: 14px;
      color: #1e293b;
    }

    code {
      background: #f1f5f9;
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 13px;
    }
  }

  .stats-cards {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
  }

  .stats-card {
    background: #f8fafc;
    padding: 16px;
    border-radius: 8px;
    text-align: center;

    .stats-number {
      display: block;
      font-size: 24px;
      font-weight: 600;
      color: #1e293b;
    }

    .stats-label {
      font-size: 12px;
      color: #64748b;
    }

    &.success {
      background: #dcfce7;
    }
    &.danger {
      background: #fee2e2;
    }
  }
}

@media (max-width: 768px) {
  .webhook-list {
    grid-template-columns: 1fr;
  }

  .webhook-content .stats {
    flex-wrap: wrap;
  }
}
</style>
