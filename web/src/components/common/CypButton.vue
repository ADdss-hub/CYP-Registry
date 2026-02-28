<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  type?: 'default' | 'primary' | 'success' | 'warning' | 'danger'
  size?: 'small' | 'medium' | 'large'
  disabled?: boolean
  loading?: boolean
  block?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
  size: 'medium',
  disabled: false,
  loading: false,
  block: false,
})

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

const buttonClass = computed(() => [
  'cyp-button',
  `cyp-button--${props.type}`,
  `cyp-button--${props.size}`,
  {
    'cyp-button--disabled': props.disabled || props.loading,
    'cyp-button--block': props.block,
  },
])

function handleClick(event: MouseEvent) {
  if (!props.disabled && !props.loading) {
    emit('click', event)
  }
}
</script>

<template>
  <button
    :class="buttonClass"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <span v-if="loading" class="cyp-button__loading">
      <svg class="spinner" viewBox="0 0 24 24">
        <circle
          class="path"
          cx="12"
          cy="12"
          r="10"
          fill="none"
          stroke-width="4"
        />
      </svg>
    </span>
    <slot />
  </button>
</template>

<style lang="scss" scoped>
@use '@/assets/styles/variables.scss' as *;

.cyp-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  &--small {
    padding: 6px 12px;
    font-size: 12px;
  }

  &--medium {
    padding: 10px 20px;
    font-size: 14px;
  }

  &--large {
    padding: 14px 28px;
    font-size: 16px;
  }

  &--block {
    display: flex;
    width: 100%;
  }

  &--default {
    background: $light-bg-tertiary;
    border-color: $light-border;
    color: $light-text;

    &:hover:not(:disabled) {
      background: $light-border;
    }
  }

  &--primary {
    background: $primary-color;
    border-color: $primary-color;
    color: white;

    &:hover:not(:disabled) {
      background: $primary-hover;
      border-color: $primary-hover;
    }
  }

  &--success {
    background: $success-color;
    border-color: $success-color;
    color: white;

    &:hover:not(:disabled) {
      background: darken($success-color, 10%);
    }
  }

  &--warning {
    background: $warning-color;
    border-color: $warning-color;
    color: white;

    &:hover:not(:disabled) {
      background: darken($warning-color, 10%);
    }
  }

  &--danger {
    background: $danger-color;
    border-color: $danger-color;
    color: white;

    &:hover:not(:disabled) {
      background: darken($danger-color, 10%);
    }
  }

  &__loading {
    position: absolute;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .spinner {
    width: 16px;
    height: 16px;
    animation: rotate 2s linear infinite;
  }

  .path {
    stroke: currentColor;
    stroke-linecap: round;
    animation: dash 1.5s ease-in-out infinite;
  }
}

@keyframes rotate {
  100% {
    transform: rotate(360deg);
  }
}

@keyframes dash {
  0% {
    stroke-dasharray: 1, 150;
    stroke-dashoffset: 0;
  }
  50% {
    stroke-dasharray: 90, 150;
    stroke-dashoffset: -35;
  }
  100% {
    stroke-dasharray: 90, 150;
    stroke-dashoffset: -124;
  }
}
</style>

