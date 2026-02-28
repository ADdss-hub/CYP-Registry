<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { useProjectStore } from '@/stores/project'
import { useNotificationStore } from '@/stores/notification'
import CypButton from '@/components/common/CypButton.vue'
import CypInput from '@/components/common/CypInput.vue'
import CypTable from '@/components/common/CypTable.vue'
import CypDialog from '@/components/common/CypDialog.vue'
import type { Project } from '@/types'
import { copyToClipboard } from '@/utils/clipboard'

const router = useRouter()
const projectStore = useProjectStore()
const notificationStore = useNotificationStore()

const searchKeyword = ref('')
const showCreateModal = ref(false)
const isCreating = ref(false) // 防止重复提交
const newProject = ref({
  name: '',
  description: '',
  isPublic: false,
  storageQuota: 10 * 1024 * 1024 * 1024, // 默认10GB
})

// 删除项目确认对话框（替代浏览器 confirm，遵循界面规范3.3/3.4节）
const showDeleteConfirmDialog = ref(false)
const projectToDelete = ref<Project | null>(null)

// 项目分享对话框
const showShareDialog = ref(false)
const projectToShare = ref<Project | null>(null)

// 通用提示框（创建项目校验/错误信息）
const showMessageDialog = ref(false)
const messageDialogTitle = ref('')
const messageDialogContent = ref('')

function openMessageDialog(title: string, content: string) {
  messageDialogTitle.value = title
  messageDialogContent.value = content
  showMessageDialog.value = true
}

// 构造项目分享内容（包含本系统 Web 访问地址 + Docker 命令）
const shareContent = computed(() => {
  if (!projectToShare.value) return ''
  const host = window.location.host || 'localhost:8080'
  // 项目名称即仓库命名空间，遵循后端约定（仅字母和数字）
  const repo = projectToShare.value.name || projectToShare.value.id
  const lines: string[] = [
    '# 一、项目访问信息（本系统）',
    `项目名称：${projectToShare.value.name}`,
    `项目 ID：${projectToShare.value.id}`,
    '',
    '# 1. Web 控制台项目列表',
    `http://${host}/projects`,
    '',
    '# 2. 推荐项目访问链接（如文档中使用，可二选一）',
    `http://${host}/projects/${projectToShare.value.id}`,
    `http://${host}/projects?keyword=${encodeURIComponent(projectToShare.value.name || '')}`,
    '',
    '========================================',
    '',
    '# 二、Docker 仓库与命令示例',
    '',
    '# 1. Docker 登录（如已登录可跳过）',
    `docker login ${host}`,
    '',
    '# 2. 推送镜像到该项目',
    `docker tag your-image:tag ${host}/${repo}:your-tag`,
    `docker push ${host}/${repo}:your-tag`,
    '',
    '# 3. 从该项目拉取镜像',
    `docker pull ${host}/${repo}:your-tag`,
  ]
  return lines.join('\n')
})

const columns = [
  { key: 'name', title: '项目名称' },
  { key: 'description', title: '描述' },
  {
    key: 'isPublic',
    title: '可见性',
    customRender: (value: boolean) => value ? '公开' : '私有',
  },
  {
    key: 'storageUsed',
    title: '已用存储',
    customRender: (value: number | undefined | null) => {
      if (value === undefined || value === null || isNaN(value)) {
        return '0 B'
      }
      return formatBytes(value)
    },
  },
  {
    key: 'imageCount',
    title: '镜像数量',
    customRender: (value: number | undefined | null) => {
      if (value === undefined || value === null || isNaN(value)) {
        return '0'
      }
      return String(value)
    },
  },
  {
    key: 'createdAt',
    title: '创建时间',
    customRender: (value: string | undefined | null) => {
      if (!value) {
        return '-'
      }
      return formatDate(value)
    },
  },
  {
    key: 'actions',
    title: '操作',
    align: 'right' as const,
    customRender: (_: any, record: Project) => {
      return h('div', { class: 'action-buttons' }, [
        // 注意：表格行本身绑定了 rowClick，会触发跳转；这里必须 stopPropagation，否则“查看/删除”会被行点击吞掉
        h(
          CypButton,
          {
            size: 'small',
            onClick: (e: MouseEvent) => {
              e.stopPropagation()
              navigateToProject(record)
            },
          },
          { default: () => '查看' }
        ),
        h(
          CypButton,
          {
            size: 'small',
            type: 'default',
            style: { marginLeft: '8px' },
            onClick: (e: MouseEvent) => {
              e.stopPropagation()
              openShare(record)
            },
          },
          { default: () => '分享' }
        ),
        h(
          CypButton,
          {
            size: 'small',
            type: 'danger',
            style: { marginLeft: '8px' },
            onClick: (e: MouseEvent) => {
              e.stopPropagation()
              handleDelete(record)
            },
          },
          { default: () => '删除' }
        ),
      ])
    },
  },
]

