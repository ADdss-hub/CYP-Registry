<script setup lang="ts">
import { onMounted, ref } from "vue";
import type {
  AuditLog,
  AuditLogListResponse,
  AuditLogQueryParams,
} from "@/services/admin";
import { adminApi } from "@/services/admin";

const loading = ref(false);
const logs = ref<AuditLog[]>([]);

const page = ref(1);
const pageSize = ref(20);
const total = ref(0);

const query = ref<AuditLogQueryParams>({});

async function loadLogs() {
  loading.value = true;
  try {
    const res: AuditLogListResponse = await adminApi.getAuditLogs({
      page: page.value,
      page_size: pageSize.value,
      ...query.value,
    });
    logs.value = res.logs;
    total.value = res.total;
  } finally {
    loading.value = false;
  }
}

function formatTime(value: string) {
  if (!value) return "-";
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return value;
  return d.toLocaleString();
}

onMounted(() => {
  loadLogs();
});
</script>

<template>
  <div class="admin-logs-page">
    <div class="header">
      <h1>系统审计日志</h1>
      <p class="subtitle">
        展示近期的关键操作记录（登录、权限变更、镜像操作等），便于排查问题与安全审计。
      </p>
    </div>

    <div class="card">
      <div class="card-header">
        <div class="card-title">日志列表</div>
        <div class="card-actions">
          <button
            class="btn"
            type="button"
            @click="loadLogs"
            :disabled="loading"
          >
            刷新
          </button>
        </div>
      </div>

      <div v-if="loading" class="logs-loading">正在加载日志...</div>
      <div v-else-if="logs.length === 0" class="logs-empty">
        暂无审计日志记录。
      </div>
      <div v-else class="logs-table-wrapper">
        <table class="logs-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>用户</th>
              <th>操作</th>
              <th>资源</th>
              <th>IP</th>
              <th>详情</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in logs" :key="item.id">
              <td>{{ formatTime(item.created_at) }}</td>
              <td>{{ item.user_id || "-" }}</td>
              <td>{{ item.action }}</td>
              <td>{{ item.resource }}</td>
              <td>{{ item.ip }}</td>
              <td class="details-cell">
                <pre>{{ item.details }}</pre>
              </td>
            </tr>
          </tbody>
        </table>

        <div class="pagination">
          <span>共 {{ total }} 条</span>
          <button
            type="button"
            class="btn"
            :disabled="page <= 1 || loading"
            @click="
              page--;
              loadLogs();
            "
          >
            上一页
          </button>
          <button
            type="button"
            class="btn"
            :disabled="logs.length < pageSize || loading"
            @click="
              page++;
              loadLogs();
            "
          >
            下一页
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-logs-page {
  padding: 24px;
}

.header {
  margin-bottom: 16px;
}

.header h1 {
  margin: 0 0 4px;
  font-size: 20px;
}

.subtitle {
  margin: 0;
  color: #64748b;
  font-size: 13px;
}

.card {
  background: #0f172a;
  border-radius: 12px;
  padding: 16px;
  border: 1px solid #1e293b;
  box-shadow: 0 18px 45px rgba(15, 23, 42, 0.65);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.card-title {
  font-weight: 600;
}

.card-actions .btn {
  padding: 6px 12px;
  border-radius: 999px;
  border: none;
  background: #3b82f6;
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}

.card-actions .btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.logs-loading,
.logs-empty {
  padding: 24px 0;
  text-align: center;
  color: #94a3b8;
}

.logs-table-wrapper {
  overflow-x: auto;
}

.logs-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.logs-table th,
.logs-table td {
  padding: 8px 10px;
  border-bottom: 1px solid #1e293b;
  text-align: left;
}

.logs-table th {
  color: #94a3b8;
  font-weight: 500;
}

.logs-table tbody tr:hover {
  background: rgba(148, 163, 184, 0.08);
}

.details-cell pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
}

.pagination {
  margin-top: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #94a3b8;
}

.pagination .btn {
  padding: 4px 10px;
  border-radius: 999px;
  border: none;
  background: #1d4ed8;
  color: #fff;
  font-size: 12px;
  cursor: pointer;
}

.pagination .btn:disabled {
  opacity: 0.5;
  cursor: default;
}
</style>
