<script setup lang="ts">
import { inject, computed, ref, watch } from 'vue'
import type { FormItemInstance, FormRule, FormInstance } from './types'

interface Props {
  prop?: string
  label?: string
  required?: boolean
  error?: string
}

const props = withDefaults(defineProps<Props>(), {
  required: false,
})

const formInstance = inject<FormInstance>('formInstance')
const innerError = ref(props.error)

const isRequired = computed(() => {
  if (props.required) return true
  if (!props.prop || !formInstance?.rules) return false
  const rules = formInstance.rules[props.prop]
  return rules?.some((rule: FormRule) => rule.required)
})

const labelStyle = computed(() => ({
  width: formInstance?.model?.labelWidth || '100px',
}))

watch(() => props.error, (val) => {
  innerError.value = val
})

async function validate(): Promise<boolean> {
  if (!props.prop || !formInstance?.model) return true

  const value = formInstance.model[props.prop]
  const rules = formInstance.rules?.[props.prop]

  if (!rules) return true

  for (const rule of rules) {
    if (rule.required && (value === undefined || value === '' || value === null)) {
      innerError.value = rule.message || '此项为必填项'
      return false
    }

    if (rule.min && String(value).length < rule.min) {
      innerError.value = rule.message || `长度不能少于 ${rule.min} 个字符`
      return false
    }

    if (rule.max && String(value).length > rule.max) {
      innerError.value = rule.message || `长度不能超过 ${rule.max} 个字符`
      return false
    }

    if (rule.pattern && !rule.pattern.test(value)) {
      innerError.value = rule.message || '格式不正确'
      return false
    }
  }

  innerError.value = undefined
  return true
}

function resetField() {
  if (props.prop && formInstance?.model) {
    formInstance.model[props.prop] = undefined
    innerError.value = undefined
  }
}

function clearValidate() {
  innerError.value = undefined
}

defineExpose<FormItemInstance>({
  validate,
  resetField,
  clearValidate,
})
</script>

<template>
  <div class="cyp-form-item" :class="{ 'cyp-form-item--error': innerError }">
    <label v-if="label" class="cyp-form-item__label" :style="labelStyle">
      <span v-if="isRequired" class="cyp-form-item__required">*</span>
      {{ label }}
    </label>
    <div class="cyp-form-item__content">
      <slot />
      <span v-if="innerError" class="cyp-form-item__error-message">
        {{ innerError }}
      </span>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.cyp-form-item {
  display: flex;
  flex-direction: column;
  gap: 6px;

  &__label {
    font-size: 14px;
    font-weight: 500;
    color: #374151;
    display: flex;
    align-items: center;
  }

  &__required {
    color: #ef4444;
    margin-right: 4px;
  }

  &__content {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__error-message {
    font-size: 12px;
    color: #ef4444;
  }

  &--error {
    .cyp-form-item__label {
      color: #ef4444;
    }
  }
}
</style>

