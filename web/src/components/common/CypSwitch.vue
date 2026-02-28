<script setup lang="ts">
import { computed } from "vue";

interface Props {
  modelValue: boolean;
  disabled?: boolean;
  size?: "small" | "medium" | "large";
  activeText?: string;
  inactiveText?: string;
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  size: "medium",
  activeText: "开启",
  inactiveText: "关闭",
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  change: [value: boolean];
}>();

const switchClass = computed(() => [
  "cyp-switch",
  `cyp-switch--${props.size}`,
  {
    "cyp-switch--checked": props.modelValue,
    "cyp-switch--disabled": props.disabled,
  },
]);

function handleClick() {
  if (props.disabled) return;
  const newValue = !props.modelValue;
  emit("update:modelValue", newValue);
  emit("change", newValue);
}
</script>

<template>
  <button
    :class="switchClass"
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :disabled="disabled"
    @click="handleClick"
  >
    <span class="cyp-switch__core">
      <span class="cyp-switch__knob" />
    </span>
    <span
      v-if="activeText || inactiveText"
      class="cyp-switch__text"
    >
      {{ modelValue ? activeText : inactiveText }}
    </span>
  </button>
</template>

<style lang="scss" scoped>
.cyp-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;

  &--small {
    .cyp-switch__core {
      width: 32px;
      height: 20px;
    }

    .cyp-switch__knob {
      width: 16px;
      height: 16px;
    }

    .cyp-switch--checked .cyp-switch__knob {
      transform: translateX(12px);
    }
  }

  &--medium {
    .cyp-switch__core {
      width: 44px;
      height: 24px;
    }

    .cyp-switch__knob {
      width: 20px;
      height: 20px;
    }

    .cyp-switch--checked .cyp-switch__knob {
      transform: translateX(20px);
    }
  }

  &--large {
    .cyp-switch__core {
      width: 56px;
      height: 28px;
    }

    .cyp-switch__knob {
      width: 24px;
      height: 24px;
    }

    .cyp-switch--checked .cyp-switch__knob {
      transform: translateX(28px);
    }
  }

  &--checked {
    .cyp-switch__core {
      background: #6366f1;
    }
  }

  &--disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  &__core {
    position: relative;
    display: inline-block;
    border-radius: 999px;
    background: #e2e8f0;
    transition: background 0.2s ease;
  }

  &__knob {
    position: absolute;
    top: 2px;
    left: 2px;
    background: white;
    border-radius: 50%;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    transition: transform 0.2s ease;
  }

  &__text {
    font-size: 14px;
    color: #64748b;
    user-select: none;
  }
}
</style>