const isLoading = computed(() => projectStore.isLoading)
const projects = computed(() => projectStore.projects)
const pagination = computed(() => projectStore.pagination)

const totalImages = computed(() =>
  projects.value.reduce((sum, p) => sum + (p.imageCount || 0), 0),
)

const totalStorage = computed(() =>
  projects.value.reduce((sum, p) => sum + (p.storageUsed || 0), 0),
)

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('zh-CN')
}

function navigateToProject(project: Project) {
  router.push(`/projects/${project.id}`)
}

async function handleDelete(project: Project) {
  projectToDelete.value = project
  showDeleteConfirmDialog.value = true
}

function openShare(project: Project) {
  projectToShare.value = project
  showShareDialog.value = true
}

async function handleCopyShare() {
  if (!shareContent.value) return
  try {
    await copyToClipboard(shareContent.value)
    openMessageDialog('复制成功', '项目分享信息已复制到剪贴板，可直接粘贴发送给其他人。')
    if (projectToShare.value) {
      notificationStore.addNotification({
        source: 'project',
        title: '项目分享信息已生成',
        message: `已为项目「${projectToShare.value.name}」生成并复制 Docker 登录/推送/拉取命令。`,
        status: 'success',
      })
    }
  } catch (err: any) {
    openMessageDialog('复制失败', err?.message || '无法访问剪贴板，请手动复制。')
  }
}

async function handleSearch() {
  await projectStore.fetchProjects({ keyword: searchKeyword.value })
}

async function handlePageChange(page: number, pageSize: number) {
  await projectStore.fetchProjects({ page, pageSize, keyword: searchKeyword.value })
}

async function handleCreateProject() {
  // 防止重复提交
  if (isCreating.value) {
    return
  }

  const payload = {
    ...newProject.value,
    name: newProject.value.name.trim(),
    description: newProject.value.description.trim(),
  }

  if (!payload.name) {
    openMessageDialog('校验失败', '请输入项目名称')
    return
  }
  // 与后端约定保持一致：项目名称即 Registry 仓库名，可包含命名空间
  // 允许的字符：字母、数字、减号(-)、下划线(_)、斜杠(/)、点(.)
  // 说明：例如 "test-project/test-small"、"team1/app.backend" 等均合法
  if (!/^[A-Za-z0-9._/-]{3,128}$/.test(payload.name)) {
  openMessageDialog('校验失败', '项目名称仅支持 3-128 位字母、数字、-、_、/ 或 .')
    return
  }
  
  isCreating.value = true
  try {
    const project = await projectStore.createProject(payload)
    showCreateModal.value = false
    newProject.value = {
      name: '',
      description: '',
      isPublic: false,
      storageQuota: 10 * 1024 * 1024 * 1024,
    }
    // 创建成功后刷新项目列表，确保列表与后端状态一致
    await projectStore.fetchProjects()
    notificationStore.addNotification({
      source: 'project',
      title: '项目已创建',
      message: `项目「${project?.name || payload.name}」已创建`,
      status: 'success',
    })
  } catch (err: any) {
    // 检查是否是项目已存在的错误（code 20002 或消息包含"已存在"）
    const errorCode = err?.code || err?.response?.data?.code
    const errorMessage = err?.message || err?.response?.data?.message || '创建项目失败，请稍后重试'
    
    // 如果是项目已存在的错误，显示更友好的提示
    if (errorCode === 20002 || errorMessage.includes('已存在') || errorMessage.includes('already exists')) {
      openMessageDialog('项目已存在', errorMessage || `项目 "${payload.name}" 已存在，请使用其他名称`)
    } else {
      openMessageDialog('创建失败', errorMessage)
    }
  } finally {
    isCreating.value = false
  }
}

