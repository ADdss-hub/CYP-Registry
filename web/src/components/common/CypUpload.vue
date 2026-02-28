<script setup lang="ts">
import { ref, computed } from "vue";

interface Props {
  action?: "upload" | "download";
  accept?: string;
  multiple?: boolean;
  disabled?: boolean;
  maxSize?: number; // 单位: MB
  maxCount?: number;
}

const props = withDefaults(defineProps<Props>(), {
  action: "upload",
  accept: "*",
  multiple: false,
  disabled: false,
  maxSize: 10,
  maxCount: 10,
});

const emit = defineEmits<{
  select: [files: File[]];
  progress: [progress: number];
  success: [response: any];
  error: [error: Error];
}>();

const isDragging = ref(false);
const fileInput = ref<HTMLInputElement>();
const uploading = ref(false);
const progress = ref(0);

const uploadAreaClass = computed(() => [
  "cyp-upload",
  {
    "cyp-upload--dragging": isDragging,
    "cyp-upload--disabled": props.disabled,
  },
]);

function handleDragOver(e: DragEvent) {
  e.preventDefault();
  if (!props.disabled) {
    isDragging.value = true;
  }
}

function handleDragLeave(e: DragEvent) {
  e.preventDefault();
  isDragging.value = false;
}

function handleDrop(e: DragEvent) {
  e.preventDefault();
  isDragging.value = false;

  if (props.disabled) return;

  const files = Array.from(e.dataTransfer?.files || []);
  handleFiles(files);
}

function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  const files = Array.from(target.files || []);
  handleFiles(files);
  target.value = "";
}

function handleFiles(files: File[]) {
  // 验证文件
  const validFiles = files.filter((file) => {
    if (props.maxSize && file.size > props.maxSize * 1024 * 1024) {
      console.warn(`文件 ${file.name} 大小超过限制`);
      return false;
    }
    return true;
  });

  if (validFiles.length > 0) {
    emit("select", validFiles);
  }
}

function triggerSelect() {
  if (!props.disabled) {
    fileInput.value?.click();
  }
}

function simulateUpload() {
  // 模拟上传过程
  uploading.value = true;
  progress.value = 0;

  const interval = setInterval(() => {
    progress.value += 10;
    emit("progress", progress.value);

    if (progress.value >= 100) {
      clearInterval(interval);
      uploading.value = false;
    }
  }, 200);
}

defineExpose({
  triggerSelect,
  simulateUpload,
});
</script>

<template>
  <div :class="uploadAreaClass">
    <input
      ref="fileInput"
      type="file"
      class="cyp-upload__input"
      :accept="accept"
      :multiple="multiple"
      :disabled="disabled"
      @change="handleFileSelect"
    />

    <div
      class="cyp-upload__area"
      @click="triggerSelect"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
    >
      <div v-if="uploading" class="cyp-upload__uploading">
        <div class="cyp-upload__spinner" />
        <span class="cyp-upload__progress-text">上传中 {{ progress }}%</span>
        <div class="cyp-upload__progress-bar">
          <div
            class="cyp-upload__progress-fill"
            :style="{ width: `${progress}%` }"
          />
        </div>
      </div>

      <template v-else>
        <svg
          v-if="action === 'upload'"
          class="cyp-upload__icon"
          viewBox="0 0 24 24"
          width="48"
          height="48"
        >
          <path
            fill="currentColor"
            d="M9 16h6v-6h4l-7-7-7 7h4v6zm-4 2h14v2H5v-2z"
          />
        </svg>
        <svg
          v-else
          class="cyp-upload__icon"
          viewBox="0 0 24 24"
          width="48"
          height="48"
        >
          <path
            fill="currentColor"
            d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z"
          />
        </svg>
        <p class="cyp-upload__text">
          <span class="cyp-upload__link">点击或拖拽文件</span>
          到此区域
        </p>
        <p v-if="maxSize" class="cyp-upload__hint">
          单文件最大 {{ maxSize }}MB
        </p>
      </template>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.cyp-upload {
  &__input {
    display: none;
  }

  &__area {
    border: 2px dashed #e2e8f0;
    border-radius: 12px;
    padding: 48px;
    text-align: center;
    cursor: pointer;
    transition: all 0.2s ease;
    background: #fafafa;
  }

  &--dragging &__area {
    border-color: #6366f1;
    background: #eef2ff;
  }

  &--disabled &__area {
    opacity: 0.6;
    cursor: not-allowed;
  }

  &:not(&--disabled) &__area:hover {
    border-color: #6366f1;
    background: #f8fafc;
  }

  &__icon {
    color: #94a3b8;
    margin-bottom: 16px;
  }

  &__text {
    font-size: 14px;
    color: #64748b;
    margin: 0 0 8px;
  }

  &__link {
    color: #6366f1;
    font-weight: 500;
  }

  &__hint {
    font-size: 12px;
    color: #94a3b8;
    margin: 0;
  }

  &__uploading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }

  &__spinner {
    width: 32px;
    height: 32px;
    border: 3px solid #e2e8f0;
    border-top-color: #6366f1;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  &__progress-text {
    font-size: 14px;
    color: #64748b;
  }

  &__progress-bar {
    width: 200px;
    height: 6px;
    background: #e2e8f0;
    border-radius: 3px;
    overflow: hidden;
  }

  &__progress-fill {
    height: 100%;
    background: #6366f1;
    transition: width 0.2s ease;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
