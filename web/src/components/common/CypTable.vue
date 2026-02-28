<script setup lang="ts">

interface Column {
  key: string
  title: string
  width?: string
  align?: 'left' | 'center' | 'right'
  customRender?: (value: any, record: any) => any
}

interface Props {
  columns: Column[]
  data: any[]
  loading?: boolean
  rowKey?: string
  pagination?: {
    page: number
    pageSize: number
    total: number
    onChange: (page: number, pageSize: number) => void
  }
}

withDefaults(defineProps<Props>(), {
  loading: false,
  rowKey: 'id',
})

const emit = defineEmits<{
  rowClick: [record: any]
}>()

function getCellStyle(column: Column) {
  return {
    textAlign: column.align || 'left',
    width: column.width || 'auto',
  }
}

function handleRowClick(record: any) {
  emit('rowClick', record)
}
</script>

<template>
  <div class="cyp-table">
    <div class="cyp-table__container">
      <table>
        <thead>
          <tr>
            <th
              v-for="column in columns"
              :key="column.key"
              :style="getCellStyle(column)"
            >
              {{ column.title }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="record in data"
            :key="record[rowKey]"
            @click="handleRowClick(record)"
          >
            <td
              v-for="column in columns"
              :key="column.key"
              :style="getCellStyle(column)"
            >
              <template v-if="column.customRender">
                <template v-if="typeof column.customRender(record[column.key], record) === 'string'">
                  {{ column.customRender(record[column.key], record) }}
                </template>
                <component v-else :is="column.customRender(record[column.key], record)" />
              </template>
              <template v-else>
                {{ record[column.key] ?? '-' }}
              </template>
            </td>
          </tr>
          <tr v-if="!loading && data.length === 0">
            <td :colspan="columns.length" class="cyp-table__empty">
              <slot name="empty">
                <div class="cyp-table__empty-content">
                  <svg class="empty-icon" viewBox="0 0 24 24" width="48" height="48">
                    <path fill="currentColor" d="M20 6h-8l-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"/>
                  </svg>
                  <p>暂无数据</p>
                </div>
              </slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="cyp-table__loading">
      <div class="cyp-table__loading-spinner" />
    </div>

    <!-- 分页 -->
    <div v-if="pagination" class="cyp-table__pagination">
      <span class="cyp-table__pagination-info">
        共 {{ pagination.total }} 条
      </span>
      <div class="cyp-table__pagination-buttons">
        <button
          class="cyp-table__pagination-btn"
          :disabled="pagination.page === 1"
          @click="pagination.onChange(pagination.page - 1, pagination.pageSize)"
        >
          上一页
        </button>
        <span class="cyp-table__pagination-page">
          {{ pagination.page }}
        </span>
        <button
          class="cyp-table__pagination-btn"
          :disabled="pagination.page * pagination.pageSize >= pagination.total"
          @click="pagination.onChange(pagination.page + 1, pagination.pageSize)"
        >
          下一页
        </button>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.cyp-table {
  position: relative;
  background: white;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);

  &__container {
    overflow-x: auto;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th,
  td {
    padding: 14px 16px;
    text-align: left;
    border-bottom: 1px solid #f1f5f9;
  }

  th {
    background: #f8fafc;
    font-weight: 600;
    color: #64748b;
    font-size: 13px;
    white-space: nowrap;
  }

  tbody tr {
    transition: background 0.15s ease;
    cursor: pointer;

    &:hover {
      background: #f8fafc;
    }
  }

  td {
    color: #1e293b;
    font-size: 14px;
  }

  &__empty {
    text-align: center;
    padding: 60px 0 !important;
  }

  &__empty-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: #94a3b8;

    .empty-icon {
      opacity: 0.5;
    }
  }

  &__loading {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(255, 255, 255, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  &__loading-spinner {
    width: 32px;
    height: 32px;
    border: 3px solid #e2e8f0;
    border-top-color: #6366f1;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  &__pagination {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
    border-top: 1px solid #f1f5f9;
  }

  &__pagination-info {
    color: #64748b;
    font-size: 13px;
  }

  &__pagination-buttons {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__pagination-btn {
    padding: 6px 12px;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    background: white;
    color: #64748b;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s ease;

    &:hover:not(:disabled) {
      border-color: #6366f1;
      color: #6366f1;
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  &__pagination-page {
    padding: 6px 12px;
    background: #6366f1;
    color: white;
    border-radius: 6px;
    font-size: 13px;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>