onMounted(() => {
  projectStore.fetchProjects()
})
</script>

<template>
  <div class="projects-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">项目管理</h2>
        <p class="page-subtitle">管理您的容器镜像项目</p>
      </div>
      <CypButton type="primary" @click="showCreateModal = true">
        创建项目
      </CypButton>
    </div>

    <!-- 全局统计概览 -->
    <div class="stats-bar">
      <div class="stat-item">
        <span class="stat-label">项目总数</span>
        <span class="stat-value">{{ pagination.total }}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">镜像总数</span>
        <span class="stat-value">{{ totalImages }}</span>
      </div>
      <div class="stat-item">
        <span class="stat-label">总存储用量</span>
        <span class="stat-value">{{ formatBytes(totalStorage) }}</span>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <CypInput
        v-model="searchKeyword"
        placeholder="搜索项目名称..."
        @keyup.enter="handleSearch"
      />
      <CypButton type="primary" @click="handleSearch">
        搜索
      </CypButton>
      <CypButton type="default" :loading="isLoading" @click="() => projectStore.fetchProjects()">
        刷新列表
      </CypButton>
    </div>

    <!-- 项目列表 -->
    <CypTable
      :columns="columns"
      :data="projects"
      :loading="isLoading"
      :pagination="{
        page: pagination.page,
        pageSize: pagination.pageSize,
        total: pagination.total,
        onChange: handlePageChange,
      }"
      @rowClick="navigateToProject"
    >
      <template #empty>
        <div class="empty-state">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path fill="currentColor" d="M20 6h-8l-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
          </svg>
          <h3>暂无项目</h3>
          <p>创建您的第一个项目开始使用镜像仓库</p>
          <CypButton type="primary" @click="showCreateModal = true">
            创建项目
          </CypButton>
        </div>
      </template>
    </CypTable>

    <!-- 项目分享对话框 -->
    <CypDialog
      v-model="showShareDialog"
      :title="projectToShare ? `分享项目：${projectToShare.name}` : '分享项目'"
    >
      <p class="share-description">
        将下方信息复制给使用者：包含本系统的 Web 控制台项目访问地址，以及对应项目的 Docker 登录 / 推送 /
        拉取命令示例，便于统一粘贴到文档或 IM 中使用。
      </p>
      <textarea
        class="share-textarea"
        :value="shareContent"
        readonly
      ></textarea>
      <template #footer>
        <CypButton type="default" @click="showShareDialog = false">
          关闭
        </CypButton>
        <CypButton type="primary" style="margin-left: 8px" @click="handleCopyShare">
          复制分享信息
        </CypButton>
      </template>
    </CypDialog>

    <!-- 创建项目弹窗（历史实现，整体样式已基本符合系统框规范） -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h2>创建项目</h2>
          <button class="close-btn" @click="showCreateModal = false">
            <svg viewBox="0 0 24 24" width="24" height="24">
              <path fill="currentColor" d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
            </svg>
          </button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label class="form-label">项目名称 *</label>
            <CypInput
              v-model="newProject.name"
              placeholder="请输入项目名称"
            />
          </div>
          <div class="form-group">
            <label class="form-label">描述</label>
            <textarea
              v-model="newProject.description"
              class="textarea"
              placeholder="请输入项目描述"
              rows="3"
            />
          </div>
          <div class="form-group">
            <label class="form-label">
              <input
                v-model="newProject.isPublic"
                type="checkbox"
              />
              公开项目（允许匿名拉取；公开/私有项目均支持 Docker CLI 与自动化工具，差别仅在是否允许匿名拉取）
            </label>
          </div>
        </div>
        <div class="modal-footer">
          <CypButton @click="showCreateModal = false" :disabled="isCreating">取消</CypButton>
          <CypButton type="primary" @click="handleCreateProject" :loading="isCreating">创建</CypButton>
        </div>
      </div>
    </div>

    <!-- 删除项目确认对话框（系统框 + 确认提示框规范） -->
    <CypDialog
      v-model="showDeleteConfirmDialog"
      title="删除项目"
      width="420px"
      @close="showDeleteConfirmDialog = false"
    >
      <div v-if="projectToDelete" class="confirm-content">
        <p>确定要删除项目 "<strong>{{ projectToDelete.name }}</strong>" 吗？</p>
        <p class="warning">此操作无法撤销，项目下的所有镜像和配置将被永久移除。</p>
      </div>
      <template #footer>
        <CypButton @click="showDeleteConfirmDialog = false">取消</CypButton>
        <CypButton
          type="danger"
          @click="
            async () => {
              if (!projectToDelete) return
              await projectStore.deleteProject(projectToDelete.id)
              showDeleteConfirmDialog = false
              projectToDelete = null
              notificationStore.addNotification({
                source: 'project',
                title: '项目已删除',
                message: '选中的项目及其镜像已被删除',
                status: 'success',
              })
            }
          "
        >
          确认删除
        </CypButton>
      </template>
    </CypDialog>

    <!-- 通用提示框（表单校验/错误提示） -->
    <CypDialog
      v-model="showMessageDialog"
      :title="messageDialogTitle"
      width="360px"
      @close="showMessageDialog = false"
    >
      <p>{{ messageDialogContent }}</p>
      <template #footer>
        <CypButton type="primary" @click="showMessageDialog = false">知道了</CypButton>
      </template>
    </CypDialog>
  </div>
