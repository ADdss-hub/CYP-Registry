<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  modelValue: boolean | string | number | any[]
  value?: string | number
  disabled?: boolean
  indeterminate?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  indeterminate: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean | string | number | any[]]
  change: [value: boolean | string | number | any[]]
}>()

const isChecked = computed(() => {
  if (Array.isArray(props.modelValue)) {
    return props.modelValue.includes(props.value)
  }
  return Boolean(props.modelValue)
})

function handleChange() {
  if (props.disabled) return

  let newValue: boolean | string | number | any[]

  if (Array.isArray(props.modelValue)) {
    if (isChecked.value) {
      newValue = props.modelValue.filter((v) => v !== props.value)
    } else {
      newValue = [...props.modelValue, props.value]
    }
  } else {
    newValue = !props.modelValue
  }

  emit('update:modelValue', newValue)
  emit('change', newValue)
}
</script>

<template>
  <label class="cyp-checkbox" :class="{ 'cyp-checkbox--disabled': disabled }">
    <input
      type="checkbox"
      :checked="isChecked"
      :disabled="disabled"
      class="cyp-checkbox__input"
      @change="handleChange"
    />
    <span class="cyp-checkbox__box">
      <svg v-if="indeterminate" viewBox="0 0 24 24" width="14" height="14">
        <path
          fill="currentColor"
          d="M19 13H5v-2h14v2z"
        />
      </svg>
      <svg v-else-if="isChecked" viewBox="0 0 24 24" width="14" height="14">
        <path
          fill="currentColor"
          d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
        />
      </svg>
    </span>
    <span v-if="$slots.default" class="cyp-checkbox__label">
      <slot />
    </span>
  </label>
</template>

<style lang="scss" scoped>
.cyp-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;

  &--disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  &__input {
    position: absolute;
    opacity: 0;
    width: 0;
    height: 0;
  }

  &__box {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border: 2px solid #cbd5e1;
    border-radius: 4px;
    background: white;
    transition: all 0.2s ease;
    color: white;

    .cyp-checkbox__input:checked + & {
      background: #6366f1;
      border-color: #6366f1;
    }

    .cyp-checkbox__input:indeterminate + & {
      background: #6366f1;
      border-color: #6366f1;
    }

    .cyp-checkbox:hover:not(&--disabled) & {
      border-color: #6366f1;
    }
  }

  &__label {
    font-size: 14px;
    color: #374151;
  }
}
</style>

