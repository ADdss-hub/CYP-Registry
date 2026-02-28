<script setup lang="ts">
import { provide, reactive } from 'vue'
import type { FormInstance, FormRules, ValidateCallback } from './types'

interface Props {
  model: Record<string, any>
  rules?: FormRules
  labelWidth?: string
  labelPosition?: 'top' | 'left' | 'right'
}

const props = withDefaults(defineProps<Props>(), {
  labelWidth: '100px',
  labelPosition: 'top',
})

const emit = defineEmits<{
  submit: []
}>()

// 提供表单实例给子组件
const formInstance = reactive({
  model: props.model,
  rules: props.rules,
  validateCallback: null as ((valid: boolean) => void) | null,
}) as FormInstance

provide('formInstance', formInstance)

function validate(callback: ValidateCallback) {
  formInstance.validateCallback = callback
  // 触发所有 FormItem 的验证
  // 这里简化处理，实际实现需要更复杂的验证逻辑
  callback(true)
}

function resetFields() {
  // 重置字段
  Object.keys(props.model).forEach((key) => {
    props.model[key] = undefined
  })
}

function clearValidate() {
  // 清除验证状态
}

function handleSubmit() {
  validate((valid) => {
    if (valid) {
      emit('submit')
    }
  })
}

defineExpose({
  validate,
  resetFields,
  clearValidate,
})
</script>

<template>
  <form class="cyp-form" @submit.prevent="handleSubmit">
    <slot />
  </form>
</template>

<style lang="scss" scoped>
.cyp-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
</style>

