<script setup lang="ts">
import { computed } from "vue";

type TagType = "default" | "primary" | "success" | "warning" | "danger";

interface Props {
  type?: TagType;
  size?: "small" | "medium";
  closable?: boolean;
  round?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  type: "default",
  size: "medium",
  closable: false,
  round: true,
});

const emit = defineEmits<{
  close: [];
  click: [];
}>();

const tagClass = computed(() => [
  "cyp-tag",
  `cyp-tag--${props.type}`,
  `cyp-tag--${props.size}`,
  {
    "cyp-tag--round": props.round,
  },
]);

function handleClose(e: Event) {
  e.stopPropagation();
  emit("close");
}

function handleClick() {
  emit("click");
}
</script>

<template>
  <span :class="tagClass" @click="handleClick">
    <slot />
    <button
      v-if="closable"
      class="cyp-tag__close"
      type="button"
      @click="handleClose"
    >
      <svg viewBox="0 0 24 24" width="12" height="12">
        <path
          fill="currentColor"
          d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"
        />
      </svg>
    </button>
  </span>
</template>

<style lang="scss" scoped>
.cyp-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border-radius: 6px;
  font-weight: 500;
  cursor: default;
  transition: all 0.2s ease;

  &--round {
    border-radius: 999px;
  }

  &--small {
    padding: 2px 8px;
    font-size: 12px;
  }

  &--medium {
    padding: 4px 12px;
    font-size: 13px;
  }

  &--default {
    background: #f1f5f9;
    color: #475569;
  }

  &--primary {
    background: #eef2ff;
    color: #6366f1;
  }

  &--success {
    background: #f0fdf4;
    color: #22c55e;
  }

  &--warning {
    background: #fffbeb;
    color: #f59e0b;
  }

  &--danger {
    background: #fef2f2;
    color: #ef4444;
  }

  &__close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    padding: 0;
    margin-left: 2px;
    cursor: pointer;
    color: inherit;
    opacity: 0.7;
    transition: opacity 0.2s ease;

    &:hover {
      opacity: 1;
    }
  }
}
</style>
