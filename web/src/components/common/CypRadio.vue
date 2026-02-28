<script setup lang="ts">
interface Option {
  value: string | number;
  label: string;
  disabled?: boolean;
}

interface Props {
  modelValue: string | number;
  options: Option[];
  disabled?: boolean;
  size?: "small" | "medium" | "large";
}

withDefaults(defineProps<Props>(), {
  disabled: false,
  size: "medium",
});

const emit = defineEmits<{
  "update:modelValue": [value: string | number];
  change: [value: string | number];
}>();

function handleChange(value: string | number, option: Option) {
  if (option.disabled) return;
  emit("update:modelValue", value);
  emit("change", value);
}
</script>

<template>
  <div
    class="cyp-radio-group"
    :class="[`cyp-radio-group--${size}`]"
  >
    <label
      v-for="option in options"
      :key="option.value"
      class="cyp-radio"
      :class="{
        'cyp-radio--checked': modelValue === option.value,
        'cyp-radio--disabled': disabled || option.disabled,
      }"
    >
      <input
        type="radio"
        :value="option.value"
        :checked="modelValue === option.value"
        :disabled="disabled || option.disabled"
        class="cyp-radio__input"
        @change="handleChange(option.value, option)"
      >
      <span class="cyp-radio__dot" />
      <span class="cyp-radio__label">{{ option.label }}</span>
    </label>
  </div>
</template>

<style lang="scss" scoped>
.cyp-radio-group {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;

  &--small {
    .cyp-radio {
      padding: 6px 12px;
      font-size: 12px;
    }
  }

  &--medium {
    .cyp-radio {
      padding: 10px 16px;
      font-size: 14px;
    }
  }

  &--large {
    .cyp-radio {
      padding: 14px 20px;
      font-size: 16px;
    }
  }
}

.cyp-radio {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover:not(&--disabled) {
    border-color: #6366f1;
  }

  &--checked {
    background: #eef2ff;
    border-color: #6366f1;

    .cyp-radio__dot {
      background: #6366f1;
      box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.2);

      &::after {
        opacity: 1;
      }
    }

    .cyp-radio__label {
      color: #6366f1;
    }
  }

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

  &__dot {
    width: 16px;
    height: 16px;
    border: 2px solid #cbd5e1;
    border-radius: 50%;
    background: white;
    position: relative;
    transition: all 0.2s ease;

    &::after {
      content: "";
      position: absolute;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: white;
      opacity: 0;
      transition: opacity 0.2s ease;
    }
  }

  &__label {
    font-size: 14px;
    color: #374151;
  }
}
</style>
