<script setup lang="ts">
interface Props {
  title?: string;
  hoverable?: boolean;
}

defineProps<Props>();
</script>

<template>
  <div class="cyp-card" :class="{ 'cyp-card--hoverable': hoverable }">
    <div v-if="title || $slots.header" class="cyp-card__header">
      <slot name="header">
        <h3 class="cyp-card__title">
          {{ title }}
        </h3>
      </slot>
    </div>
    <div class="cyp-card__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="cyp-card__footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.cyp-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;

  &--hoverable {
    transition: all 0.2s ease;
    cursor: pointer;

    &:hover {
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
      transform: translateY(-2px);
    }
  }

  &__header {
    padding: 20px 24px;
    border-bottom: 1px solid #f1f5f9;
  }

  &__title {
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
    margin: 0;
  }

  &__body {
    padding: 24px;
  }

  &__footer {
    padding: 16px 24px;
    border-top: 1px solid #f1f5f9;
    background: #f8fafc;
  }
}
</style>
