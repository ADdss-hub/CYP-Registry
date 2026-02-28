<script setup lang="ts">

interface Tab {
  key: string
  title: string
  disabled?: boolean
}

interface Props {
  modelValue: string
  tabs: Tab[]
  type?: 'line' | 'card'
}

withDefaults(defineProps<Props>(), {
  type: 'line',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()

function handleTabClick(tab: Tab) {
  if (tab.disabled) return
  emit('update:modelValue', tab.key)
  emit('change', tab.key)
}
</script>

<template>
  <div class="cyp-tabs" :class="[`cyp-tabs--${type}`]">
    <div class="cyp-tabs__nav">
      <div
        v-for="tab in tabs"
        :key="tab.key"
        class="cyp-tabs__tab"
        :class="{
          'cyp-tabs__tab--active': modelValue === tab.key,
          'cyp-tabs__tab--disabled': tab.disabled,
        }"
        @click="handleTabClick(tab)"
      >
        {{ tab.title }}
      </div>
      <div class="cyp-tabs__ink" :style="{ left: `${tabs.findIndex(t => t.key === modelValue) * 100}px` }" />
    </div>
    <div class="cyp-tabs__content">
      <slot />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.cyp-tabs {
  &__nav {
    display: flex;
    position: relative;
    border-bottom: 1px solid #e2e8f0;
    margin-bottom: 24px;
  }

  &__tab {
    padding: 12px 20px;
    font-size: 14px;
    color: #64748b;
    cursor: pointer;
    transition: all 0.2s ease;
    position: relative;
    user-select: none;

    &:hover:not(&--disabled) {
      color: #1e293b;
    }

    &--active {
      color: #6366f1;
      font-weight: 500;
    }

    &--disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  &__ink {
    position: absolute;
    bottom: -1px;
    height: 2px;
    background: #6366f1;
    transition: left 0.2s ease;
  }

  &__content {
    min-height: 100px;
  }

  &--card {
    .cyp-tabs__nav {
      border-bottom: none;
      gap: 8px;
      margin-bottom: 16px;
    }

    .cyp-tabs__tab {
      background: #f1f5f9;
      border-radius: 8px 8px 0 0;
      border: 1px solid #e2e8f0;
      border-bottom: none;

      &--active {
        background: white;
        color: #1e293b;
      }
    }

    .cyp-tabs__ink {
      display: none;
    }
  }
}
</style>

