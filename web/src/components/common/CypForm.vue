<script setup lang="ts">
import { provide, reactive } from "vue";
import type { FormInstance, FormRules, ValidateCallback } from "./types";

interface Props {
  model: Record<string, any>;
  rules?: FormRules;
  labelWidth?: string;
  labelPosition?: "top" | "left" | "right";
}

const props = withDefaults(defineProps<Props>(), {
  // 如果未传入校验规则，则使用空对象，避免访问 undefined
  rules: () => ({}) as FormRules,
  labelWidth: "100px",
  labelPosition: "top",
});

const emit = defineEmits<{
  submit: [];
}>();

// 提供表单实例给子组件
const formInstance = reactive({
  model: props.model,
  rules: props.rules,
  validateCallback: null as ((valid: boolean) => void) | null,
}) as FormInstance;

provide("formInstance", formInstance);

function validate(callback: ValidateCallback) {
  formInstance.validateCallback = callback;
  // 触发所有 FormItem 的验证
  // 这里简化处理，实际实现需要更复杂的验证逻辑
  callback(true);
}

function resetFields() {
  // 重置字段
  Object.keys(props.model).forEach((key) => {
    // 避免直接修改 props，使用提供给子组件的 formInstance.model
    formInstance.model[key] = undefined;
  });
}

function clearValidate() {
  // 清除验证状态
}

function handleSubmit() {
  validate((valid) => {
    if (valid) {
      emit("submit");
    }
  });
}

defineExpose({
  validate,
  resetFields,
  clearValidate,
});
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
