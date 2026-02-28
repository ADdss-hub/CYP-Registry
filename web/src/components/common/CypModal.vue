<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'

interface Props {
  modelValue: boolean
  title?: string
  width?: string
  fullscreen?: boolean
  closable?: boolean
  maskClosable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  width: '600px',
  fullscreen: false,
  closable: true,
  maskClosable: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

const isVisible = computed(() => props.modelValue)

function handleClose() {
  emit('update:modelValue', false)
  emit('close')
}

function handleMaskClick() {
  if (props.maskClosable) {
    handleClose()
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.closable) {
    handleClose()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="isVisible" class="cyp-modal__mask" @click="handleMaskClick">
        <div
          class="cyp-modal"
          :class="{ 'cyp-modal--fullscreen': fullscreen }"
          :style="{ width }"
          @click.stop
        >
          <div class="cyp-modal__header">
            <h3 v-if="title" class="cyp-modal__title">{{ title }}</h3>
            <div class="cyp-modal__header-right">
              <button
                v-if="fullscreen"
                class="cyp-modal__fullscreen-btn"
                @click="$emit('update:modelValue', false)"
              >
                <svg viewBox="0 0 24 24" width="18" height="18">
                  <path
                    fill="currentColor"
                    d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"
                  />
                </svg>
              </button>
              <button
                v-if="closable"
                class="cyp-modal__close"
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
          </div>

          <div class="cyp-modal__body">
            <slot />
          </div>

          <div v-if="$slots.footer" class="cyp-modal__footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style lang="scss" scoped>
.cyp-modal {
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
    z-index: 1100;
    padding: 20px;
  }

  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2);
  max-height: calc(100vh - 40px);
  display: flex;
  flex-direction: column;

  &--fullscreen {
    width: 100% !important;
    max-width: 100%;
    height: 100%;
    max-height: 100%;
    border-radius: 0;
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 24px;
    border-bottom: 1px solid #e2e8f0;
  }

  &__header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  &__title {
    font-size: 18px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
  }

  &__close,
  &__fullscreen-btn {
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
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: all 0.3s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;

  .cyp-modal {
    transform: scale(0.95);
    opacity: 0;
  }
}
</style>

