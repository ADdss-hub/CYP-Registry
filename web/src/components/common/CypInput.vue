<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  modelValue: string | number
  type?: 'text' | 'password' | 'email' | 'number'
  placeholder?: string
  disabled?: boolean
  error?: string
  size?: 'small' | 'medium' | 'large'
  autocomplete?: string
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  placeholder: '',
  disabled: false,
  size: 'medium',
  autocomplete: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

const inputClass = computed(() => [
  'cyp-input',
  `cyp-input--${props.size}`,
  {
    'cyp-input--error': props.error,
    'cyp-input--disabled': props.disabled,
  },
])

function handleInput(event: Event) {
  const target = event.target as HTMLInputElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <div class="cyp-input-wrapper">
    <input
      :type="type"
      :class="inputClass"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :autocomplete="autocomplete"
      @input="handleInput"
    />
    <span v-if="error" class="cyp-input__error">{{ error }}</span>
  </div>
</template>

<style lang="scss" scoped>
@use '@/assets/styles/variables.scss' as *;

.cyp-input-wrapper {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.cyp-input {
  width: 100%;
  border: 1px solid $light-border;
  border-radius: 8px;
  background: $light-bg-secondary;
  color: $light-text;
  transition: all 0.2s ease;

  &--small {
    padding: 6px 10px;
    font-size: 12px;
  }

  &--medium {
    padding: 10px 14px;
    font-size: 14px;
  }

  &--large {
    padding: 14px 18px;
    font-size: 16px;
  }

  &:focus {
    outline: none;
    border-color: $primary-color;
    box-shadow: 0 0 0 3px rgba($primary-color, 0.1);
  }

  &--error {
    border-color: $danger-color;

    &:focus {
      box-shadow: 0 0 0 3px rgba($danger-color, 0.1);
    }
  }

  &--disabled {
    background: $light-bg-tertiary;
    cursor: not-allowed;
  }

  &::placeholder {
    color: $light-text-secondary;
  }

  &__error {
    font-size: 12px;
    color: $danger-color;
  }
}
</style>