</template>

<style lang="scss" scoped>
.projects-page {
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: center;
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

.stats-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;

  .stat-item {
    padding: 8px 12px;
    border-radius: 8px;
    background: #f8fafc;
    font-size: 13px;
    color: #64748b;

    .stat-label {
      margin-right: 8px;
    }

    .stat-value {
      font-weight: 600;
      color: #1e293b;
    }
  }
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;

  .cyp-input-wrapper {
    flex: 1;
    max-width: 400px;
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 48px;
  color: #64748b;

  svg {
    opacity: 0.5;
  }

  h3 {
    font-size: 18px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
  }

  p {
    font-size: 14px;
    margin: 0;
  }
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  width: 100%;
  max-width: 480px;
  background: white;
  border-radius: 12px;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid #e2e8f0;

  h2 {
    font-size: 18px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
  }

  .close-btn {
    background: none;
    border: none;
    color: #64748b;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    transition: all 0.2s ease;

    &:hover {
      background: #f1f5f9;
      color: #1e293b;
    }
  }
}

.modal-body {
  padding: 24px;
}

.form-group {
  margin-bottom: 20px;

  &:last-child {
    margin-bottom: 0;
  }
}

.form-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  margin-bottom: 8px;
  cursor: pointer;
}

.textarea {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #1e293b;
  resize: vertical;
  font-family: inherit;

  &:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
  }

  &::placeholder {
    color: #94a3b8;
  }
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid #e2e8f0;
  background: #f8fafc;
}

.action-buttons {
  display: flex;
  align-items: center;
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

.share-description {
  margin-bottom: 12px;
  font-size: 14px;
  color: #64748b;
}

.share-textarea {
  width: 100%;
  min-height: 180px;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  background: #0f172a0d;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #0f172a;
  resize: vertical;

  &:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
  }
}
</style>

