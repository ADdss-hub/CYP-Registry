<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useProjectStore } from "@/stores/project";
import { projectApi } from "@/services/project";
import CypButton from "@/components/common/CypButton.vue";

const router = useRouter();
const projectStore = useProjectStore();

const statistics = ref<{
  total_projects: number;
  total_images: number;
  total_storage: number;
} | null>(null);

const statsCards = computed(() => {
  // 优先使用后端统计接口返回的准确数据，避免只统计当前页导致仪表盘与项目列表不一致
  const projectCount =
    statistics.value?.total_projects ??
    (projectStore.pagination?.total ??
      (projectStore.projects?.length ?? 0));

  const imageCount =
    statistics.value?.total_images ??
    (projectStore.projects || []).reduce(
      (sum, p: any) => sum + (p.imageCount || 0),
      0,
    );
  return [
    {
      title: "项目总数",
      value: projectCount,
      icon: "project",
      color: "#6366f1",
    },
    { title: "镜像总数", value: imageCount, icon: "image", color: "#22c55e" },
  ];
});

async function loadDashboardData() {
  await projectStore.fetchProjects({ pageSize: 5 });
  // 加载统计数据
  try {
    statistics.value = await projectApi.getStatistics();
  } catch (err) {
    console.error("Failed to load statistics:", err);
  }
}

onMounted(() => {
  loadDashboardData();
});

function formatBytes(bytes: number | undefined | null): string {
  if (!bytes || bytes === 0 || isNaN(bytes)) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

function navigateToProjects() {
  router.push("/projects");
}
</script>

<template>
  <div class="dashboard-page">
    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div v-for="stat in statsCards" :key="stat.title" class="stat-card">
        <div class="stat-icon" :style="{ background: stat.color }">
          <svg viewBox="0 0 24 24" width="24" height="24">
            <rect fill="white" width="24" height="24" rx="4" />
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">
            {{ stat.value }}
          </div>
          <div class="stat-title">
            {{ stat.title }}
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区 -->
    <div class="dashboard-content">
      <!-- 最近项目 -->
      <div class="dashboard-card">
        <div class="card-header">
          <h2>最近项目</h2>
          <CypButton size="small" @click="navigateToProjects">
            查看全部
          </CypButton>
        </div>
        <div class="project-list">
          <div
            v-for="project in projectStore.projects.slice(0, 5)"
            :key="project.id"
            class="project-item"
            @click="router.push(`/projects/${project.id}`)"
          >
            <div class="project-info">
              <div class="project-name">
                {{ project.name }}
              </div>
              <div class="project-meta">
                <span>{{ project.imageCount }} 个镜像</span>
                <span class="separator">•</span>
                <span>{{ project.isPublic ? "公开" : "私有" }}</span>
              </div>
            </div>
            <div class="project-storage">
              {{ formatBytes(project.storageUsed ?? 0) }}
            </div>
          </div>
          <div v-if="projectStore.projects.length === 0" class="empty-state">
            <svg viewBox="0 0 24 24" width="48" height="48">
              <path
                fill="currentColor"
                d="M20 6h-8l-2-2H4C2.9 4 2 4.9 2 6v10c0 1.1.9 2 2 2h5v-2H4V8h16v8h-3v2h3c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2z"
              />
            </svg>
            <p>暂无项目</p>
            <CypButton type="primary" size="small" @click="navigateToProjects">
              创建项目
            </CypButton>
          </div>
        </div>
      </div>
    </div>

    <!-- 快捷操作 -->
    <div class="quick-actions">
      <h2>快捷操作</h2>
      <div class="actions-grid">
        <div class="action-card" @click="navigateToProjects">
          <div class="action-icon">
            <svg viewBox="0 0 24 24" width="32" height="32">
              <path
                fill="currentColor"
                d="M20 6h-8l-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"
              />
            </svg>
          </div>
          <div class="action-title">管理项目</div>
          <div class="action-desc">创建和管理镜像项目</div>
        </div>

        <div class="action-card">
          <div class="action-icon" style="background: #22c55e">
            <svg viewBox="0 0 24 24" width="32" height="32">
              <path
                fill="currentColor"
                d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"
              />
            </svg>
          </div>
          <div class="action-title">推送镜像</div>
          <div class="action-desc">使用Docker CLI推送</div>
        </div>

        <div class="action-card">
          <div class="action-icon" style="background: #3b82f6">
            <svg viewBox="0 0 24 24" width="32" height="32">
              <path
                fill="currentColor"
                d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM14 13v4h-4v-4H7l5-5 5 5h-3z"
              />
            </svg>
          </div>
          <div class="action-title">拉取镜像</div>
          <div class="action-desc">从仓库拉取镜像</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.dashboard-page {
  max-width: 1400px;
  margin: 0 auto;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1e293b;
}

.stat-title {
  font-size: 14px;
  color: #64748b;
}

.dashboard-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.dashboard-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.card-header {
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

.project-list {
  display: flex;
  flex-direction: column;
}

.project-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid #f1f5f9;
  cursor: pointer;
  transition: background 0.2s ease;

  &:hover {
    background: #f8fafc;
    margin: 0 -20px;
    padding: 12px 20px;
  }

  &:last-child {
    border-bottom: none;
  }
}

.project-name {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
}

.project-meta {
  font-size: 12px;
  color: #64748b;
  margin-top: 4px;

  .separator {
    margin: 0 8px;
  }
}

.project-storage {
  font-size: 13px;
  color: #64748b;
}

.activity-list {
  display: flex;
  flex-direction: column;
}

.activity-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f1f5f9;

  &:last-child {
    border-bottom: none;
  }
}

.activity-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;

  &.push {
    background: #dcfce7;
    color: #22c55e;
  }

  &.scan {
    background: #fef3c7;
    color: #f59e0b;
  }

  &.delete {
    background: #fee2e2;
    color: #ef4444;
  }
}

.activity-text {
  font-size: 14px;
  color: #64748b;
  line-height: 1.5;

  strong {
    color: #1e293b;
  }
}

.activity-time {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
}

.empty-state {
  text-align: center;
  padding: 24px;
  color: #64748b;

  svg {
    opacity: 0.5;
    margin-bottom: 8px;
  }
}

.quick-actions {
  h2 {
    font-size: 16px;
    font-weight: 600;
    color: var(--text-primary, #1e293b);
    margin: 0 0 16px;
  }
}

.actions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.action-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 24px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }
}

.action-icon {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  background: #6366f1;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  color: white;
}

.action-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 4px;
}

.action-desc {
  font-size: 13px;
  color: #64748b;
}
</style>
