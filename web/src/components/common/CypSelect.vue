<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'

interface Option {
  value: string | number
  label: string
  disabled?: boolean
}

interface Props {
  modelValue: string | number
  options: Option[]
  placeholder?: string
  disabled?: boolean
  size?: 'small' | 'medium' | 'large'
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '请选择',
  disabled: false,
  size: 'medium',
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
  change: [value: string | number]
}>()

const isOpen = ref(false)
const selectRef = ref<HTMLElement>()

const selectedLabel = computed(() => {
  const option = props.options.find((opt) => opt.value === props.modelValue)
  return option?.label || props.placeholder
})

function handleSelect(option: Option) {
  if (option.disabled) return
  emit('update:modelValue', option.value)
  emit('change', option.value)
  isOpen.value = false
}

function handleClickOutside(e: Event) {
  if (selectRef.value && !selectRef.value.contains(e.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="selectRef" class="cyp-select" :class="[`cyp-select--${size}`]">
    <div
      class="cyp-select__trigger"
      :class="{ 'cyp-select__trigger--open': isOpen, 'cyp-select__trigger--disabled': disabled }"
      @click="!disabled && (isOpen = !isOpen)"
    >
      <span class="cyp-select__value" :class="{ 'cyp-select__value--placeholder': !modelValue }">
        {{ selectedLabel }}
      </span>
      <svg class="cyp-select__arrow" viewBox="0 0 24 24" width="16" height="16">
        <path
          fill="currentColor"
          d="M7 10l5 5 5-5z"
        />
      </svg>
    </div>

    <Transition name="select-drop">
      <div v-if="isOpen" class="cyp-select__dropdown">
        <div
          v-for="option in options"
          :key="option.value"
          class="cyp-select__option"
          :class="{
            'cyp-select__option--selected': modelValue === option.value,
            'cyp-select__option--disabled': option.disabled,
          }"
          @click="handleSelect(option)"
        >
          {{ option.label }}
        </div>
      </div>
    </Transition>
  </div>
</template>

<style lang="scss" scoped>
.cyp-select {
  position: relative;
  width: 100%;

  &--small {
    .cyp-select__trigger {
      padding: 6px 10px;
      font-size: 12px;
    }
  }

  &--medium {
    .cyp-select__trigger {
      padding: 10px 14px;
      font-size: 14px;
    }
  }

  &--large {
    .cyp-select__trigger {
      padding: 14px 18px;
      font-size: 16px;
    }
  }

  &__trigger {
    display: flex;
    align-items: center;
    justify-content: space-between;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: white;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover:not(&--disabled) {
      border-color: #cbd5e1;
    }

    &--open {
      border-color: #6366f1;
      box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);

      .cyp-select__arrow {
        transform: rotate(180deg);
      }
    }

    &--disabled {
      background: #f1f5f9;
      cursor: not-allowed;
    }
  }

  &__value {
    flex: 1;
    color: #1e293b;

    &--placeholder {
      color: #94a3b8;
    }
  }

  &__arrow {
    color: #64748b;
    transition: transform 0.2s ease;
  }

  &__dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: white;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    max-height: 200px;
    overflow-y: auto;
    z-index: 100;
  }

  &__option {
    padding: 10px 14px;
    cursor: pointer;
    transition: background 0.15s ease;

    &:hover:not(&--selected) {
      background: #f8fafc;
    }

    &--selected {
      background: #eef2ff;
      color: #6366f1;
      font-weight: 500;
    }

    &--disabled {
      color: #cbd5e1;
      cursor: not-allowed;
    }
  }
}

.select-drop-enter-active,
.select-drop-leave-active {
  transition: all 0.2s ease;
}

.select-drop-enter-from,
.select-drop-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>

