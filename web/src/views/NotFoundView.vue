<script setup lang="ts">
import { useRouter } from 'vue-router'
import CypButton from '@/components/common/CypButton.vue'

const router = useRouter()

function goHome() {
  router.push('/')
}
</script>

<template>
  <div class="not-found-page">
    <div class="not-found-content">
      <div class="error-code">404</div>
      <h1>页面未找到</h1>
      <p>抱歉，您访问的页面不存在或已被移除。</p>
      <div class="actions">
        <CypButton type="primary" @click="goHome">返回首页</CypButton>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.not-found-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 24px;
}

.not-found-content {
  text-align: center;
  color: white;
}

.error-code {
  font-size: 120px;
  font-weight: 700;
  line-height: 1;
  margin-bottom: 16px;
  opacity: 0.9;
}

h1 {
  font-size: 32px;
  font-weight: 600;
  margin: 0 0 12px;
}

p {
  font-size: 16px;
  opacity: 0.9;
  margin: 0 0 32px;
}

.actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}
</style>

