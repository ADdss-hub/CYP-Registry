<script setup lang="ts">
interface Props {
  fullscreen?: boolean;
  text?: string;
}

withDefaults(defineProps<Props>(), {
  fullscreen: false,
  text: "加载中...",
});
</script>

<template>
  <div :class="['cyp-loading', { 'cyp-loading--fullscreen': fullscreen }]">
    <div class="cyp-loading__spinner">
      <svg viewBox="0 0 50 50">
        <circle class="cyp-loading__track" cx="25" cy="25" r="20" />
        <circle class="cyp-loading__progress" cx="25" cy="25" r="20" />
      </svg>
    </div>
    <p class="cyp-loading__text">
      {{ text }}
    </p>
  </div>
</template>

<style scoped lang="scss">
@use "@/assets/styles/variables.scss" as *;

.cyp-loading {
  display: inline-flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;

  &--fullscreen {
    position: fixed;
    inset: 0;
    background: rgba(255, 255, 255, 0.8);
    z-index: 1200;
  }

  &__spinner {
    width: 48px;
    height: 48px;
  }

  svg {
    width: 100%;
    height: 100%;
  }

  &__track {
    fill: none;
    stroke: #e0e0e0;
    stroke-width: 4;
  }

  &__progress {
    fill: none;
    stroke-width: 4;
    stroke-linecap: round;
    stroke: $primary-color;
    stroke-dasharray: 126;
    stroke-dashoffset: 100;
    transform-origin: 50% 50%;
    animation: cyp-loading-rotate 1.5s linear infinite;
  }

  &__text {
    font-size: 14px;
    color: $light-text-secondary;
  }
}

@keyframes cyp-loading-rotate {
  0% {
    transform: rotate(0deg);
    stroke-dashoffset: 100;
  }
  100% {
    transform: rotate(360deg);
    stroke-dashoffset: -26;
  }
}
</style>
