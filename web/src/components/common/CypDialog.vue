<script setup lang="ts">
import { computed, onMounted, onUnmounted } from "vue";

interface Props {
  modelValue: boolean;
  title?: string;
  width?: string;
  closable?: boolean;
  maskClosable?: boolean;
  footer?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  title: "",
  width: "480px",
  closable: true,
  maskClosable: true,
  footer: true,
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  close: [];
}>();

const isVisible = computed(() => props.modelValue);

function handleClose() {
  emit("update:modelValue", false);
  emit("close");
}

function handleMaskClick() {
  if (props.maskClosable) {
    handleClose();
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === "Escape" && props.closable) {
    handleClose();
  }
}

onMounted(() => {
  document.addEventListener("keydown", handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener("keydown", handleKeydown);
});
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog-fade">
      <div v-if="isVisible" class="cyp-dialog__mask" @click="handleMaskClick">
        <div class="cyp-dialog" :style="{ width }" @click.stop>
          <div class="cyp-dialog__header">
            <h3 v-if="title" class="cyp-dialog__title">
              {{ title }}
            </h3>
            <button
              v-if="closable"
              class="cyp-dialog__close"
              @click="handleClose"
            >
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"
                />
              </svg>
            </button>
          </div>

          <div class="cyp-dialog__body">
            <slot />
          </div>

          <div v-if="footer" class="cyp-dialog__footer">
            <slot name="footer">
              <button
                class="cyp-dialog__btn cyp-dialog__btn--default"
                @click="handleClose"
              >
                取消
              </button>
              <button
                class="cyp-dialog__btn cyp-dialog__btn--primary"
                @click="handleClose"
              >
                确定
              </button>
            </slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style lang="scss" scoped>
.cyp-dialog {
  &__mask {
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
    padding: 20px;
  }

  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  max-height: 90vh;
  display: flex;
  flex-direction: column;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 24px;
    border-bottom: 1px solid #e2e8f0;
  }

  &__title {
    font-size: 18px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
  }

  &__close {
    background: none;
    border: none;
    color: #64748b;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover {
      background: #f1f5f9;
      color: #1e293b;
    }
  }

  &__body {
    padding: 24px;
    overflow-y: auto;
    flex: 1;
  }

  &__footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding: 16px 24px;
    border-top: 1px solid #e2e8f0;
    background: #f8fafc;
    border-radius: 0 0 12px 12px;
  }

  &__btn {
    padding: 10px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;

    &--default {
      background: white;
      border: 1px solid #e2e8f0;
      color: #1e293b;

      &:hover {
        background: #f8fafc;
        border-color: #cbd5e1;
      }
    }

    &--primary {
      background: #6366f1;
      border: 1px solid #6366f1;
      color: white;

      &:hover {
        background: #4f46e5;
        border-color: #4f46e5;
      }
    }
  }
}

// 动画
.dialog-fade-enter-active,
.dialog-fade-leave-active {
  transition: all 0.3s ease;
}

.dialog-fade-enter-from,
.dialog-fade-leave-to {
  opacity: 0;

  .cyp-dialog {
    transform: scale(0.95);
    opacity: 0;
  }
}
</style>
